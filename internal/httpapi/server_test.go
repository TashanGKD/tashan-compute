package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthDoesNotExposeConfiguration(t *testing.T) {
	handler := NewServer(ServerOptions{Version: "test-version"})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/health", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	body := recorder.Body.String()
	if body != "{\"status\":\"ok\",\"version\":\"test-version\"}\n" {
		t.Fatalf("body = %q", body)
	}
}

func TestDecodeJSONRejectsMalformedAndTrailingValues(t *testing.T) {
	for _, input := range []string{`{"name":`, `{"name":"alice"} {}`} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(input))
		var target struct {
			Name string `json:"name"`
		}

		err := DecodeJSON(recorder, request, &target)
		if err == nil {
			t.Fatalf("DecodeJSON(%q) error = nil", input)
		}
		var response ErrorResponse
		if decodeErr := json.Unmarshal(recorder.Body.Bytes(), &response); decodeErr != nil {
			t.Fatalf("response JSON error = %v", decodeErr)
		}
		if response.Error.Code != "request.invalid_json" {
			t.Fatalf("code = %q", response.Error.Code)
		}
		if strings.Contains(recorder.Body.String(), "unexpected EOF") {
			t.Fatalf("internal parser error leaked: %s", recorder.Body.String())
		}
	}
}

func TestWriteErrorDoesNotLeakInternalError(t *testing.T) {
	recorder := httptest.NewRecorder()
	WriteError(recorder, http.StatusInternalServerError, "internal.error", "request failed", "req-1", errors.New("database password leaked"))

	if strings.Contains(recorder.Body.String(), "database password") {
		t.Fatalf("internal error leaked: %s", recorder.Body.String())
	}
}
