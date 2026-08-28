package e2e

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TashanGKD/tashan-compute/internal/app"
	"github.com/TashanGKD/tashan-compute/internal/auth"
	"github.com/TashanGKD/tashan-compute/internal/httpapi"
	"github.com/TashanGKD/tashan-compute/internal/identity"
	store "github.com/TashanGKD/tashan-compute/internal/store/postgres"
	"github.com/TashanGKD/tashan-compute/internal/testkit"
)

func TestTwoManagedUsersCompleteIndependentPasswordLifecycles(t *testing.T) {
	ctx := context.Background()
	db := testkit.Postgres(t)
	if _, err := db.Exec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public`); err != nil {
		t.Fatalf("reset schema: %v", err)
	}
	if err := store.Migrate(ctx, db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	hasher := auth.NewPasswordHasher(auth.DefaultPasswordParams())
	adminHash, _ := hasher.Hash("platform admin fixture password", "root")
	accounts := store.NewAccountStore(db)
	if _, err := accounts.Bootstrap(ctx, identity.Account{Username: "root", PasswordHash: adminHash, PasswordVersion: 1}); err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	sessions := store.NewSessionStore(db)
	authService := auth.NewService(accounts, hasher)
	now := time.Now
	operations := app.NewAuthOperations(app.AuthDependencies{
		Accounts: accounts, Devices: store.NewDeviceStore(db), Sessions: sessions,
		AuthService:    authService,
		SessionManager: auth.NewSessionManager(sessions, []byte("fixture refresh pepper at least 32 bytes"), 30*24*time.Hour, now, rand.Reader),
		AccessSigner:   auth.NewAccessTokenSigner(privateKey, "tashan-compute", "tcompute", 10*time.Minute, now),
		AccessLifetime: 10 * time.Minute, Hasher: hasher, Now: now,
	})
	authenticator := httpapi.NewTokenAuthenticator(
		auth.NewAccessTokenVerifier(publicKey, "tashan-compute", "tcompute", now),
		store.NewPrincipalStore(db),
	)
	handler := httpapi.NewServer(httpapi.ServerOptions{
		Version: "test", Authenticator: authenticator, AuthOperations: operations,
		AdminUsers: app.NewAdminUsers(authService),
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	admin := login(t, server.URL, "root", "platform admin fixture password", "admin-device")
	for _, username := range []string{"alice", "bob"} {
		response := requestJSON(t, http.MethodPost, server.URL+"/v1/admin/users", admin.AccessToken, map[string]any{
			"username": username, "initial_password": username + " initial fixture password",
		})
		if response.Code != http.StatusCreated {
			t.Fatalf("create %s status=%d body=%s", username, response.Code, response.Body.String())
		}
		initial := login(t, server.URL, username, username+" initial fixture password", username+"-device")
		if !initial.MustChangePassword {
			t.Fatalf("%s did not require initial password change", username)
		}
		changed := requestJSON(t, http.MethodPost, server.URL+"/v1/auth/password/initial-change", initial.AccessToken, map[string]any{
			"current_password": username + " initial fixture password",
			"new_password":     username + " permanent fixture password",
		})
		if changed.Code != http.StatusOK {
			t.Fatalf("change %s status=%d body=%s", username, changed.Code, changed.Body.String())
		}
		oldWhoAmI := requestJSON(t, http.MethodGet, server.URL+"/v1/auth/whoami", initial.AccessToken, nil)
		if oldWhoAmI.Code != http.StatusUnauthorized {
			t.Fatalf("old %s token status=%d", username, oldWhoAmI.Code)
		}
		permanent := login(t, server.URL, username, username+" permanent fixture password", username+"-device")
		if permanent.MustChangePassword {
			t.Fatalf("%s permanent login still requires change", username)
		}
		whoami := requestJSON(t, http.MethodGet, server.URL+"/v1/auth/whoami", permanent.AccessToken, nil)
		if whoami.Code != http.StatusOK {
			t.Fatalf("whoami %s status=%d body=%s", username, whoami.Code, whoami.Body.String())
		}
	}
}

func login(t *testing.T, baseURL, username, password, fingerprint string) httpapi.LoginResult {
	t.Helper()
	response := requestJSON(t, http.MethodPost, baseURL+"/v1/auth/login", "", map[string]any{
		"username": username, "password": password,
		"device_label": username + " laptop", "device_fingerprint": fingerprint,
	})
	if response.Code != http.StatusOK {
		t.Fatalf("login %s status=%d body=%s", username, response.Code, response.Body.String())
	}
	var result httpapi.LoginResult
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	return result
}

type capturedResponse struct {
	Code int
	Body *bytes.Buffer
}

func requestJSON(t *testing.T, method, url, token string, body any) *capturedResponse {
	t.Helper()
	var encoded []byte
	if body != nil {
		var err error
		encoded, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("encode request: %v", err)
		}
	}
	request, err := http.NewRequest(method, url, bytes.NewReader(encoded))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer response.Body.Close()
	contents, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	return &capturedResponse{Code: response.StatusCode, Body: bytes.NewBuffer(contents)}
}
