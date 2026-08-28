package capability

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateRejectsDuplicateIDs(t *testing.T) {
	err := Validate([]Capability{
		{ID: "system.health.read", Version: 1, Auth: AuthPublic, SideEffect: SideEffectNone, CLI: "health"},
		{ID: "system.health.read", Version: 1, Auth: AuthPublic, SideEffect: SideEffectNone, CLI: "health duplicate"},
	})
	if !errors.Is(err, ErrDuplicateCapability) {
		t.Fatalf("Validate() error = %v, want ErrDuplicateCapability", err)
	}
}

func TestValidateRejectsUnknownAuthorizationAndSideEffect(t *testing.T) {
	tests := []struct {
		name string
		cap  Capability
	}{
		{
			name: "unknown authorization",
			cap:  Capability{ID: "bad.auth", Version: 1, Auth: "client_admin", SideEffect: SideEffectNone, CLI: "bad auth"},
		},
		{
			name: "unknown side effect",
			cap:  Capability{ID: "bad.effect", Version: 1, Auth: AuthPublic, SideEffect: "shell", CLI: "bad effect"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := Validate([]Capability{test.cap}); err == nil {
				t.Fatal("Validate() error = nil, want rejection")
			}
		})
	}
}

func TestCheckBindingsRejectsMissingCLICommand(t *testing.T) {
	manifest := []Capability{{
		ID:         "admin.user.create",
		Version:    1,
		Auth:       AuthPlatformAdmin,
		SideEffect: SideEffectWrite,
		CLI:        "admin user create",
	}}

	err := CheckBindings(manifest, map[string]string{})
	if err == nil || err.Error() != "missing CLI binding: admin.user.create" {
		t.Fatalf("CheckBindings() error = %v", err)
	}
}

func TestLoadRejectsUnknownFieldsAndTrailingJSON(t *testing.T) {
	inputs := []string{
		`[{"id":"system.health.read","version":1,"cli":"health","auth":"public","side_effect":"none","role":"admin"}]`,
		`[{"id":"system.health.read","version":1,"cli":"health","auth":"public","side_effect":"none"}] {}`,
	}

	for _, input := range inputs {
		if _, err := Load(strings.NewReader(input)); err == nil {
			t.Fatalf("Load(%q) error = nil, want rejection", input)
		}
	}
}
