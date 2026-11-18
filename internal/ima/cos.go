package ima

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Uploader interface {
	Upload(ctx context.Context, filePath, contentType string, credential COSCredential) error
}

type COSUploader struct {
	HTTPClient *http.Client
}

func (u *COSUploader) Upload(ctx context.Context, filePath, contentType string, credential COSCredential) error {
	stat, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	if credential.SecretID == "" || credential.SecretKey == "" || credential.Token == "" || credential.BucketName == "" || credential.Region == "" || credential.COSKey == "" {
		return fmt.Errorf("IMA returned incomplete COS credentials")
	}
	if strings.ContainsAny(credential.BucketName+credential.Region, "/:@?#") {
		return fmt.Errorf("invalid COS bucket or region")
	}
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	host := fmt.Sprintf("%s.cos.%s.myqcloud.com", credential.BucketName, credential.Region)
	path := "/" + strings.TrimPrefix(credential.COSKey, "/")
	start := credential.StartTime
	expires := credential.ExpiredTime
	now := time.Now().Unix()
	if start == 0 {
		start = now
	}
	if expires <= start {
		expires = start + 3600
	}
	signedHeaders := map[string]string{
		"content-length": strconv.FormatInt(stat.Size(), 10),
		"host":           host,
	}
	auth := buildCOSAuthorization(credential.SecretID, credential.SecretKey, http.MethodPut, path, signedHeaders, start, expires)

	target := (&url.URL{Scheme: "https", Host: host, Path: path}).String()
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, target, file)
	if err != nil {
		return err
	}
	req.ContentLength = stat.Size()
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", auth)
	req.Header.Set("x-cos-security-token", credential.Token)

	client := u.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Minute}
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("COS upload failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		return fmt.Errorf("COS upload failed with HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func buildCOSAuthorization(secretID, secretKey, method, path string, headers map[string]string, start, expires int64) string {
	keyTime := fmt.Sprintf("%d;%d", start, expires)
	signKey := hmacSHA1Hex([]byte(secretKey), keyTime)

	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, strings.ToLower(key))
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, key+"="+encodeURIComponent(headers[key]))
	}
	httpString := strings.ToLower(method) + "\n" + path + "\n\n" + strings.Join(pairs, "&") + "\n"
	sum := sha1.Sum([]byte(httpString))
	stringToSign := "sha1\n" + keyTime + "\n" + hex.EncodeToString(sum[:]) + "\n"
	signature := hmacSHA1Hex([]byte(signKey), stringToSign)

	return strings.Join([]string{
		"q-sign-algorithm=sha1",
		"q-ak=" + secretID,
		"q-sign-time=" + keyTime,
		"q-key-time=" + keyTime,
		"q-header-list=" + strings.Join(keys, ";"),
		"q-url-param-list=",
		"q-signature=" + signature,
	}, "&")
}

func hmacSHA1Hex(key []byte, data string) string {
	h := hmac.New(sha1.New, key)
	_, _ = h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func encodeURIComponent(s string) string {
	// QueryEscape differs from JavaScript encodeURIComponent only by using '+' for spaces.
	// COS signed header values here (host and content-length) never contain spaces, but keep
	// the replacement for correctness and future reuse.
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}
