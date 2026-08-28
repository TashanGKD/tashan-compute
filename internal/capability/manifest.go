package capability

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

type AuthClass string

const (
	AuthPublic        AuthClass = "public"
	AuthInitial       AuthClass = "initial_password"
	AuthUser          AuthClass = "user"
	AuthRefreshToken  AuthClass = "refresh_token"
	AuthPlatformAdmin AuthClass = "platform_admin"
	AuthOrgAdmin      AuthClass = "org_admin"
	AuthScopedAuditor AuthClass = "scoped_auditor"
)

type SideEffect string

const (
	SideEffectNone    SideEffect = "none"
	SideEffectSession SideEffect = "session"
	SideEffectWrite   SideEffect = "write"
	SideEffectRevoke  SideEffect = "revoke"
)

var ErrDuplicateCapability = errors.New("duplicate capability")

type Capability struct {
	ID         string     `json:"id"`
	Version    int        `json:"version"`
	CLI        string     `json:"cli"`
	Auth       AuthClass  `json:"auth"`
	SideEffect SideEffect `json:"side_effect"`
}

func Load(reader io.Reader) ([]Capability, error) {
	decoder := json.NewDecoder(io.LimitReader(reader, 1<<20))
	decoder.DisallowUnknownFields()

	var capabilities []Capability
	if err := decoder.Decode(&capabilities); err != nil {
		return nil, fmt.Errorf("decode capability manifest: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("decode capability manifest: trailing JSON value")
		}
		return nil, fmt.Errorf("decode capability manifest: %w", err)
	}
	if err := Validate(capabilities); err != nil {
		return nil, err
	}
	return capabilities, nil
}

func Validate(capabilities []Capability) error {
	seen := make(map[string]struct{}, len(capabilities))
	for _, item := range capabilities {
		if item.ID == "" {
			return errors.New("capability ID is required")
		}
		if _, exists := seen[item.ID]; exists {
			return fmt.Errorf("%w: %s", ErrDuplicateCapability, item.ID)
		}
		seen[item.ID] = struct{}{}
		if item.Version < 1 {
			return fmt.Errorf("capability version must be positive: %s", item.ID)
		}
		if item.CLI == "" {
			return fmt.Errorf("capability CLI binding is required: %s", item.ID)
		}
		if !validAuth(item.Auth) {
			return fmt.Errorf("unknown capability authorization %q: %s", item.Auth, item.ID)
		}
		if !validSideEffect(item.SideEffect) {
			return fmt.Errorf("unknown capability side effect %q: %s", item.SideEffect, item.ID)
		}
	}
	return nil
}

func CheckBindings(capabilities []Capability, bindings map[string]string) error {
	for _, item := range capabilities {
		command, ok := bindings[item.ID]
		if !ok {
			return fmt.Errorf("missing CLI binding: %s", item.ID)
		}
		if command != item.CLI {
			return fmt.Errorf("CLI binding mismatch: %s: manifest=%q binding=%q", item.ID, item.CLI, command)
		}
	}
	for id := range bindings {
		found := false
		for _, item := range capabilities {
			if item.ID == id {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unknown CLI capability: %s", id)
		}
	}
	return nil
}

func validAuth(value AuthClass) bool {
	switch value {
	case AuthPublic, AuthInitial, AuthUser, AuthRefreshToken, AuthPlatformAdmin, AuthOrgAdmin, AuthScopedAuditor:
		return true
	default:
		return false
	}
}

func validSideEffect(value SideEffect) bool {
	switch value {
	case SideEffectNone, SideEffectSession, SideEffectWrite, SideEffectRevoke:
		return true
	default:
		return false
	}
}
