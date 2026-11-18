package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/steamwo/ima-mcp-server/internal/ima"
)

func registerKnowledgeTools(server *mcp.Server, service *ima.Service) {
	mcp.AddTool(server, &mcp.Tool{Name: "ima_get_knowledge_bases", Description: "Get details for 1-20 IMA knowledge bases by ID."}, func(ctx context.Context, _ *mcp.CallToolRequest, in GetKnowledgeBasesInput) (*mcp.CallToolResult, ima.GetKnowledgeBasesData, error) {
		out, err := service.Client.GetKnowledgeBases(ctx, in.IDs)
		return nil, out, err
	})
	mcp.AddTool(server, &mcp.Tool{Name: "ima_list_knowledge", Description: "Browse files and folders in an IMA knowledge base. Omit folder_id for the root."}, func(ctx context.Context, _ *mcp.CallToolRequest, in ListKnowledgeInput) (*mcp.CallToolResult, ima.ListKnowledgeData, error) {
		if in.Limit == 0 {
			in.Limit = 20
		}
		out, err := service.Client.ListKnowledge(ctx, in.KnowledgeBaseID, in.FolderID, in.Cursor, in.Limit)
		return nil, out, err
	})
	mcp.AddTool(server, &mcp.Tool{Name: "ima_search_knowledge", Description: "Search content within one IMA knowledge base. Results may include both files and folders."}, func(ctx context.Context, _ *mcp.CallToolRequest, in SearchKnowledgeInput) (*mcp.CallToolResult, ima.SearchKnowledgeData, error) {
		out, err := service.Client.SearchKnowledge(ctx, in.KnowledgeBaseID, in.Query, in.Cursor)
		return nil, out, err
	})
	mcp.AddTool(server, &mcp.Tool{Name: "ima_search_knowledge_bases", Description: "Search IMA knowledge bases by name. An empty query lists available knowledge bases."}, func(ctx context.Context, _ *mcp.CallToolRequest, in SearchKnowledgeBasesInput) (*mcp.CallToolResult, ima.SearchKnowledgeBasesData, error) {
		if in.Limit == 0 {
			in.Limit = 20
		}
		out, err := service.Client.SearchKnowledgeBases(ctx, in.Query, in.Cursor, in.Limit)
		return nil, out, err
	})
	mcp.AddTool(server, &mcp.Tool{Name: "ima_list_addable_knowledge_bases", Description: "List knowledge bases where the current user can add content. Use when the user wants to add content but has not specified a target knowledge base."}, func(ctx context.Context, _ *mcp.CallToolRequest, in ListAddableKnowledgeBasesInput) (*mcp.CallToolResult, ima.AddableKnowledgeBasesData, error) {
		if in.Limit == 0 {
			in.Limit = 20
		}
		out, err := service.Client.ListAddableKnowledgeBases(ctx, in.Cursor, in.Limit)
		return nil, out, err
	})
	mcp.AddTool(server, &mcp.Tool{Name: "ima_check_duplicate_files", Description: "Check file names for duplicates before upload. File upload tools call this automatically; use this for batch planning."}, func(ctx context.Context, _ *mcp.CallToolRequest, in CheckDuplicatesInput) (*mcp.CallToolResult, ima.DuplicateCheckData, error) {
		out, err := service.Client.CheckRepeatedNames(ctx, in.KnowledgeBaseID, in.FolderID, in.Params)
		return nil, out, err
	})
	mcp.AddTool(server, &mcp.Tool{Name: "ima_import_urls", Description: "Import 1-10 web pages or WeChat articles into a knowledge base. For downloadable PDF/Word/PPT/etc URLs, prefer ima_add_url so the file upload safety gate is enforced."}, func(ctx context.Context, _ *mcp.CallToolRequest, in ImportURLsInput) (*mcp.CallToolResult, ima.ImportURLsData, error) {
		out, err := service.Client.ImportURLs(ctx, in.KnowledgeBaseID, in.FolderID, in.URLs)
		return nil, out, err
	})
	mcp.AddTool(server, &mcp.Tool{Name: "ima_get_media_info", Description: "Get access metadata for a knowledge-base media item. If media_type=11, notebook_ext_info.notebook_id can be passed to ima_get_note. For other media, url_info may contain a temporary URL and required headers."}, func(ctx context.Context, _ *mcp.CallToolRequest, in GetMediaInfoInput) (*mcp.CallToolResult, ima.MediaInfoData, error) {
		out, err := service.Client.GetMediaInfo(ctx, in.MediaID)
		return nil, out, err
	})
	mcp.AddTool(server, &mcp.Tool{Name: "ima_add_note_to_knowledge_base", Description: "Associate an existing IMA note with a knowledge base using media_type=11."}, func(ctx context.Context, _ *mcp.CallToolRequest, in AddNoteToKnowledgeInput) (*mcp.CallToolResult, ima.AddKnowledgeData, error) {
		out, err := service.Client.AddNoteKnowledge(ctx, in.NoteID, in.Title, in.KnowledgeBaseID, in.FolderID)
		return nil, out, err
	})
	mcp.AddTool(server, &mcp.Tool{Name: "ima_preflight_file", Description: "Validate a local file before IMA upload: supported type, size limits, and PNG/JPEG/WebP signature consistency. Does not modify or upload the file."}, func(_ context.Context, _ *mcp.CallToolRequest, in PreflightFileInput) (*mcp.CallToolResult, ima.FileMeta, error) {
		out, err := ima.PreflightFile(in.Path, in.ContentType)
		return nil, out, err
	})
	mcp.AddTool(server, &mcp.Tool{Name: "ima_upload_file", Description: "Safely upload a local file into an IMA knowledge base. Enforces preflight -> duplicate check -> create_media -> COS upload -> add_knowledge. Default duplicate policy never overwrites; status=duplicate means ask the user whether to keep both, then retry with keep_both if approved."}, func(ctx context.Context, _ *mcp.CallToolRequest, in UploadFileInput) (*mcp.CallToolResult, ima.UploadResult, error) {
		out, err := service.UploadFile(ctx, in.Path, in.ContentType, in.KnowledgeBaseID, in.FolderID, in.DuplicatePolicy, in.Password)
		return nil, out, err
	})
	mcp.AddTool(server, &mcp.Tool{Name: "ima_add_url", Description: "High-level URL ingestion. Public HTML/WeChat URLs go through import_urls; supported file URLs are downloaded without IMA credentials, preflighted, duplicate-checked, uploaded to COS, then added to the knowledge base. Private/local network URLs are blocked."}, func(ctx context.Context, _ *mcp.CallToolRequest, in AddURLInput) (*mcp.CallToolResult, ima.AddURLResult, error) {
		if in.URL == "" {
			return nil, ima.AddURLResult{}, fmt.Errorf("url is required")
		}
		out, err := service.AddURL(ctx, in.URL, in.KnowledgeBaseID, in.FolderID, in.DuplicatePolicy, in.Password)
		return nil, out, err
	})
}
