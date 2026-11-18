package ima

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientCallHeadersAndDecode(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/openapi/note/v1/get_doc_content" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("ima-openapi-clientid"); got != "cid" {
			t.Fatalf("client id header = %q", got)
		}
		if got := r.Header.Get("ima-openapi-apikey"); got != "key" {
			t.Fatalf("api key header = %q", got)
		}
		if got := r.Header.Get("ima-openapi-ctx"); got != "skill_version=1.1.10" {
			t.Fatalf("ctx header = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "msg": "success", "data": map[string]string{"content": "hello"}})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "cid", "key", "skill_version=1.1.10", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	out, err := client.GetNoteContent(context.Background(), "note-1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if out.Content != "hello" {
		t.Fatalf("content = %q", out.Content)
	}
}

func TestClientAPIError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 110030, "msg": "no permission", "data": map[string]any{}})
	}))
	defer server.Close()
	client, _ := NewClient(server.URL, "cid", "key", "", server.Client())
	_, err := client.GetMediaInfo(context.Background(), "m1")
	apiErr, ok := err.(*APIError)
	if !ok || apiErr.Code != 110030 || apiErr.Msg != "no permission" {
		t.Fatalf("unexpected error: %#v", err)
	}
}
