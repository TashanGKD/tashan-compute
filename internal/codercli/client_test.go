package codercli

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientLoginAndMe(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v2/users/login", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("content type = %q", got)
		}
		if r.UserAgent() == "" || r.Header.Get("X-Tcompute-OS") == "" || r.Header.Get("X-Tcompute-Arch") == "" || r.Header.Get("X-Tcompute-Device") == "" {
			t.Fatal("device audit headers are missing")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"session_token":"session-secret"}`))
	})
	mux.HandleFunc("GET /api/v2/users/me", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Coder-Session-Token"); got != "session-secret" {
			t.Fatalf("session header = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"username":"alice","email":"alice@example.test","status":"active","roles":[{"name":"member"}]}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	token, err := client.Login(context.Background(), "alice@example.test", "password-value")
	if err != nil {
		t.Fatal(err)
	}
	user, err := client.Me(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if token != "session-secret" || user.Username != "alice" || user.Email != "alice@example.test" {
		t.Fatalf("unexpected login result: token=%q user=%+v", token, user)
	}
}

func TestClientRejectsHostileBaseURLs(t *testing.T) {
	for _, raw := range []string{
		"http://example.com",
		"https://compute.tashan.chat@evil.example",
		"file:///etc/passwd",
	} {
		if _, err := NewClient(raw, http.DefaultClient); err == nil {
			t.Fatalf("accepted hostile base URL %q", raw)
		}
	}
}

func TestClientErrorDoesNotLeakPasswordOrToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"invalid credentials"}`, http.StatusUnauthorized)
	}))
	defer server.Close()
	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	secret := "very-secret-password"
	_, err = client.Login(context.Background(), "alice@example.test", secret)
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Fatalf("unsafe error: %v", err)
	}
}
