package ima

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Service struct {
	Client   *Client
	Uploader Uploader
	Now      func() time.Time
}

type UploadResult struct {
	Status           string `json:"status"`
	FileName         string `json:"file_name"`
	MediaID          string `json:"media_id,omitempty"`
	Duplicate        bool   `json:"duplicate,omitempty"`
	OriginalFileName string `json:"original_file_name,omitempty"`
	KnowledgeBaseID  string `json:"knowledge_base_id,omitempty"`
	FolderID         string `json:"folder_id,omitempty"`
}

type AddURLResult struct {
	Mode       string         `json:"mode"`
	ImportData ImportURLsData `json:"import_data,omitempty"`
	Upload     *UploadResult  `json:"upload,omitempty"`
}

type AppendMarkdownResult struct {
	NoteID           string          `json:"note_id"`
	Batches          []MarkdownBatch `json:"batches"`
	CompletedBatches int             `json:"completed_batches"`
	DryRun           bool            `json:"dry_run"`
}

func NewService(client *Client, uploader Uploader) *Service {
	if uploader == nil {
		uploader = &COSUploader{}
	}
	return &Service{Client: client, Uploader: uploader, Now: time.Now}
}

func (s *Service) UploadFile(ctx context.Context, path, contentType, knowledgeBaseID, folderID, duplicatePolicy, password string) (UploadResult, error) {
	meta, err := PreflightFile(path, contentType)
	if err != nil {
		return UploadResult{}, err
	}
	return s.uploadPreparedFile(ctx, meta, knowledgeBaseID, folderID, duplicatePolicy, password)
}

func (s *Service) uploadPreparedFile(ctx context.Context, meta FileMeta, knowledgeBaseID, folderID, duplicatePolicy, password string) (UploadResult, error) {
	if knowledgeBaseID == "" {
		return UploadResult{}, fmt.Errorf("knowledge_base_id is required")
	}
	if duplicatePolicy == "" {
		duplicatePolicy = "error"
	}
	if duplicatePolicy != "error" && duplicatePolicy != "keep_both" {
		return UploadResult{}, fmt.Errorf("duplicate_policy must be error or keep_both")
	}

	originalName := meta.FileName
	check, err := s.Client.CheckRepeatedNames(ctx, knowledgeBaseID, folderID, []DuplicateParam{{Name: meta.FileName, MediaType: meta.MediaType}})
	if err != nil {
		return UploadResult{}, err
	}
	if isDuplicate(check, meta.FileName) {
		if duplicatePolicy == "error" {
			return UploadResult{Status: "duplicate", FileName: meta.FileName, Duplicate: true, KnowledgeBaseID: knowledgeBaseID, FolderID: folderID}, nil
		}
		meta.FileName, err = s.uniqueTimestampedName(ctx, meta.FileName, meta.MediaType, knowledgeBaseID, folderID)
		if err != nil {
			return UploadResult{}, err
		}
	}

	created, err := s.Client.CreateMedia(ctx, meta.FileName, meta.FileSize, meta.ContentType, knowledgeBaseID, meta.FileExt)
	if err != nil {
		return UploadResult{}, err
	}
	if created.MediaID == "" || created.COSCredential.COSKey == "" {
		return UploadResult{}, fmt.Errorf("IMA create_media returned incomplete upload metadata")
	}
	if err := s.Uploader.Upload(ctx, meta.Path, meta.ContentType, created.COSCredential); err != nil {
		return UploadResult{}, err
	}

	fileInfo := map[string]any{
		"cos_key":          created.COSCredential.COSKey,
		"file_size":        meta.FileSize,
		"last_modify_time": meta.ModTimeUnix,
		"file_name":        meta.FileName,
	}
	if password != "" {
		fileInfo["password"] = password
	}
	added, err := s.Client.AddFileKnowledge(ctx, meta.MediaType, created.MediaID, meta.FileName, knowledgeBaseID, folderID, fileInfo)
	if err != nil {
		return UploadResult{}, err
	}
	mediaID := added.MediaID
	if mediaID == "" {
		mediaID = created.MediaID
	}
	result := UploadResult{Status: "uploaded", FileName: meta.FileName, MediaID: mediaID, KnowledgeBaseID: knowledgeBaseID, FolderID: folderID}
	if meta.FileName != originalName {
		result.OriginalFileName = originalName
	}
	return result, nil
}

