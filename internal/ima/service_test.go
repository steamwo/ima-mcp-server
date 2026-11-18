package ima

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type fakeUploader struct {
	mu    sync.Mutex
	calls int
	path  string
	ct    string
	creds COSCredential
	err   error
}

func (f *fakeUploader) Upload(_ context.Context, path, contentType string, credential COSCredential) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.path = path
	f.ct = contentType
	f.creds = credential
	return f.err
}

func TestUploadFileWorkflow(t *testing.T) {
	t.Parallel()
	var calls []string
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls = append(calls, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/openapi/wiki/v1/check_repeated_names":
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"results": []map[string]any{{"name": "report.pdf", "is_repeated": false}}}})
		case "/openapi/wiki/v1/create_media":
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{
				"media_id": "m1", "cos_credential": map[string]any{"secret_id": "sid", "secret_key": "skey", "token": "tok", "bucket_name": "b", "region": "r", "cos_key": "k"},
			}})
		case "/openapi/wiki/v1/add_knowledge":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["title"] != "report.pdf" {
				t.Fatalf("title = %#v", body["title"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"media_id": "m1"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	path := filepath.Join(t.TempDir(), "report.pdf")
	if err := os.WriteFile(path, []byte("%PDF-1.7\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	client, _ := NewClient(server.URL, "cid", "key", "", server.Client())
	uploader := &fakeUploader{}
	service := NewService(client, uploader)
	out, err := service.UploadFile(context.Background(), path, "", "kb1", "", "error", "")
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != "uploaded" || out.MediaID != "m1" || uploader.calls != 1 {
		t.Fatalf("unexpected result=%+v uploader=%+v", out, uploader)
	}
	want := []string{"/openapi/wiki/v1/check_repeated_names", "/openapi/wiki/v1/create_media", "/openapi/wiki/v1/add_knowledge"}
	if len(calls) != len(want) {
		t.Fatalf("calls = %#v", calls)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("calls = %#v", calls)
		}
	}
}

func TestUploadFileDuplicateStopsBeforeCreate(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/openapi/wiki/v1/check_repeated_names" {
			t.Fatalf("unexpected call after duplicate: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"results": []map[string]any{{"name": "report.pdf", "is_repeated": true}}}})
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "report.pdf")
	_ = os.WriteFile(path, []byte("%PDF-1.7\n"), 0o600)
	client, _ := NewClient(server.URL, "cid", "key", "", server.Client())
	uploader := &fakeUploader{}
	service := NewService(client, uploader)
	out, err := service.UploadFile(context.Background(), path, "", "kb1", "", "error", "")
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != "duplicate" || !out.Duplicate || uploader.calls != 0 {
		t.Fatalf("unexpected result=%+v calls=%d", out, uploader.calls)
	}
}

func TestUploadFileKeepBothUsesTimestamp(t *testing.T) {
	t.Parallel()
	checkCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/openapi/wiki/v1/check_repeated_names":
			checkCount++
			var body struct {
				Params []DuplicateParam `json:"params"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			repeated := checkCount == 1
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"results": []map[string]any{{"name": body.Params[0].Name, "is_repeated": repeated}}}})
		case "/openapi/wiki/v1/create_media":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["file_name"] != "report_20260102030405.pdf" {
				t.Fatalf("renamed file = %#v", body["file_name"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"media_id": "m1", "cos_credential": map[string]any{"secret_id": "sid", "secret_key": "skey", "token": "tok", "bucket_name": "b", "region": "r", "cos_key": "k"}}})
		case "/openapi/wiki/v1/add_knowledge":
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"media_id": "m1"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "report.pdf")
	_ = os.WriteFile(path, []byte("%PDF-1.7\n"), 0o600)
	client, _ := NewClient(server.URL, "cid", "key", "", server.Client())
	service := NewService(client, &fakeUploader{})
	service.Now = func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }
	out, err := service.UploadFile(context.Background(), path, "", "kb1", "", "keep_both", "")
	if err != nil {
		t.Fatal(err)
	}
	if out.FileName != "report_20260102030405.pdf" || out.OriginalFileName != "report.pdf" {
		t.Fatalf("unexpected result: %+v", out)
	}
}
