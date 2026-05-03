// Package client wraps the Pulse HTTP API for the TUI. Each resource
// (projects, models, usage) gets its own file. Types here mirror the JSON
// wire shape from the API, NOT the internal store millicent representation.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client is a thin HTTP wrapper around the Pulse API. Construct via New.
type Client struct {
	baseURL string
	http    *http.Client
}

// New builds a Client. baseURL has no trailing slash.
func New(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: timeout},
	}
}

// BaseURL returns the configured root URL. Used by the fatal-error screen.
func (c *Client) BaseURL() string { return c.baseURL }

// ClientError is returned for 4xx responses. Message is parsed from the
// {"error": "..."} body produced by internal/http/errors.go.
type ClientError struct {
	Status  int
	Message string
}

func (e *ClientError) Error() string {
	return fmt.Sprintf("api error %d: %s", e.Status, e.Message)
}

// ServerError is returned for 5xx responses.
type ServerError struct {
	Status int
	Body   string
}

func (e *ServerError) Error() string {
	return fmt.Sprintf("server error %d: %s", e.Status, e.Body)
}

// do executes the request and decodes the JSON response into out. If out is
// nil the body is read+discarded. 2xx is success; 4xx maps to ClientError;
// 5xx maps to ServerError; everything else is wrapped as a generic error.
func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var bodyReader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		if out == nil || len(respBody) == 0 {
			return nil
		}
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
		return nil

	case resp.StatusCode >= 400 && resp.StatusCode < 500:
		var apiErr struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(respBody, &apiErr)
		msg := apiErr.Error
		if msg == "" {
			msg = string(respBody)
		}
		return &ClientError{Status: resp.StatusCode, Message: msg}

	case resp.StatusCode >= 500:
		return &ServerError{Status: resp.StatusCode, Body: string(respBody)}

	default:
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}
}

// queryString builds an encoded query string from a map. Empty-value keys are skipped.
func queryString(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}
	v := url.Values{}
	for k, val := range params {
		if val == "" {
			continue
		}
		v.Set(k, val)
	}
	enc := v.Encode()
	if enc == "" {
		return ""
	}
	return "?" + enc
}