func (s *Service) uniqueTimestampedName(ctx context.Context, name string, mediaType int, knowledgeBaseID, folderID string) (string, error) {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	stamp := s.Now().Format("20060102150405")
	for i := 0; i < 100; i++ {
		suffix := "_" + stamp
		if i > 0 {
			suffix += "_" + strconv.Itoa(i+1)
		}
		candidate := base + suffix + ext
		check, err := s.Client.CheckRepeatedNames(ctx, knowledgeBaseID, folderID, []DuplicateParam{{Name: candidate, MediaType: mediaType}})
		if err != nil {
			return "", err
		}
		if !isDuplicate(check, candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("could not find a unique duplicate filename after 100 attempts")
}

func isDuplicate(data DuplicateCheckData, name string) bool {
	for _, result := range data.Results {
		if result.Name == name && result.IsRepeated {
			return true
		}
	}
	return false
}

func (s *Service) AppendMarkdown(ctx context.Context, noteID, markdown string, dryRun bool) (AppendMarkdownResult, error) {
	if noteID == "" {
		return AppendMarkdownResult{}, fmt.Errorf("note_id is required")
	}
	batches, err := CreateMarkdownBatches(ctx, markdown)
	if err != nil {
		return AppendMarkdownResult{}, err
	}
	result := AppendMarkdownResult{NoteID: noteID, Batches: batches, DryRun: dryRun}
	if dryRun {
		return result, nil
	}
	for i, batch := range batches {
		if _, err := s.Client.AppendNote(ctx, noteID, batch.Content); err != nil {
			result.CompletedBatches = i
			return result, fmt.Errorf("batch %d/%d (%s) failed after %d completed batches: %w", i+1, len(batches), batch.Type, i, err)
		}
		result.CompletedBatches = i + 1
	}
	return result, nil
}

func (s *Service) AddURL(ctx context.Context, sourceURL, knowledgeBaseID, folderID, duplicatePolicy, password string) (AddURLResult, error) {
	u, err := url.Parse(sourceURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return AddURLResult{}, fmt.Errorf("URL must use http or https")
	}
	lowerHost := strings.ToLower(u.Hostname())
	if lowerHost == "www.bilibili.com" && strings.HasPrefix(u.Path, "/video/") {
		return AddURLResult{}, fmt.Errorf("Bilibili video pages are not supported by this IMA OpenAPI flow; use the IMA desktop app")
	}
	if (lowerHost == "www.youtube.com" || lowerHost == "youtube.com") && u.Path == "/watch" {
		return AddURLResult{}, fmt.Errorf("YouTube video pages are not supported by this IMA OpenAPI flow; use the IMA desktop app")
	}
	if lowerHost == "mp.weixin.qq.com" && (strings.HasPrefix(u.Path, "/s/") || u.Path == "/s") {
		data, err := s.Client.ImportURLs(ctx, knowledgeBaseID, folderID, []string{sourceURL})
		return AddURLResult{Mode: "url", ImportData: data}, err
	}

	client := newSafeHTTPClient(5 * time.Minute)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return AddURLResult{}, err
	}
	req.Header.Set("Accept", "*/*")
	resp, err := client.Do(req)
	if err != nil {
		return AddURLResult{}, fmt.Errorf("URL download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return AddURLResult{}, fmt.Errorf("URL request failed with HTTP %d", resp.StatusCode)
	}
	ct := normalizeContentType(resp.Header.Get("Content-Type"))
	if ct == "text/html" || ct == "application/xhtml+xml" {
		// import_urls performs the actual fetch and type recognition inside IMA.
		data, err := s.Client.ImportURLs(ctx, knowledgeBaseID, folderID, []string{sourceURL})
		return AddURLResult{Mode: "url", ImportData: data}, err
	}

	name := fileNameFromResponse(resp, u, ct)
	temp, err := os.CreateTemp("", "ima-url-*")
	if err != nil {
		return AddURLResult{}, err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)

	written, copyErr := io.Copy(temp, io.LimitReader(resp.Body, MaxUploadSize+1))
	closeErr := temp.Close()
	if copyErr != nil {
		return AddURLResult{}, copyErr
	}
	if closeErr != nil {
		return AddURLResult{}, closeErr
	}
	if written > MaxUploadSize {
		return AddURLResult{}, fmt.Errorf("downloaded file exceeds the 200 MiB maximum upload size")
	}
	meta, err := PreflightFileAs(tempPath, name, ct)
	if err != nil {
		return AddURLResult{}, err
	}
	result, err := s.uploadPreparedFile(ctx, meta, knowledgeBaseID, folderID, duplicatePolicy, password)
	if err != nil {
		return AddURLResult{}, err
	}
	return AddURLResult{Mode: "file", Upload: &result}, nil
}

func fileNameFromResponse(resp *http.Response, u *url.URL, contentType string) string {
	if disposition := resp.Header.Get("Content-Disposition"); disposition != "" {
		if _, params, err := mime.ParseMediaType(disposition); err == nil {
			if name := sanitizeFileName(params["filename"]); name != "" {
				return name
			}
		}
	}
	if base := sanitizeFileName(filepath.Base(u.Path)); base != "" && base != "/" && base != "." {
		if filepath.Ext(base) != "" {
			return base
		}
		if typ, ok := contentTypes[contentType]; ok {
			return base + "." + typ.Extension
		}
		return base
	}
	if typ, ok := contentTypes[contentType]; ok {
		return "download." + typ.Extension
	}
	return "download"
}
