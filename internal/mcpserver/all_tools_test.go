package mcpserver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/steamwo/ima-mcp-server/internal/ima"
)

type recordingUploader struct {
	mu    sync.Mutex
	calls int
}

func (u *recordingUploader) Upload(context.Context, string, string, ima.COSCredential) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.calls++
	return nil
}

func (u *recordingUploader) Calls() int {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.calls
}

func TestAllMCPToolsThroughProtocol(t *testing.T) {
	var mu sync.Mutex
	apiCalls := map[string]int{}

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("ima-openapi-clientid"); got != "test-client" {
			t.Errorf("client id header = %q", got)
		}
		if got := r.Header.Get("ima-openapi-apikey"); got != "test-key" {
			t.Errorf("api key header = %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		var request map[string]any
		_ = json.Unmarshal(body, &request)

		mu.Lock()
		apiCalls[r.URL.Path]++
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		var data string
		switch r.URL.Path {
		case "/openapi/note/v1/search_note":
			data = `{"search_note_infos":[{"note_book_info":{"note_id":"note-1","title":"连接器","summary":"summary","create_time":"1730000000000","modify_time":"1730000001000","note_ext_info":{"folder_id":"folder-1","folder_name":"测试"}},"highlightInfo":{"doc_title":"<em>连接器</em>"}}],"is_end":"true","total_hit_num":"1"}`
		case "/openapi/note/v1/list_note":
			data = `{"note_book_list":[{"note_id":"note-1","title":"连接器","summary":"summary","create_time":"1730000000000","modify_time":"1730000001000","note_ext_info":{"folder_id":"folder-1","folder_name":"测试"}}],"is_end":"true"}`
		case "/openapi/note/v1/get_doc_content":
			data = `{"content":"测试笔记内容"}`
		case "/openapi/note/v1/import_doc":
			data = `{"note_id":"note-created"}`
		case "/openapi/note/v1/append_doc":
			data = `{"note_id":"note-1"}`
		case "/openapi/note/v1/list_notebook":
			data = `{"note_folder_infos":[{"folder_id":"folder-1","name":"测试","create_time":"1730000000000","modify_time":"1730000001000","note_number":"3","parent_folder_id":"","folder_type":"0"}],"next_cursor":0,"is_end":"true","next_version":123,"need_update":"false"}`
		case "/openapi/wiki/v1/get_knowledge_base":
			data = `{"infos":{"kb-1":{"id":"kb-1","name":"测试知识库","cover_url":"","description":"desc","recommended_questions":["q1"]}}}`
		case "/openapi/wiki/v1/get_knowledge_list":
			data = `{"knowledge_list":[{"media_id":"media-1","title":"文档","parent_folder_id":"kb-1"}],"is_end":"true","next_cursor":0,"current_path":[{"folder_id":"kb-1","name":"根目录","file_number":"1","folder_number":"0","parent_folder_id":"","is_top":"false"}]}`
		case "/openapi/wiki/v1/search_knowledge":
			data = `{"info_list":[{"media_id":"media-1","title":"连接器文档","parent_folder_id":"kb-1","highlight_content":"<em>连接器</em>"}],"is_end":"true","next_cursor":0}`
		case "/openapi/wiki/v1/search_knowledge_base":
			data = `{"info_list":[{"id":"kb-1","name":"测试知识库","cover_url":""}],"is_end":"true","next_cursor":0}`
		case "/openapi/wiki/v1/get_addable_knowledge_base_list":
			data = `{"addable_knowledge_base_list":[{"id":"kb-1","name":"测试知识库"}],"next_cursor":0,"is_end":"true"}`
		case "/openapi/wiki/v1/check_repeated_names":
			data = `{"results":[{"name":"probe.txt","is_repeated":"false"}]}`
		case "/openapi/wiki/v1/import_urls":
			data = `{"results":{"https://example.com":{"url":"https://example.com","ret_code":"0","media_id":"media-url"},"https://mp.weixin.qq.com/s/test":{"url":"https://mp.weixin.qq.com/s/test","ret_code":"0","media_id":"media-wechat"}}}`
		case "/openapi/wiki/v1/get_media_info":
			data = `{"media_type":"11","notebook_ext_info":{"notebook_id":"note-1"}}`
		case "/openapi/wiki/v1/create_media":
			data = `{"media_id":"media-upload","cos_credential":{"token":"token","secret_id":"sid","secret_key":"skey","start_time":"1730000000","expired_time":"1730003600","appid":"123","bucket_name":"bucket-123","region":"ap-test","custom_domain":"","cos_key":"uploads/probe.txt"}}`
		case "/openapi/wiki/v1/add_knowledge":
			data = `{"media_id":"media-added"}`
		default:
			http.Error(w, "unexpected endpoint "+r.URL.Path, http.StatusNotFound)
			return
		}
		_, _ = io.WriteString(w, `{"code":"0","msg":"success","data":`+data+`}`)
	}))
	defer api.Close()

	client, err := ima.NewClient(api.URL, "test-client", "test-key", "skill_version=1.1.10", api.Client())
	if err != nil {
		t.Fatal(err)
	}
	uploader := &recordingUploader{}
	service := ima.NewService(client, uploader)
	server := New(service, "test")

	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()

	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "contract-test", Version: "1"}, nil)
	clientSession, err := mcpClient.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	listed, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(listed.Tools); got != 19 {
		t.Fatalf("tool count = %d, want 19", got)
	}
	wantNames := map[string]bool{
		"ima_search_notes": true, "ima_list_notes": true, "ima_get_note": true,
		"ima_create_note": true, "ima_append_note": true, "ima_append_note_markdown": true,
		"ima_list_notebooks": true, "ima_get_knowledge_bases": true, "ima_list_knowledge": true,
		"ima_search_knowledge": true, "ima_search_knowledge_bases": true,
		"ima_list_addable_knowledge_bases": true, "ima_check_duplicate_files": true,
		"ima_import_urls": true, "ima_get_media_info": true, "ima_add_note_to_knowledge_base": true,
		"ima_preflight_file": true, "ima_upload_file": true, "ima_add_url": true,
	}
	for _, tool := range listed.Tools {
		delete(wantNames, tool.Name)
	}
	if len(wantNames) != 0 {
		t.Fatalf("missing tools: %v", wantNames)
	}

	tmp, err := os.CreateTemp("", "probe-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.WriteString("ima mcp contract test\n"); err != nil {
		t.Fatal(err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpPath)

	call := func(name string, args map[string]any) *mcp.CallToolResult {
		t.Helper()
		res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		if err != nil {
			t.Fatalf("%s protocol error: %v", name, err)
		}
		if res.IsError {
			b, _ := json.Marshal(res.Content)
			t.Fatalf("%s tool error: %s", name, b)
		}
		return res
	}

	search := call("ima_search_notes", map[string]any{"search_type": 1, "content": "连接器"})
	b, _ := json.Marshal(search.StructuredContent)
	if !strings.Contains(string(b), `"total_hit_num":1`) {
		t.Fatalf("search_notes did not normalize string integer: %s", b)
	}

	call("ima_list_notes", map[string]any{})
	call("ima_get_note", map[string]any{"note_id": "note-1"})
	call("ima_create_note", map[string]any{"content": "# contract test"})
	call("ima_append_note", map[string]any{"note_id": "note-1", "content": "append"})
	call("ima_append_note_markdown", map[string]any{"note_id": "note-1", "markdown": "markdown append"})
	call("ima_list_notebooks", map[string]any{})
	call("ima_get_knowledge_bases", map[string]any{"ids": []string{"kb-1"}})
	call("ima_list_knowledge", map[string]any{"knowledge_base_id": "kb-1"})
	call("ima_search_knowledge", map[string]any{"knowledge_base_id": "kb-1", "query": "连接器"})
	call("ima_search_knowledge_bases", map[string]any{"query": "测试"})
	call("ima_list_addable_knowledge_bases", map[string]any{})
	call("ima_check_duplicate_files", map[string]any{"knowledge_base_id": "kb-1", "params": []map[string]any{{"name": "probe.txt", "media_type": 13}}})
	call("ima_import_urls", map[string]any{"knowledge_base_id": "kb-1", "urls": []string{"https://example.com"}})
	call("ima_get_media_info", map[string]any{"media_id": "media-1"})
	call("ima_add_note_to_knowledge_base", map[string]any{"note_id": "note-1", "title": "连接器", "knowledge_base_id": "kb-1"})
	call("ima_preflight_file", map[string]any{"path": tmpPath})
	call("ima_upload_file", map[string]any{"path": tmpPath, "knowledge_base_id": "kb-1"})
	call("ima_add_url", map[string]any{"url": "https://mp.weixin.qq.com/s/test", "knowledge_base_id": "kb-1"})

	if uploader.Calls() != 1 {
		t.Fatalf("COS uploader calls = %d, want 1", uploader.Calls())
	}

	mu.Lock()
	defer mu.Unlock()
	requiredEndpoints := []string{
		"/openapi/note/v1/search_note",
		"/openapi/note/v1/list_note",
		"/openapi/note/v1/get_doc_content",
		"/openapi/note/v1/import_doc",
		"/openapi/note/v1/append_doc",
		"/openapi/note/v1/list_notebook",
		"/openapi/wiki/v1/get_knowledge_base",
		"/openapi/wiki/v1/get_knowledge_list",
		"/openapi/wiki/v1/search_knowledge",
		"/openapi/wiki/v1/search_knowledge_base",
		"/openapi/wiki/v1/get_addable_knowledge_base_list",
		"/openapi/wiki/v1/check_repeated_names",
		"/openapi/wiki/v1/import_urls",
		"/openapi/wiki/v1/get_media_info",
		"/openapi/wiki/v1/create_media",
		"/openapi/wiki/v1/add_knowledge",
	}
	for _, endpoint := range requiredEndpoints {
		if apiCalls[endpoint] == 0 {
			t.Errorf("endpoint %s was not exercised", endpoint)
		}
	}
}
