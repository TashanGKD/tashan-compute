package identity

import (
	"errors"
	"time"
)

var ErrAccountNotFound = errors.New("account not found")
var ErrPrincipalInvalid = errors.New("principal is no longer valid")

type Account struct {
	ID                 string
	Username           string
	PasswordHash       string
	PlatformAdmin      bool
	PasswordVersion    int
	MustChangePassword bool
	DisabledAt         *time.Time
}

type Actor struct {
	AccountID     string
	DeviceID      string
	PlatformAdmin bool
}

type Principal struct {
	Account   Account
	DeviceID  string
	SessionID string
}
