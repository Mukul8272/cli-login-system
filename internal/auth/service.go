package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"

	"github.com/example/containerized-cli-login/internal/config"
	"github.com/example/containerized-cli-login/internal/models"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrLocked             = errors.New("account is temporarily locked")
)

type Service struct {
	db  *sql.DB
	cfg config.Config
}

func NewService(db *sql.DB, cfg config.Config) *Service {
	return &Service{
		db:  db,
		cfg: cfg}
}

func (s *Service) Register(username, password string) error {
	username = strings.TrimSpace(username)
	if len(username) < 3 || len(username) > 50 {
		return fmt.Errorf("username must be 3-50 characters")
	}
	if strings.ContainsAny(username, " \t\r\n") {
		return fmt.Errorf("username cannot contain spaces")
	}
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	_, err = s.db.Exec(`INSERT INTO users (username, password_hash) VALUES (?, ?)`, username, string(hash))
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") {
			return fmt.Errorf("username already exists")
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (s *Service) Login(username, password, code string) (*models.User, string, time.Time, error) {
	user, err := s.getUser(username)
	if err != nil {
		return nil, "", time.Time{}, ErrInvalidCredentials
	}

	now := time.Now()

	if user.LockedUntil != nil && user.LockedUntil.After(now) {
		return nil, "", time.Time{}, ErrLocked
	}

	if user.LockedUntil != nil && !user.LockedUntil.After(now) {
		_, _ = s.db.Exec(`UPDATE users SET locked_until = NULL, failed_attempts = 0 WHERE id = ?`, user.ID)
		user.FailedAttempts = 0
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		s.recordFailure(user.ID, user.FailedAttempts)
		return nil, "", time.Time{}, ErrInvalidCredentials
	}

	if user.MFAEnabled {
		if code == "" || user.TOTPSecret == nil || !totp.Validate(code, *user.TOTPSecret) {
			s.recordFailure(user.ID, user.FailedAttempts)
			return nil, "", time.Time{}, fmt.Errorf("invalid 2FA code")
		}
	}

	if _, err := s.db.Exec(`UPDATE users SET failed_attempts = 0, locked_until = NULL, last_login_at = NOW() WHERE id = ?`, user.ID); err != nil {
		return nil, "", time.Time{}, fmt.Errorf("update login state: %w", err)
	}

	expiry := now.Add(s.cfg.SessionTimeout)
	token, err := randomToken()

	if err != nil {
		return nil, "", time.Time{}, err
	}
	hash := sha256.Sum256([]byte(token))
	if _, err := s.db.Exec(`INSERT INTO sessions (user_id, token_hash, expires_at) VALUES (?, ?, ?)`, user.ID, hex.EncodeToString(hash[:]), expiry); err != nil {
		return nil, "", time.Time{}, fmt.Errorf("create session: %w", err)
	}
	updated, _ := s.getUser(username)
	return updated, token, expiry, nil
}

func (s *Service) Enable2FA(userID int64, username string) (string, error) {
	user, err := s.getUserByID(userID)
	if err != nil {
		return "", err
	}
	if user.MFAEnabled {
		return "", fmt.Errorf("2FA is already enabled")
	}
	key, err := totp.Generate(totp.GenerateOpts{Issuer: "CLI Login System", AccountName: username, SecretSize: 20})
	if err != nil {
		return "", fmt.Errorf("generate 2FA secret: %w", err)
	}
	if _, err := s.db.Exec(`UPDATE users SET totp_secret = ?, mfa_enabled = TRUE WHERE id = ?`, key.Secret(), userID); err != nil {
		return "", err
	}
	return key.URL(), nil
}

func (s *Service) Disable2FA(userID int64) error {
	user, err := s.getUserByID(userID)
	if err != nil {
		return err
	}
	if !user.MFAEnabled {
		return fmt.Errorf("2FA is already disabled")
	}
	_, err = s.db.Exec(`UPDATE users SET totp_secret = NULL, mfa_enabled = FALSE WHERE id = ?`, userID)
	return err
}

func (s *Service) Logout(userID int64, token string) error {
	hash := sha256.Sum256([]byte(token))
	_, err := s.db.Exec(`DELETE FROM sessions WHERE user_id = ? AND token_hash = ?`, userID, hex.EncodeToString(hash[:]))
	return err
}

func (s *Service) SessionValid(userID int64, token string) bool {
	hash := sha256.Sum256([]byte(token))
	var exists int
	err := s.db.QueryRow(`SELECT 1 FROM sessions WHERE user_id = ? AND token_hash = ? AND expires_at > NOW()`, userID, hex.EncodeToString(hash[:])).Scan(&exists)
	return err == nil
}

func (s *Service) GetUser(id int64) (*models.User, error) {
	return s.getUserByID(id)
}
func (s *Service) getUser(username string) (*models.User, error) {
	return s.scanUser(s.db.QueryRow(`SELECT id, username, password_hash, totp_secret, mfa_enabled, failed_attempts, locked_until, registration_date, last_login_at FROM users WHERE username = ?`, strings.TrimSpace(username)))
}
func (s *Service) getUserByID(id int64) (*models.User, error) {
	return s.scanUser(s.db.QueryRow(`SELECT id, username, password_hash, totp_secret, mfa_enabled, failed_attempts, locked_until, registration_date, last_login_at FROM users WHERE id = ?`, id))
}
func (s *Service) scanUser(row *sql.Row) (*models.User, error) {
	var u models.User
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.TOTPSecret, &u.MFAEnabled, &u.FailedAttempts, &u.LockedUntil, &u.RegistrationDate, &u.LastLoginAt); err != nil {
		return nil, err
	}
	return &u, nil
}
func (s *Service) recordFailure(id int64, current int) {
	attempts := current + 1
	if attempts >= s.cfg.MaxLoginAttempts {
		_, _ = s.db.Exec(`UPDATE users SET failed_attempts = ?, locked_until = ? WHERE id = ?`, attempts, time.Now().Add(s.cfg.LockoutDuration), id)
		return
	}
	_, _ = s.db.Exec(`UPDATE users SET failed_attempts = ? WHERE id = ?`, attempts, id)
}
func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
