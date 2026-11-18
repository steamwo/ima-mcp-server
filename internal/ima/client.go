package ima

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
	"time"
)

const maxAPIResponseBytes = 16 << 20

type Client struct {
	baseURL        *url.URL
	httpClient     *http.Client
	clientID       string
	apiKey         string
	openAPIContext string
}

type APIError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (e *APIError) Error() string {
	if e.Msg != "" {
		return fmt.Sprintf("IMA API error %d: %s", e.Code, e.Msg)
	}
	return fmt.Sprintf("IMA API error %d", e.Code)
}

type apiEnvelope struct {
	Code Int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func NewClient(baseURL, clientID, apiKey, openAPIContext string, httpClient *http.Client) (*Client, error) {
	u, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid IMA base URL %q", baseURL)
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	return &Client{
		baseURL:        u,
		httpClient:     httpClient,
		clientID:       clientID,
		apiKey:         apiKey,
		openAPIContext: openAPIContext,
	}, nil
}

func (c *Client) Call(ctx context.Context, apiPath string, body any, out any) error {
	apiPath = strings.TrimPrefix(strings.TrimSpace(apiPath), "/")
	if !strings.HasPrefix(apiPath, "openapi/") || strings.Contains(apiPath, "..") {
		return fmt.Errorf("invalid IMA API path %q", apiPath)
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	u := *c.baseURL
	u.Path = strings.TrimRight(u.Path, "/") + "/" + apiPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("ima-openapi-clientid", c.clientID)
	req.Header.Set("ima-openapi-apikey", c.apiKey)
	if c.openAPIContext != "" {
		req.Header.Set("ima-openapi-ctx", c.openAPIContext)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("IMA request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxAPIResponseBytes+1))
	if err != nil {
		return fmt.Errorf("read IMA response: %w", err)
	}
	if len(raw) > maxAPIResponseBytes {
		return errors.New("IMA response exceeds 16 MiB limit")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("IMA HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var envelope apiEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("IMA returned invalid JSON: %w", err)
	}
	if envelope.Code != 0 {
		return &APIError{Code: int(envelope.Code), Msg: envelope.Msg}
	}
	if out == nil || len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(envelope.Data, out); err != nil {
		return fmt.Errorf("decode IMA response data: %w", err)
	}
	return nil
}
