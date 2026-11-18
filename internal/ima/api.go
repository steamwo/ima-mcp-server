package ima

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) SearchNotes(ctx context.Context, searchType, sortType int, title, content string, start, end int64) (SearchNotesData, error) {
	if start < 0 || end <= start || end-start > 20 {
		return SearchNotesData{}, fmt.Errorf("invalid note search range: require 0 <= start < end and end-start <= 20")
	}
	body := map[string]any{
		"search_type": searchType,
		"sort_type":   sortType,
		"start":       start,
		"end":         end,
	}
	query := map[string]string{}
	if title != "" {
		query["title"] = title
	}
	if content != "" {
		query["content"] = content
	}
	if len(query) > 0 {
		body["query_info"] = query
	}
	var out SearchNotesData
	return out, c.Call(ctx, "openapi/note/v1/search_note", body, &out)
}

func (c *Client) ListNotes(ctx context.Context, folderID string, sortType int, cursor string, limit int) (ListNotesData, error) {
	if limit < 1 || limit > 20 {
		return ListNotesData{}, fmt.Errorf("limit must be between 1 and 20")
	}
	body := map[string]any{"sort_type": sortType, "cursor": cursor, "limit": limit}
	if folderID != "" {
		body["folder_id"] = folderID
	}
	var out ListNotesData
	return out, c.Call(ctx, "openapi/note/v1/list_note", body, &out)
}

func (c *Client) GetNoteContent(ctx context.Context, noteID string, format int) (NoteContentData, error) {
	if noteID == "" {
		return NoteContentData{}, fmt.Errorf("note_id is required")
	}
	var out NoteContentData
	return out, c.Call(ctx, "openapi/note/v1/get_doc_content", map[string]any{
		"note_id": noteID, "target_content_format": format,
	}, &out)
}

func (c *Client) CreateNote(ctx context.Context, content, folderID, folderName string) (NoteWriteData, error) {
	if strings.TrimSpace(content) == "" {
		return NoteWriteData{}, fmt.Errorf("content must not be empty")
	}
	if strings.ToValidUTF8(content, "") != content {
		return NoteWriteData{}, fmt.Errorf("content is not valid UTF-8")
	}
	body := map[string]any{"content_format": 1, "content": content}
	if folderID != "" {
		body["folder_id"] = folderID
	}
	if folderName != "" {
		body["folder_name"] = folderName
	}
	var out NoteWriteData
	return out, c.Call(ctx, "openapi/note/v1/import_doc", body, &out)
}

func (c *Client) AppendNote(ctx context.Context, noteID, content string) (NoteWriteData, error) {
	if noteID == "" {
		return NoteWriteData{}, fmt.Errorf("note_id is required")
	}
	if strings.TrimSpace(content) == "" {
		return NoteWriteData{}, fmt.Errorf("content must not be empty")
	}
	if strings.ToValidUTF8(content, "") != content {
		return NoteWriteData{}, fmt.Errorf("content is not valid UTF-8")
	}
	var out NoteWriteData
	return out, c.Call(ctx, "openapi/note/v1/append_doc", map[string]any{
		"note_id": noteID, "content_format": 1, "content": content,
	}, &out)
}

func (c *Client) ListNotebooks(ctx context.Context, cursor string, limit int, version string) (ListNotebooksData, error) {
	if limit < 1 || limit > 20 {
		return ListNotebooksData{}, fmt.Errorf("limit must be between 1 and 20")
	}
	body := map[string]any{"cursor": cursor, "limit": limit}
	if version != "" {
		body["version"] = version
	}
	var out ListNotebooksData
	return out, c.Call(ctx, "openapi/note/v1/list_notebook", body, &out)
}

func (c *Client) GetKnowledgeBases(ctx context.Context, ids []string) (GetKnowledgeBasesData, error) {
	if len(ids) < 1 || len(ids) > 20 {
		return GetKnowledgeBasesData{}, fmt.Errorf("ids must contain between 1 and 20 items")
	}
	var out GetKnowledgeBasesData
	return out, c.Call(ctx, "openapi/wiki/v1/get_knowledge_base", map[string]any{"ids": ids}, &out)
}

func (c *Client) ListKnowledge(ctx context.Context, knowledgeBaseID, folderID, cursor string, limit int) (ListKnowledgeData, error) {
	if knowledgeBaseID == "" {
		return ListKnowledgeData{}, fmt.Errorf("knowledge_base_id is required")
	}
	if limit < 1 || limit > 50 {
		return ListKnowledgeData{}, fmt.Errorf("limit must be between 1 and 50")
	}
	body := map[string]any{"knowledge_base_id": knowledgeBaseID, "cursor": cursor, "limit": limit}
	if folderID != "" {
		body["folder_id"] = folderID
	}
	var out ListKnowledgeData
	return out, c.Call(ctx, "openapi/wiki/v1/get_knowledge_list", body, &out)
}

func (c *Client) SearchKnowledge(ctx context.Context, knowledgeBaseID, query, cursor string) (SearchKnowledgeData, error) {
	if knowledgeBaseID == "" {
		return SearchKnowledgeData{}, fmt.Errorf("knowledge_base_id is required")
	}
	var out SearchKnowledgeData
	return out, c.Call(ctx, "openapi/wiki/v1/search_knowledge", map[string]any{
		"query": query, "cursor": cursor, "knowledge_base_id": knowledgeBaseID,
	}, &out)
}

