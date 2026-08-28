package audit

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRedactRemovesSensitiveValuesRecursively(t *testing.T) {
	input := map[string]any{
		"username": "alice",
		"password": "fixture-password",
		"nested": map[string]any{
			"refresh_token": "fixture-refresh-token",
			"Authorization": "Bearer fixture-access-token",
			"safe":          "visible",
		},
		"items": []any{map[string]any{"private_key": "fixture-private-key"}},
	}

	encoded, err := json.Marshal(Redact(input))
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	output := string(encoded)
	for _, secret := range []string{"fixture-password", "fixture-refresh-token", "fixture-access-token", "fixture-private-key"} {
		if strings.Contains(output, secret) {
			t.Fatalf("redacted output contains %q: %s", secret, output)
		}
	}
	if !strings.Contains(output, "visible") || !strings.Contains(output, "alice") {
		t.Fatalf("safe values were removed: %s", output)
	}
}
