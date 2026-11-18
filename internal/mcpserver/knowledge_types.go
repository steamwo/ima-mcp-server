package mcpserver

import "github.com/steamwo/ima-mcp-server/internal/ima"

type GetKnowledgeBasesInput struct {
	IDs []string `json:"ids" jsonschema:"1-20 knowledge base IDs"`
}

type ListKnowledgeInput struct {
	KnowledgeBaseID string `json:"knowledge_base_id" jsonschema:"Target knowledge base ID"`
	FolderID        string `json:"folder_id,omitempty" jsonschema:"Optional folder ID; omit for root"`
	Cursor          string `json:"cursor,omitempty" jsonschema:"First page cursor is empty string"`
	Limit           int    `json:"limit,omitempty" jsonschema:"Page size 1-50; default 20"`
}

type SearchKnowledgeInput struct {
	KnowledgeBaseID string `json:"knowledge_base_id" jsonschema:"Target knowledge base ID"`
	Query           string `json:"query" jsonschema:"Search keyword; results may include files and folders"`
	Cursor          string `json:"cursor,omitempty" jsonschema:"First page cursor is empty string"`
}

type SearchKnowledgeBasesInput struct {
	Query  string `json:"query,omitempty" jsonschema:"Knowledge base name keyword. Empty query lists knowledge bases."`
	Cursor string `json:"cursor,omitempty" jsonschema:"First page cursor is empty string"`
	Limit  int    `json:"limit,omitempty" jsonschema:"Page size 1-20; default 20"`
}

type ListAddableKnowledgeBasesInput struct {
	Cursor string `json:"cursor,omitempty" jsonschema:"First page cursor is empty string"`
	Limit  int    `json:"limit,omitempty" jsonschema:"Page size 1-50; default 20"`
}

type CheckDuplicatesInput struct {
	KnowledgeBaseID string               `json:"knowledge_base_id" jsonschema:"Target knowledge base ID"`
	FolderID        string               `json:"folder_id,omitempty" jsonschema:"Optional target folder ID"`
	Params          []ima.DuplicateParam `json:"params" jsonschema:"1-2000 file name/media type pairs"`
}

type ImportURLsInput struct {
	KnowledgeBaseID string   `json:"knowledge_base_id" jsonschema:"Target knowledge base ID"`
	FolderID        string   `json:"folder_id,omitempty" jsonschema:"Target folder ID; omit for root and the server will use knowledge_base_id"`
	URLs            []string `json:"urls" jsonschema:"1-10 web page or WeChat article URLs"`
}

type GetMediaInfoInput struct {
	MediaID string `json:"media_id" jsonschema:"IMA media ID returned by knowledge search/list tools"`
}

type AddNoteToKnowledgeInput struct {
	NoteID          string `json:"note_id" jsonschema:"Existing IMA note ID"`
	Title           string `json:"title" jsonschema:"Note title to display in the knowledge base"`
	KnowledgeBaseID string `json:"knowledge_base_id" jsonschema:"Target knowledge base ID"`
	FolderID        string `json:"folder_id,omitempty" jsonschema:"Optional target folder ID"`
}

type PreflightFileInput struct {
	Path        string `json:"path" jsonschema:"Local file path on the machine running this MCP server"`
	ContentType string `json:"content_type,omitempty" jsonschema:"Optional MIME type; recognized content type takes precedence over extension"`
}

type UploadFileInput struct {
	Path            string `json:"path" jsonschema:"Local file path on the machine running this MCP server"`
	ContentType     string `json:"content_type,omitempty" jsonschema:"Optional MIME type; useful when the file has no extension"`
	KnowledgeBaseID string `json:"knowledge_base_id" jsonschema:"Target knowledge base ID"`
	FolderID        string `json:"folder_id,omitempty" jsonschema:"Optional target folder ID; omit for root"`
	DuplicatePolicy string `json:"duplicate_policy,omitempty" jsonschema:"error (default) returns status=duplicate without writing; keep_both appends a timestamp and uploads. Overwrite is not supported."`
	Password        string `json:"password,omitempty" jsonschema:"Optional document password sent in file_info"`
}

type AddURLInput struct {
	URL             string `json:"url" jsonschema:"Public http(s) URL. HTML/WeChat pages use import_urls; supported file URLs are downloaded, preflighted and uploaded."`
	KnowledgeBaseID string `json:"knowledge_base_id" jsonschema:"Target knowledge base ID"`
	FolderID        string `json:"folder_id,omitempty" jsonschema:"Optional target folder ID"`
	DuplicatePolicy string `json:"duplicate_policy,omitempty" jsonschema:"For file URLs: error (default) or keep_both; overwrite is not supported"`
	Password        string `json:"password,omitempty" jsonschema:"Optional document password for file URLs"`
}