func (c *Client) SearchKnowledgeBases(ctx context.Context, query, cursor string, limit int) (SearchKnowledgeBasesData, error) {
	if limit < 1 || limit > 20 {
		return SearchKnowledgeBasesData{}, fmt.Errorf("limit must be between 1 and 20")
	}
	var out SearchKnowledgeBasesData
	return out, c.Call(ctx, "openapi/wiki/v1/search_knowledge_base", map[string]any{
		"query": query, "cursor": cursor, "limit": limit,
	}, &out)
}

func (c *Client) ListAddableKnowledgeBases(ctx context.Context, cursor string, limit int) (AddableKnowledgeBasesData, error) {
	if limit < 1 || limit > 50 {
		return AddableKnowledgeBasesData{}, fmt.Errorf("limit must be between 1 and 50")
	}
	var out AddableKnowledgeBasesData
	return out, c.Call(ctx, "openapi/wiki/v1/get_addable_knowledge_base_list", map[string]any{
		"cursor": cursor, "limit": limit,
	}, &out)
}

type DuplicateParam struct {
	Name      string `json:"name"`
	MediaType int    `json:"media_type"`
}

func (c *Client) CheckRepeatedNames(ctx context.Context, knowledgeBaseID, folderID string, params []DuplicateParam) (DuplicateCheckData, error) {
	if knowledgeBaseID == "" {
		return DuplicateCheckData{}, fmt.Errorf("knowledge_base_id is required")
	}
	if len(params) < 1 || len(params) > 2000 {
		return DuplicateCheckData{}, fmt.Errorf("params must contain between 1 and 2000 items")
	}
	body := map[string]any{"knowledge_base_id": knowledgeBaseID, "params": params}
	if folderID != "" {
		body["folder_id"] = folderID
	}
	var out DuplicateCheckData
	return out, c.Call(ctx, "openapi/wiki/v1/check_repeated_names", body, &out)
}

func (c *Client) ImportURLs(ctx context.Context, knowledgeBaseID, folderID string, urls []string) (ImportURLsData, error) {
	if knowledgeBaseID == "" {
		return ImportURLsData{}, fmt.Errorf("knowledge_base_id is required")
	}
	if folderID == "" {
		folderID = knowledgeBaseID
	}
	if len(urls) < 1 || len(urls) > 10 {
		return ImportURLsData{}, fmt.Errorf("urls must contain between 1 and 10 items")
	}
	for _, u := range urls {
		if strings.TrimSpace(u) == "" {
			return ImportURLsData{}, fmt.Errorf("urls must not contain empty values")
		}
	}
	var out ImportURLsData
	return out, c.Call(ctx, "openapi/wiki/v1/import_urls", map[string]any{
		"knowledge_base_id": knowledgeBaseID, "folder_id": folderID, "urls": urls,
	}, &out)
}

func (c *Client) GetMediaInfo(ctx context.Context, mediaID string) (MediaInfoData, error) {
	if mediaID == "" {
		return MediaInfoData{}, fmt.Errorf("media_id is required")
	}
	var out MediaInfoData
	return out, c.Call(ctx, "openapi/wiki/v1/get_media_info", map[string]any{"media_id": mediaID}, &out)
}

func (c *Client) CreateMedia(ctx context.Context, fileName string, fileSize int64, contentType, knowledgeBaseID, fileExt string) (CreateMediaData, error) {
	var out CreateMediaData
	return out, c.Call(ctx, "openapi/wiki/v1/create_media", map[string]any{
		"file_name": fileName, "file_size": fileSize, "content_type": contentType,
		"knowledge_base_id": knowledgeBaseID, "file_ext": fileExt,
	}, &out)
}

func (c *Client) AddFileKnowledge(ctx context.Context, mediaType int, mediaID, title, knowledgeBaseID, folderID string, fileInfo map[string]any) (AddKnowledgeData, error) {
	body := map[string]any{
		"media_type": mediaType, "media_id": mediaID, "title": title,
		"knowledge_base_id": knowledgeBaseID, "file_info": fileInfo,
	}
	if folderID != "" {
		body["folder_id"] = folderID
	}
	var out AddKnowledgeData
	return out, c.Call(ctx, "openapi/wiki/v1/add_knowledge", body, &out)
}

func (c *Client) AddNoteKnowledge(ctx context.Context, noteID, title, knowledgeBaseID, folderID string) (AddKnowledgeData, error) {
	if noteID == "" || title == "" || knowledgeBaseID == "" {
		return AddKnowledgeData{}, fmt.Errorf("note_id, title and knowledge_base_id are required")
	}
	body := map[string]any{
		"media_type":        11,
		"note_info":         map[string]string{"content_id": noteID},
		"title":             title,
		"knowledge_base_id": knowledgeBaseID,
	}
	if folderID != "" {
		body["folder_id"] = folderID
	}
	var out AddKnowledgeData
	return out, c.Call(ctx, "openapi/wiki/v1/add_knowledge", body, &out)
}
