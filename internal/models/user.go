package models

import "time"

type User struct {
	ID               int64
	Username         string
	PasswordHash     string
	TOTPSecret       *string
	MFAEnabled       bool
	FailedAttempts   int
	LockedUntil      *time.Time
	RegistrationDate time.Time
	LastLoginAt      *time.Time
}
