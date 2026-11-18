package ima

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	markdownImageRE = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	dataImageRE     = regexp.MustCompile(`(?i)^data:(image/(?:png|jpeg|webp));base64,([A-Za-z0-9+/]+={0,2})$`)
)

type MarkdownBatch struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	Bytes   int    `json:"bytes"`
}

func CreateMarkdownBatches(ctx context.Context, markdown string) ([]MarkdownBatch, error) {
	matches := markdownImageRE.FindAllStringSubmatchIndex(markdown, -1)
	if len(matches) == 0 {
		if strings.TrimSpace(markdown) == "" {
			return nil, fmt.Errorf("input content is empty")
		}
		return []MarkdownBatch{{Type: "text", Content: markdown, Bytes: len([]byte(markdown))}}, nil
	}

	// Preprocess every image before returning any batch. This preserves the Skill's
	// all-images-preflight-before-write safety rule.
	batches := make([]MarkdownBatch, 0, len(matches)*2+1)
	last := 0
	for _, m := range matches {
		if m[0] > last {
			text := markdown[last:m[0]]
			if strings.TrimSpace(text) != "" {
				batches = append(batches, MarkdownBatch{Type: "text", Content: text, Bytes: len([]byte(text))})
			}
		}
		alt := markdown[m[2]:m[3]]
		source := strings.TrimSpace(markdown[m[4]:m[5]])
		imageMarkdown, err := normalizeMarkdownImage(ctx, alt, source)
		if err != nil {
			return nil, err
		}
		batches = append(batches, MarkdownBatch{Type: "image", Content: imageMarkdown, Bytes: len([]byte(imageMarkdown))})
		last = m[1]
	}
	if last < len(markdown) {
		text := markdown[last:]
		if strings.TrimSpace(text) != "" {
			batches = append(batches, MarkdownBatch{Type: "text", Content: text, Bytes: len([]byte(text))})
		}
	}
	if len(batches) == 0 {
		return nil, fmt.Errorf("input content is empty")
	}
	return batches, nil
}

func normalizeMarkdownImage(ctx context.Context, alt, source string) (string, error) {
	var contentType string
	var data []byte
	var err error

	switch {
	case strings.HasPrefix(source, "data:"):
		contentType, data, err = parseDataImage(source)
	case strings.HasPrefix(source, "file://"):
		path, parseErr := fileURIPath(source)
		if parseErr != nil {
			return "", parseErr
		}
		contentType, data, err = readLocalImage(path)
	case filepath.IsAbs(source):
		contentType, data, err = readLocalImage(source)
	case strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://"):
		contentType, data, err = fetchRemoteImage(ctx, source)
	default:
		return "", fmt.Errorf("unsupported image source %q: use an absolute path, file:// URI, data URL, or http(s) URL", source)
	}
	if err != nil {
		return "", err
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("![%s](data:%s;base64,%s)", alt, contentType, encoded), nil
}

func readLocalImage(path string) (string, []byte, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return "", nil, fmt.Errorf("image file not found: %w", err)
	}
	if !stat.Mode().IsRegular() {
		return "", nil, fmt.Errorf("image path is not a file: %s", path)
	}
	if stat.Size() >= MaxNoteImageSize {
		return "", nil, fmt.Errorf("image must be smaller than 3 MiB: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, err
	}
	contentType := detectImageType(data)
	if contentType == "" {
		return "", nil, fmt.Errorf("unsupported or invalid image: %s", path)
	}
	return contentType, data, nil
}

func parseDataImage(source string) (string, []byte, error) {
	match := dataImageRE.FindStringSubmatch(source)
	if match == nil {
		return "", nil, fmt.Errorf("unsupported or invalid data URL; only base64 PNG, JPEG and WebP are supported")
	}
	data, err := base64.StdEncoding.DecodeString(match[2])
	if err != nil {
		return "", nil, fmt.Errorf("invalid image base64: %w", err)
	}
	if len(data) >= MaxNoteImageSize {
		return "", nil, fmt.Errorf("image must be smaller than 3 MiB")
	}
	declared := strings.ToLower(match[1])
	detected := detectImageType(data)
	if detected == "" || detected != declared {
		return "", nil, fmt.Errorf("image signature does not match declared data URL type")
	}
	return detected, data, nil
}

func fetchRemoteImage(ctx context.Context, source string) (string, []byte, error) {
	u, err := url.Parse(source)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return "", nil, fmt.Errorf("invalid image URL %q", source)
	}
	client := newSafeHTTPClient(30 * time.Second)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Accept", "image/png,image/jpeg,image/webp,image/*;q=0.8")
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("remote image download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil, fmt.Errorf("remote image request failed with HTTP %d", resp.StatusCode)
	}
	ct := normalizeContentType(resp.Header.Get("Content-Type"))
	if ct != "" && ct != "application/octet-stream" && ct != "image/png" && ct != "image/jpeg" && ct != "image/webp" {
		return "", nil, fmt.Errorf("remote response is not a supported image (%s)", ct)
	}
	if resp.ContentLength >= MaxNoteImageSize {
		return "", nil, fmt.Errorf("remote image must be smaller than 3 MiB")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxNoteImageSize))
	if err != nil {
		return "", nil, err
	}
	if len(data) >= MaxNoteImageSize {
		return "", nil, fmt.Errorf("remote image must be smaller than 3 MiB")
	}
	detected := detectImageType(data)
	if detected == "" {
		return "", nil, fmt.Errorf("remote image has an invalid or unsupported signature")
	}
	if strings.HasPrefix(ct, "image/") && ct != detected {
		return "", nil, fmt.Errorf("remote image signature %s does not match Content-Type %s", detected, ct)
	}
	return detected, data, nil
}

func fileURIPath(source string) (string, error) {
	u, err := url.Parse(source)
	if err != nil || u.Scheme != "file" {
		return "", fmt.Errorf("invalid file URI %q", source)
	}
	if u.Host != "" && u.Host != "localhost" {
		return "", fmt.Errorf("remote file URI hosts are not supported")
	}
	path, err := url.PathUnescape(u.Path)
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", fmt.Errorf("empty file URI path")
	}
	return filepath.FromSlash(path), nil
}
