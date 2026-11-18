package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/steamwo/ima-mcp-server/internal/ima"
)

type SearchNotesInput struct {
	SearchType int    `json:"search_type,omitempty" jsonschema:"Search mode: 0=title search (default), 1=content search"`
	SortType   int    `json:"sort_type,omitempty" jsonschema:"Sort mode: 0=modify time (default), 1=create time, 2=title, 3=size"`
	Title      string `json:"title,omitempty" jsonschema:"Title search keyword"`
	Content    string `json:"content,omitempty" jsonschema:"Content search keyword"`
	Start      int64  `json:"start,omitempty" jsonschema:"Offset start, default 0"`
	End        int64  `json:"end,omitempty" jsonschema:"Offset end; default 20 and must be at most 20 greater than start"`
}

type ListNotesInput struct {
	FolderID string `json:"folder_id,omitempty" jsonschema:"Notebook/folder ID; omit to list all notes"`
	SortType int    `json:"sort_type,omitempty" jsonschema:"Sort mode: 0=modify time (default), 1=create time, 2=title"`
	Cursor   string `json:"cursor,omitempty" jsonschema:"Cursor; first page is empty string"`
	Limit    int    `json:"limit,omitempty" jsonschema:"Page size 1-20; default 20"`
}

type GetNoteInput struct {
	NoteID string `json:"note_id" jsonschema:"Internal IMA note ID"`
	Format int    `json:"format,omitempty" jsonschema:"Output format: 0=plain text (recommended/default), 2=JSON; Markdown read format is not supported by IMA"`
}

type CreateNoteInput struct {
	Content    string `json:"content" jsonschema:"UTF-8 Markdown content; content_format is fixed to Markdown"`
	FolderID   string `json:"folder_id,omitempty" jsonschema:"Optional target notebook ID"`
	FolderName string `json:"folder_name,omitempty" jsonschema:"Optional target notebook name"`
}

type AppendNoteInput struct {
	NoteID  string `json:"note_id" jsonschema:"Target note ID. Only call after the user clearly selected the existing note."`
	Content string `json:"content" jsonschema:"UTF-8 Markdown to append"`
}

type AppendMarkdownInput struct {
	NoteID   string `json:"note_id" jsonschema:"Target note ID. This mutates an existing note and is not reversible."`
	Markdown string `json:"markdown" jsonschema:"Markdown. Images may use absolute local paths, file:// URIs, base64 data URLs, or public http(s) URLs."`
	DryRun   bool   `json:"dry_run,omitempty" jsonschema:"If true, preflight all images and return batches without modifying the note"`
}

type ListNotebooksInput struct {
	Cursor  string `json:"cursor,omitempty" jsonschema:"First page cursor is \"0\"; defaults to \"0\""`
	Limit   int    `json:"limit,omitempty" jsonschema:"Page size 1-20; default 20"`
	Version string `json:"version,omitempty" jsonschema:"Optional previous next_version for incremental update checks"`
}

func registerNoteTools(server *mcp.Server, service *ima.Service) {
	mcp.AddTool(server, &mcp.Tool{Name: "ima_search_notes", Description: "Search IMA personal notes by title or content. Returns internal note IDs for chaining; do not expose IDs to end users unless they ask."}, func(ctx context.Context, _ *mcp.CallToolRequest, in SearchNotesInput) (*mcp.CallToolResult, ima.SearchNotesData, error) {
		if in.End == 0 {
			in.End = in.Start + 20
		}
		out, err := service.Client.SearchNotes(ctx, in.SearchType, in.SortType, in.Title, in.Content, in.Start, in.End)
		return nil, out, err
	})
	mcp.AddTool(server, &mcp.Tool{Name: "ima_list_notes", Description: "List IMA notes, optionally within one notebook. First cursor is an empty string."}, func(ctx context.Context, _ *mcp.CallToolRequest, in ListNotesInput) (*mcp.CallToolResult, ima.ListNotesData, error) {
		if in.Limit == 0 {
			in.Limit = 20
		}
		out, err := service.Client.ListNotes(ctx, in.FolderID, in.SortType, in.Cursor, in.Limit)
		return nil, out, err
	})
	mcp.AddTool(server, &mcp.Tool{Name: "ima_get_note", Description: "Read the content of a note owned by the current IMA account. Prefer plain text format (0)."}, func(ctx context.Context, _ *mcp.CallToolRequest, in GetNoteInput) (*mcp.CallToolResult, ima.NoteContentData, error) {
		out, err := service.Client.GetNoteContent(ctx, in.NoteID, in.Format)
		return nil, out, err
	})
	mcp.AddTool(server, &mcp.Tool{Name: "ima_create_note", Description: "Create a new IMA note from UTF-8 Markdown. Use only when the user intends to create a new note, not when they intend to append to an existing note."}, func(ctx context.Context, _ *mcp.CallToolRequest, in CreateNoteInput) (*mcp.CallToolResult, ima.NoteWriteData, error) {
		out, err := service.Client.CreateNote(ctx, in.Content, in.FolderID, in.FolderName)
		return nil, out, err
	})
	mcp.AddTool(server, &mcp.Tool{Name: "ima_append_note", Description: "Append UTF-8 Markdown to an existing note. This is a sensitive irreversible write: call only after the target note is unambiguous."}, func(ctx context.Context, _ *mcp.CallToolRequest, in AppendNoteInput) (*mcp.CallToolResult, ima.NoteWriteData, error) {
		out, err := service.Client.AppendNote(ctx, in.NoteID, in.Content)
		return nil, out, err
	})
	mcp.AddTool(server, &mcp.Tool{Name: "ima_append_note_markdown", Description: "Append mixed Markdown text and images to an existing note. All images are preflighted first; PNG/JPEG/WebP must each be smaller than 3 MiB. Text and images are then appended in original order, one image per batch."}, func(ctx context.Context, _ *mcp.CallToolRequest, in AppendMarkdownInput) (*mcp.CallToolResult, ima.AppendMarkdownResult, error) {
		out, err := service.AppendMarkdown(ctx, in.NoteID, in.Markdown, in.DryRun)
		return nil, out, err
	})
	mcp.AddTool(server, &mcp.Tool{Name: "ima_list_notebooks", Description: "List IMA notebooks/folders. The first cursor is \"0\"."}, func(ctx context.Context, _ *mcp.CallToolRequest, in ListNotebooksInput) (*mcp.CallToolResult, ima.ListNotebooksData, error) {
		if in.Cursor == "" {
			in.Cursor = "0"
		}
		if in.Limit == 0 {
			in.Limit = 20
		}
		out, err := service.Client.ListNotebooks(ctx, in.Cursor, in.Limit, in.Version)
		return nil, out, err
	})
}
