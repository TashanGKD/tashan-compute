package config

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

type Source interface {
	Get(string) string
}

type MapSource map[string]string

func (source MapSource) Get(key string) string {
	return source[key]
}

type Config struct {
	Environment          string
	ListenAddress        string
	PublicURL            string
	DatabaseURL          string
	RedisURL             string
	AccessPrivateKeyFile string
	AccessPublicKeyFile  string
	RefreshPepperFile    string
	TrustedProxyCIDRs    []string
	AllowedOrigins       []string
}

func Load(source Source) (Config, error) {
	cfg := Config{
		Environment:          valueOr(source.Get("TCOMPUTE_ENV"), "development"),
		ListenAddress:        valueOr(source.Get("TCOMPUTE_LISTEN_ADDRESS"), "127.0.0.1:8180"),
		PublicURL:            valueOr(source.Get("TCOMPUTE_PUBLIC_URL"), "http://127.0.0.1:8180"),
		DatabaseURL:          source.Get("TCOMPUTE_DATABASE_URL"),
		RedisURL:             source.Get("TCOMPUTE_REDIS_URL"),
		AccessPrivateKeyFile: source.Get("TCOMPUTE_ACCESS_PRIVATE_KEY_FILE"),
		AccessPublicKeyFile:  source.Get("TCOMPUTE_ACCESS_PUBLIC_KEY_FILE"),
		RefreshPepperFile:    source.Get("TCOMPUTE_REFRESH_PEPPER_FILE"),
		TrustedProxyCIDRs:    splitList(source.Get("TCOMPUTE_TRUSTED_PROXY_CIDRS")),
		AllowedOrigins:       splitList(source.Get("TCOMPUTE_ALLOWED_ORIGINS")),
	}

	if cfg.Environment != "production" {
		return cfg, nil
	}
	publicURL, err := url.Parse(cfg.PublicURL)
	if err != nil || publicURL.Scheme != "https" || publicURL.Host == "" {
		return Config{}, errors.New("production public URL must use https")
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("TCOMPUTE_DATABASE_URL is required")
	}
	if isLoopbackURL(cfg.DatabaseURL) {
		return Config{}, errors.New("production database URL cannot use loopback")
	}
	if cfg.RedisURL == "" {
		return Config{}, errors.New("TCOMPUTE_REDIS_URL is required")
	}
	if isLoopbackURL(cfg.RedisURL) {
		return Config{}, errors.New("production redis URL cannot use loopback")
	}
	for _, required := range []struct {
		name  string
		value string
	}{
		{"TCOMPUTE_ACCESS_PRIVATE_KEY_FILE", cfg.AccessPrivateKeyFile},
		{"TCOMPUTE_ACCESS_PUBLIC_KEY_FILE", cfg.AccessPublicKeyFile},
		{"TCOMPUTE_REFRESH_PEPPER_FILE", cfg.RefreshPepperFile},
	} {
		if required.value == "" {
			return Config{}, errors.New(required.name + " is required")
		}
	}
	if len(cfg.AllowedOrigins) == 0 {
		return Config{}, errors.New("TCOMPUTE_ALLOWED_ORIGINS is required")
	}
	for _, origin := range cfg.AllowedOrigins {
		if origin == "*" {
			return Config{}, errors.New("production allowed origins cannot contain wildcard")
		}
	}
	return cfg, nil
}

func valueOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func splitList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func isLoopbackURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.TrimSpace(strings.ToLower(parsed.Hostname()))
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
