package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/TashanGKD/tashan-compute/internal/httpapi"
)

const maxResponseBytes = 1 << 20

type Client struct {
	baseURL *url.URL
	http    *http.Client
}

func New(rawBaseURL string, httpClient *http.Client) (*Client, error) {
	parsed, err := url.Parse(rawBaseURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil {
		return nil, errors.New("API base URL must be an HTTP(S) origin without credentials")
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{baseURL: parsed, http: httpClient}, nil
}

func (client *Client) Login(ctx context.Context, input httpapi.LoginInput) (httpapi.LoginResult, error) {
	var result httpapi.LoginResult
	if err := client.Do(ctx, http.MethodPost, httpapi.RouteLogin, "", input, &result); err != nil {
		return httpapi.LoginResult{}, err
	}
	return result, nil
}

func (client *Client) Do(ctx context.Context, method, path, accessToken string, input, output any) error {
	var body []byte
	var err error
	if input != nil {
		body, err = json.Marshal(input)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
	}
	target := client.baseURL.ResolveReference(&url.URL{Path: path})
	request, err := http.NewRequestWithContext(ctx, method, target.String(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if accessToken != "" {
		request.Header.Set("Authorization", "Bearer "+accessToken)
	}
	response, err := client.http.Do(request)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer response.Body.Close()
	limited := io.LimitReader(response.Body, maxResponseBytes+1)
	contents, err := io.ReadAll(limited)
	if err != nil {
		return fmt.Errorf("read API response: %w", err)
	}
	if len(contents) > maxResponseBytes {
		return errors.New("API response exceeds size limit")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("API returned HTTP %d", response.StatusCode)
	}
	if output == nil || len(contents) == 0 {
		return nil
	}
	decoder := json.NewDecoder(strings.NewReader(string(contents)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		return fmt.Errorf("decode API response: %w", err)
	}
	return nil
}
