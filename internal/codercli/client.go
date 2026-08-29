package codercli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
)

const maxResponseBytes = 1 << 20

type Client struct {
	baseURL string
	http    *http.Client
}

type User struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Status   string `json:"status"`
	Roles    []struct {
		Name string `json:"name"`
	} `json:"roles"`
}

func NewClient(rawBaseURL string, httpClient *http.Client) (*Client, error) {
	baseURL, err := validateBaseURL(rawBaseURL)
	if err != nil {
		return nil, err
	}
	if httpClient == nil {
		return nil, errors.New("HTTP client is required")
	}
	return &Client{baseURL: baseURL, http: httpClient}, nil
}

func validateBaseURL(rawBaseURL string) (string, error) {
	parsed, err := url.Parse(rawBaseURL)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return "", errors.New("invalid Coder base URL")
	}
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && isLoopback(parsed.Hostname())) {
		return "", errors.New("Coder base URL must use HTTPS or loopback HTTP")
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (client *Client) Login(ctx context.Context, email, password string) (string, error) {
	if strings.TrimSpace(email) == "" || password == "" {
		return "", errors.New("email and password are required")
	}
	var response struct {
		SessionToken string `json:"session_token"`
	}
	if err := client.do(ctx, http.MethodPost, "/api/v2/users/login", "", map[string]string{"email": email, "password": password}, &response); err != nil {
		return "", err
	}
	if response.SessionToken == "" {
		return "", errors.New("Coder login returned no session token")
	}
	return response.SessionToken, nil
}

func (client *Client) Me(ctx context.Context, token string) (User, error) {
	var user User
	err := client.do(ctx, http.MethodGet, "/api/v2/users/me", token, nil, &user)
	return user, err
}

func (client *Client) Logout(ctx context.Context, token string) error {
	return client.do(ctx, http.MethodPost, "/api/v2/users/logout", token, nil, nil)
}

func (client *Client) ChangePassword(ctx context.Context, token, currentPassword, newPassword string) error {
	if currentPassword == "" || newPassword == "" {
		return errors.New("current and new passwords are required")
	}
	return client.do(ctx, http.MethodPut, "/api/v2/users/me/password", token, map[string]string{"old_password": currentPassword, "password": newPassword}, nil)
}

func (client *Client) CreateUser(ctx context.Context, token, username, email, name, password string) (User, error) {
	if username == "" || email == "" || password == "" {
		return User{}, errors.New("username, email and password are required")
	}
	var organizations []struct {
		ID string `json:"id"`
	}
	if err := client.do(ctx, http.MethodGet, "/api/v2/organizations", token, nil, &organizations); err != nil {
		return User{}, err
	}
	if len(organizations) == 0 || organizations[0].ID == "" {
		return User{}, errors.New("Coder has no default organization")
	}
	payload := map[string]any{
		"username": username, "email": email, "name": name, "password": password,
		"login_type": "password", "organization_ids": []string{organizations[0].ID},
		"roles": []string{}, "service_account": false, "user_status": "active",
	}
	var user User
	err := client.do(ctx, http.MethodPost, "/api/v2/users", token, payload, &user)
	return user, err
}

func (client *Client) ResetPassword(ctx context.Context, token, username, password string) error {
	if username == "" || password == "" {
		return errors.New("username and password are required")
	}
	return client.do(ctx, http.MethodPut, "/api/v2/users/"+url.PathEscape(username)+"/password", token, map[string]string{"old_password": "", "password": password}, nil)
}

func (client *Client) do(ctx context.Context, method, path, token string, input, output any) error {
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return fmt.Errorf("encode Coder request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, client.baseURL+path, body)
	if err != nil {
		return errors.New("create Coder request")
	}
	if input != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		if strings.ContainsAny(token, "\r\n") {
			return errors.New("invalid stored Coder session")
		}
		request.Header.Set("Coder-Session-Token", token)
	}
	response, err := client.http.Do(request)
	if err != nil {
		return errors.New("Coder request failed")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return fmt.Errorf("Coder request failed with HTTP %d", response.StatusCode)
	}
	if output == nil || response.StatusCode == http.StatusNoContent {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maxResponseBytes))
		return nil
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, maxResponseBytes))
	if err := decoder.Decode(output); err != nil {
		return errors.New("decode Coder response")
	}
	return nil
}
