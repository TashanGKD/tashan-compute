package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TashanGKD/tashan-compute/internal/httpapi"
)

func TestLoginUsesServerContractWithoutRoleField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/auth/login" || request.Method != http.MethodPost {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if _, exists := body["role"]; exists {
			t.Fatalf("client sent role: %v", body)
		}
		_ = json.NewEncoder(writer).Encode(httpapi.LoginResult{
			AccessToken: "fixture-access", RefreshToken: "fixture-refresh",
			AccessExpiresAt: time.Now().Add(time.Minute), MustChangePassword: true,
		})
	}))
	defer server.Close()

	api, err := New(server.URL, server.Client())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	result, err := api.Login(t.Context(), httpapi.LoginInput{
		Username: "alice", Password: "fixture-password", DeviceLabel: "mac", DeviceFingerprint: "fp",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if result.RefreshToken != "fixture-refresh" {
		t.Fatalf("Login() result = %+v", result)
	}
}

func TestClientRejectsNonHTTPBaseURLAndOversizedResponse(t *testing.T) {
	if _, err := New("file:///tmp/socket", http.DefaultClient); err == nil {
		t.Fatal("New() accepted non-HTTP URL")
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write(make([]byte, maxResponseBytes+1))
	}))
	defer server.Close()
	api, _ := New(server.URL, server.Client())
	if _, err := api.Login(t.Context(), httpapi.LoginInput{Username: "alice"}); err == nil {
		t.Fatal("Login() accepted oversized response")
	}
}

func TestDoSendsBearerTokenAndDecodesStableJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer fixture-access" {
			t.Fatalf("Authorization = %q", request.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(writer).Encode(map[string]bool{"ok": true})
	}))
	defer server.Close()
	api, _ := New(server.URL, server.Client())
	var output struct {
		OK bool `json:"ok"`
	}
	if err := api.Do(t.Context(), http.MethodGet, "/v1/example", "fixture-access", nil, &output); err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if !output.OK {
		t.Fatal("Do() did not decode response")
	}
}
