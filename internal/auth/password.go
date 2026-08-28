package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/crypto/argon2"
)

const argon2Version = 19

var usernamePattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{2,63}$`)

type PasswordParams struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

type PasswordHasher struct {
	params PasswordParams
}

func DefaultPasswordParams() PasswordParams {
	return PasswordParams{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
}

func NewPasswordHasher(params PasswordParams) PasswordHasher {
	return PasswordHasher{params: params}
}

func (hasher PasswordHasher) Hash(password, username string) (string, error) {
	if !usernamePattern.MatchString(username) {
		return "", errors.New("username must match ^[a-z][a-z0-9_.-]{2,63}$")
	}
	if len(password) < 14 || len(password) > 1024 {
		return "", errors.New("password must contain between 14 and 1024 bytes")
	}
	if strings.EqualFold(password, username) {
		return "", errors.New("password must not equal username")
	}
	if err := validateParams(hasher.params); err != nil {
		return "", err
	}
	salt := make([]byte, hasher.params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, hasher.params.Iterations, hasher.params.Memory, hasher.params.Parallelism, hasher.params.KeyLength)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2Version,
		hasher.params.Memory,
		hasher.params.Iterations,
		hasher.params.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

func (hasher PasswordHasher) Verify(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return false, errors.New("invalid password hash format")
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2Version {
		return false, errors.New("unsupported password hash version")
	}
	var params PasswordParams
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &params.Memory, &params.Iterations, &params.Parallelism); err != nil {
		return false, errors.New("invalid password hash parameters")
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) < 8 || len(salt) > 64 {
		return false, errors.New("invalid password hash salt")
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(expected) < 16 || len(expected) > 64 {
		return false, errors.New("invalid password hash key")
	}
	params.SaltLength = uint32(len(salt))
	params.KeyLength = uint32(len(expected))
	if err := validateParams(params); err != nil {
		return false, err
	}
	actual := argon2.IDKey([]byte(password), salt, params.Iterations, params.Memory, params.Parallelism, params.KeyLength)
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}

func validateParams(params PasswordParams) error {
	if params.Memory < 8*1024 || params.Memory > 256*1024 {
		return errors.New("password hash memory is outside allowed range")
	}
	if params.Iterations < 1 || params.Iterations > 10 {
		return errors.New("password hash iterations are outside allowed range")
	}
	if params.Parallelism < 1 || params.Parallelism > 16 {
		return errors.New("password hash parallelism is outside allowed range")
	}
	if params.SaltLength < 8 || params.SaltLength > 64 || params.KeyLength < 16 || params.KeyLength > 64 {
		return errors.New("password hash lengths are outside allowed range")
	}
	return nil
}
