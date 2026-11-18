package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const DefaultBaseURL = "https://ima.qq.com"

type Config struct {
	ClientID       string
	APIKey         string
	BaseURL        string
	OpenAPIContext string
	Timeout        time.Duration
}

func Load() (Config, error) {
	clientID := firstNonEmpty(
		os.Getenv("IMA_OPENAPI_CLIENTID"),
		os.Getenv("IMA_CLIENT_ID"),
		readCredential("client_id"),
	)
	apiKey := firstNonEmpty(
		os.Getenv("IMA_OPENAPI_APIKEY"),
		os.Getenv("IMA_API_KEY"),
		readCredential("api_key"),
	)
	if clientID == "" || apiKey == "" {
		return Config{}, errors.New("missing IMA credentials: set IMA_OPENAPI_CLIENTID and IMA_OPENAPI_APIKEY (or IMA_CLIENT_ID/IMA_API_KEY), or place them in ~/.config/ima/client_id and ~/.config/ima/api_key")
	}

	baseURL := strings.TrimSpace(os.Getenv("IMA_BASE_URL"))
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if err := validateBaseURL(baseURL); err != nil {
		return Config{}, err
	}

	timeout := 60 * time.Second
	if raw := strings.TrimSpace(os.Getenv("IMA_HTTP_TIMEOUT")); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil || parsed <= 0 {
			return Config{}, fmt.Errorf("invalid IMA_HTTP_TIMEOUT %q", raw)
		}
		timeout = parsed
	}

	ctx := strings.TrimSpace(os.Getenv("IMA_OPENAPI_CTX"))
	if ctx == "" {
		ctx = "skill_version=1.1.10"
	}

	return Config{
		ClientID:       clientID,
		APIKey:         apiKey,
		BaseURL:        baseURL,
		OpenAPIContext: ctx,
		Timeout:        timeout,
	}, nil
}

func validateBaseURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("invalid IMA_BASE_URL %q", raw)
	}
	if u.Scheme == "https" && strings.EqualFold(u.Hostname(), "ima.qq.com") && (u.Path == "" || u.Path == "/") && u.RawQuery == "" && u.Fragment == "" {
		return nil
	}
	if os.Getenv("IMA_ALLOW_CUSTOM_BASE_URL") != "1" {
		return fmt.Errorf("refusing custom IMA_BASE_URL %q: credentials are only sent to %s by default; set IMA_ALLOW_CUSTOM_BASE_URL=1 only for a trusted test endpoint", raw, DefaultBaseURL)
	}
	if u.Scheme != "https" && os.Getenv("IMA_ALLOW_INSECURE_BASE_URL") != "1" {
		return fmt.Errorf("custom IMA_BASE_URL must use https unless IMA_ALLOW_INSECURE_BASE_URL=1")
	}
	return nil
}

func readCredential(name string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	b, err := os.ReadFile(filepath.Join(home, ".config", "ima", name))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
