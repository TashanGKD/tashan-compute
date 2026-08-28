package config

import "testing"

func TestProductionRejectsUnsafeOrIncompleteConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		values  MapSource
		message string
	}{
		{
			name: "plain HTTP public URL",
			values: MapSource{
				"TCOMPUTE_ENV":        "production",
				"TCOMPUTE_PUBLIC_URL": "http://compute.example.test",
			},
			message: "production public URL must use https",
		},
		{
			name: "missing database",
			values: MapSource{
				"TCOMPUTE_ENV":        "production",
				"TCOMPUTE_PUBLIC_URL": "https://compute.example.test",
			},
			message: "TCOMPUTE_DATABASE_URL is required",
		},
		{
			name: "wildcard origin",
			values: completeProduction(MapSource{
				"TCOMPUTE_ALLOWED_ORIGINS": "*",
			}),
			message: "production allowed origins cannot contain wildcard",
		},
		{
			name: "loopback database",
			values: completeProduction(MapSource{
				"TCOMPUTE_DATABASE_URL": "postgres://user:pass@127.0.0.1:5432/compute",
			}),
			message: "production database URL cannot use loopback",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Load(test.values)
			if err == nil || err.Error() != test.message {
				t.Fatalf("Load() error = %v, want %q", err, test.message)
			}
		})
	}
}

func TestDevelopmentDefaultsStayOnLoopback(t *testing.T) {
	cfg, err := Load(MapSource{})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.ListenAddress != "127.0.0.1:8180" {
		t.Fatalf("ListenAddress = %q", cfg.ListenAddress)
	}
	if cfg.PublicURL != "http://127.0.0.1:8180" {
		t.Fatalf("PublicURL = %q", cfg.PublicURL)
	}
}

func completeProduction(overrides MapSource) MapSource {
	values := MapSource{
		"TCOMPUTE_ENV":                     "production",
		"TCOMPUTE_PUBLIC_URL":              "https://compute.example.test",
		"TCOMPUTE_LISTEN_ADDRESS":          "0.0.0.0:8180",
		"TCOMPUTE_DATABASE_URL":            "postgres://compute-db:5432/compute",
		"TCOMPUTE_REDIS_URL":               "redis://compute-redis:6379/0",
		"TCOMPUTE_ACCESS_PRIVATE_KEY_FILE": "/run/secrets/access-private-key",
		"TCOMPUTE_ACCESS_PUBLIC_KEY_FILE":  "/run/secrets/access-public-key",
		"TCOMPUTE_REFRESH_PEPPER_FILE":     "/run/secrets/refresh-pepper",
		"TCOMPUTE_ALLOWED_ORIGINS":         "https://compute.example.test",
	}
	for key, value := range overrides {
		values[key] = value
	}
	return values
}
