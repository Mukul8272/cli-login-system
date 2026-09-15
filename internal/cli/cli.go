package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/chzyer/readline"

	"github.com/example/containerized-cli-login/internal/auth"
	"github.com/example/containerized-cli-login/internal/models"
)

var errExit = errors.New("exit requested")

type CLI struct {
	service *auth.Service
	current *models.User
	token   string
	expiry  time.Time
}

func New(service *auth.Service) *CLI { return &CLI{service: service} }

func (c *CLI) Run() error {
	fmt.Println("========================================")
	fmt.Println(" Containerized CLI Login System")
	fmt.Println("========================================")
	fmt.Println("Type 'help' to see available commands.")
	fmt.Println()

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "login> ",
		HistoryFile:     "/tmp/cli_login_history",
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
		AutoComplete:    commandCompleter{},
	})
	if err != nil {
		return err
	}
	defer rl.Close()

	for {
		if c.current != nil && time.Now().After(c.expiry) {
			_ = c.logout()
			fmt.Println("Session expired. Please login again.")
		}
		if c.current == nil {
			rl.SetPrompt("login> ")
		} else {
			rl.SetPrompt(c.current.Username + "@cli> ")
		}

		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				continue
			}
			if err == io.EOF {
				return nil
			}
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if err := c.handle(rl, line); err != nil {
			if errors.Is(err, errExit) {
				fmt.Println("Goodbye!")
				return nil
			}
			fmt.Println("Error:", err)
		}
	}
}

func (c *CLI) handle(rl *readline.Instance, line string) error {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil
	}
	cmd := strings.ToLower(parts[0])
	switch cmd {
	case "help":
		c.help()
	case "register":
		return c.register(rl)
	case "login":
		return c.login(rl)
	case "exit", "quit":
		return errExit
	case "whoami":
		return c.whoami()
	case "enable-2fa":
		return c.enable2FA()
	case "disable-2fa":
		return c.disable2FA()
	case "logout":
		return c.logout()
	default:
		fmt.Println("Unknown command. Type 'help' for available commands.")
	}
	return nil
}

func (c *CLI) help() {
	if c.current == nil {
		fmt.Println("Before login:")
		fmt.Println("  register   Create a new user")
		fmt.Println("  login      Login with username/password and 2FA if enabled")
		fmt.Println("  help       Show available commands")
		fmt.Println("  exit       Quit")
		return
	}
	fmt.Println("After login:")
	fmt.Println("  whoami       Show current user details")
	fmt.Println("  enable-2fa   Enable TOTP-based MFA")
	fmt.Println("  disable-2fa  Disable MFA")
	fmt.Println("  logout       End session")
	fmt.Println("  help         Show available commands")
	fmt.Println("  exit         Quit")
}

func (c *CLI) register(rl *readline.Instance) error {
	if c.current != nil {
		return fmt.Errorf("logout first")
	}
	username, err := readLine(rl, "Username: ", false)
	if err != nil {
		return err
	}
	password, err := readLine(rl, "Password: ", true)
	if err != nil {
		return err
	}
	confirm, err := readLine(rl, "Confirm password: ", true)
	if err != nil {
		return err
	}
	if password != confirm {
		return fmt.Errorf("passwords do not match")
	}
	if err := c.service.Register(username, password); err != nil {
		return err
	}
	fmt.Println("Registration successful. You can now login.")
	return nil
}

func (c *CLI) login(rl *readline.Instance) error {
	if c.current != nil {
		return fmt.Errorf("already logged in")
	}
	username, err := readLine(rl, "Username: ", false)
	if err != nil {
		return err
	}
	password, err := readLine(rl, "Password: ", true)
	if err != nil {
		return err
	}
	code, err := readLine(rl, "2FA code (press Enter if not enabled): ", false)
	if err != nil {
		return err
	}

	user, token, expiry, err := c.service.Login(username, password, strings.TrimSpace(code))
	if err != nil {
		return err
	}
	c.current, c.token, c.expiry = user, token, expiry
	fmt.Println("Login successful.")
	c.displayUser(user)
	return nil
}

func (c *CLI) whoami() error {
	if c.current == nil {
		return fmt.Errorf("please login first")
	}
	if !c.service.SessionValid(c.current.ID, c.token) {
		c.current, c.token = nil, ""
		return fmt.Errorf("session expired")
	}
	user, err := c.service.GetUser(c.current.ID)
	if err != nil {
		return err
	}
	c.current = user
	c.displayUser(user)
	return nil
}

func (c *CLI) displayUser(u *models.User) {
	fmt.Println("----------------------------------------")
	fmt.Println("Username:", u.Username)
	fmt.Println("Registration date:", formatDateTime(u.RegistrationDate))
	if u.MFAEnabled {
		fmt.Println("MFA status: enabled")
	} else {
		fmt.Println("MFA status: disabled")
	}
	fmt.Println("Session expiration time:", formatDateTime(c.expiry))
	if u.LastLoginAt != nil {
		fmt.Println("Last login time:", formatDateTime(*u.LastLoginAt))
	} else {
		fmt.Println("Last login time: not available")
	}
	fmt.Println("----------------------------------------")
}

func (c *CLI) enable2FA() error {
	if c.current == nil {
		return fmt.Errorf("please login first")
	}
	url, err := c.service.Enable2FA(c.current.ID, c.current.Username)
	if err != nil {
		return err
	}
	c.current.MFAEnabled = true
	fmt.Println("2FA enabled.")
	fmt.Println("Import this URL into Google Authenticator or another compatible TOTP app:")
	fmt.Println(url)
	fmt.Println("Log out and log in again. A 6-digit TOTP code will then be required.")
	return nil
}

func (c *CLI) disable2FA() error {
	if c.current == nil {
		return fmt.Errorf("please login first")
	}
	if err := c.service.Disable2FA(c.current.ID); err != nil {
		return err
	}
	c.current.MFAEnabled = false
	fmt.Println("2FA disabled.")
	return nil
}

func (c *CLI) logout() error {
	if c.current == nil {
		return nil
	}
	if c.token != "" {
		if err := c.service.Logout(c.current.ID, c.token); err != nil {
			return err
		}
	}
	c.current, c.token = nil, ""
	c.expiry = time.Time{}
	fmt.Println("Logged out successfully.")
	return nil
}

func readLine(rl *readline.Instance, prompt string, secret bool) (string, error) {
	rl.SetPrompt(prompt)
	if secret {
		rl.SetMaskRune('*')
	} else {
		rl.SetMaskRune(0)
	}
	value, err := rl.Readline()
	rl.SetMaskRune(0)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

type commandCompleter struct{}

func (commandCompleter) Do(line []rune, pos int) ([][]rune, int) {
	input := string(line[:pos])
	if strings.ContainsAny(input, " \t") {
		return nil, 0
	}
	commands := []string{"register", "login", "help", "exit", "whoami", "enable-2fa", "disable-2fa", "logout"}
	var matches [][]rune
	for _, command := range commands {
		if strings.HasPrefix(command, input) {
			matches = append(matches, []rune(command))
		}
	}
	return matches, pos
}

func formatDateTime(t time.Time) string {
	return t.Local().Format("02 January 2006, 03:04 PM")
}
