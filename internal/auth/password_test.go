package auth

import (
	"strings"
	"testing"
)

func TestPasswordHashVerifiesWithoutContainingPlaintext(t *testing.T) {
	hasher := NewPasswordHasher(DefaultPasswordParams())
	password := "very long fixture password 2026"
	hash, err := hasher.Hash(password, "alice")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if strings.Contains(hash, password) {
		t.Fatal("password hash contains plaintext")
	}
	if ok, err := hasher.Verify(password, hash); err != nil || !ok {
		t.Fatalf("Verify(correct) = %v, %v", ok, err)
	}
	if ok, err := hasher.Verify("wrong fixture password", hash); err != nil || ok {
		t.Fatalf("Verify(wrong) = %v, %v", ok, err)
	}
}

func TestPasswordAndUsernameValidationRejectsBreakingInputs(t *testing.T) {
	hasher := NewPasswordHasher(DefaultPasswordParams())
	tests := []struct {
		username string
		password string
	}{
		{username: "Ａlice", password: "very long fixture password 2026"},
		{username: "../alice", password: "very long fixture password 2026"},
		{username: "alice", password: "alice"},
		{username: "alice", password: strings.Repeat("x", 1025)},
	}
	for _, test := range tests {
		if _, err := hasher.Hash(test.password, test.username); err == nil {
			t.Fatalf("Hash(%q, %q) error = nil", test.password, test.username)
		}
	}
}

func TestVerifyRejectsMalformedAndResourceExhaustingHashes(t *testing.T) {
	hasher := NewPasswordHasher(DefaultPasswordParams())
	inputs := []string{
		"not-a-password-hash",
		"$argon2id$v=19$m=4294967295,t=3,p=2$c2FsdA$aGFzaA",
		"$argon2id$v=18$m=65536,t=3,p=2$c2FsdA$aGFzaA",
	}
	for _, input := range inputs {
		if _, err := hasher.Verify("fixture password", input); err == nil {
			t.Fatalf("Verify(%q) error = nil", input)
		}
	}
}
