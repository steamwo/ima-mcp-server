package ima

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
)

const (
	MiB              = 1024 * 1024
	MaxUploadSize    = 200 * MiB
	MaxNoteImageSize = 3 * MiB
)

type fileType struct {
	MediaType   int
	ContentType string
	Extension   string
}

var extensionTypes = map[string]fileType{
	"pdf":      {1, "application/pdf", "pdf"},
	"doc":      {3, "application/msword", "doc"},
	"docx":     {3, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "docx"},
	"ppt":      {4, "application/vnd.ms-powerpoint", "ppt"},
	"pptx":     {4, "application/vnd.openxmlformats-officedocument.presentationml.presentation", "pptx"},
	"xls":      {5, "application/vnd.ms-excel", "xls"},
	"xlsx":     {5, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "xlsx"},
	"csv":      {5, "text/csv", "csv"},
	"md":       {7, "text/markdown", "md"},
	"markdown": {7, "text/markdown", "markdown"},
	"png":      {9, "image/png", "png"},
	"jpg":      {9, "image/jpeg", "jpg"},
	"jpeg":     {9, "image/jpeg", "jpeg"},
	"webp":     {9, "image/webp", "webp"},
	"txt":      {13, "text/plain", "txt"},
	"xmind":    {14, "application/x-xmind", "xmind"},
	"mp3":      {15, "audio/mpeg", "mp3"},
	"m4a":      {15, "audio/x-m4a", "m4a"},
	"wav":      {15, "audio/wav", "wav"},
	"aac":      {15, "audio/aac", "aac"},
	"html":     {20, "text/html", "html"},
	"epub":     {21, "application/epub+zip", "epub"},
}

var contentTypes = map[string]fileType{
	"application/pdf":    {1, "application/pdf", "pdf"},
	"application/msword": {3, "application/msword", "doc"},
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   {3, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "docx"},
	"application/vnd.ms-powerpoint":                                             {4, "application/vnd.ms-powerpoint", "ppt"},
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": {4, "application/vnd.openxmlformats-officedocument.presentationml.presentation", "pptx"},
	"application/vnd.ms-excel":                                                  {5, "application/vnd.ms-excel", "xls"},
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         {5, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "xlsx"},
	"text/csv":                       {5, "text/csv", "csv"},
	"text/markdown":                  {7, "text/markdown", "md"},
	"text/x-markdown":                {7, "text/x-markdown", "md"},
	"application/md":                 {7, "application/md", "md"},
	"application/markdown":           {7, "application/markdown", "md"},
	"image/png":                      {9, "image/png", "png"},
	"image/jpeg":                     {9, "image/jpeg", "jpg"},
	"image/webp":                     {9, "image/webp", "webp"},
	"text/plain":                     {13, "text/plain", "txt"},
	"application/x-xmind":            {14, "application/x-xmind", "xmind"},
	"application/vnd.xmind.workbook": {14, "application/vnd.xmind.workbook", "xmind"},
	"application/zip":                {14, "application/zip", "xmind"},
	"audio/mpeg":                     {15, "audio/mpeg", "mp3"},
	"audio/x-m4a":                    {15, "audio/x-m4a", "m4a"},
	"audio/wav":                      {15, "audio/wav", "wav"},
	"audio/aac":                      {15, "audio/aac", "aac"},
	"text/html":                      {20, "text/html", "html"},
	"application/epub+zip":           {21, "application/epub+zip", "epub"},
}

var unsupportedVideoExt = map[string]bool{
	"mp4": true, "avi": true, "mov": true, "mkv": true, "wmv": true, "flv": true,
	"webm": true, "m4v": true, "rmvb": true, "rm": true, "3gp": true,
}

var unsupportedVideoContentType = map[string]bool{
	"video/mp4": true, "video/x-msvideo": true, "video/quicktime": true,
	"video/x-matroska": true, "video/x-ms-wmv": true, "video/x-flv": true, "video/webm": true,
}

type FileMeta struct {
	Path        string `json:"path"`
	FileName    string `json:"file_name"`
	FileExt     string `json:"file_ext"`
	FileSize    int64  `json:"file_size"`
	MediaType   int    `json:"media_type"`
	ContentType string `json:"content_type"`
	ModTimeUnix int64  `json:"last_modify_time"`
}

func PreflightFile(path, contentType string) (FileMeta, error) {
	return PreflightFileAs(path, filepath.Base(path), contentType)
}

func PreflightFileAs(path, fileName, contentType string) (FileMeta, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return FileMeta{}, err
	}
	stat, err := os.Stat(abs)
	if err != nil {
		return FileMeta{}, fmt.Errorf("file not found: %w", err)
	}
	if !stat.Mode().IsRegular() {
		return FileMeta{}, fmt.Errorf("path is not a regular file: %s", abs)
	}
	fileName = sanitizeFileName(fileName)
	if fileName == "" {
		return FileMeta{}, fmt.Errorf("file name is empty")
	}

	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fileName), "."))
	ct := normalizeContentType(contentType)
	if unsupportedVideoExt[ext] || unsupportedVideoContentType[ct] {
		return FileMeta{}, fmt.Errorf("video files are not supported by the IMA OpenAPI upload flow; use the IMA desktop app")
	}
	if ext == "mhtml" || ct == "application/xhtml+xml" {
		return FileMeta{}, fmt.Errorf("web pages must be added with ima_import_urls, not uploaded as files")
	}

	var typ fileType
	var ok bool
	if ct != "" {
		typ, ok = contentTypes[ct]
	}
	if !ok && ext != "" {
		typ, ok = extensionTypes[ext]
	}
	if !ok {
		if ct != "" {
			return FileMeta{}, fmt.Errorf("unsupported content type %q and file extension %q", ct, ext)
		}
		return FileMeta{}, fmt.Errorf("unsupported file extension %q", ext)
	}
	if ext == "" {
		ext = typ.Extension
	}

	limit := int64(MaxUploadSize)
	switch typ.MediaType {
	case 5, 7, 13, 14, 20:
		limit = 10 * MiB
	case 9:
		limit = 30 * MiB
	case 21:
		limit = 50 * MiB
	}
	if stat.Size() > int64(limit) {
		return FileMeta{}, fmt.Errorf("file size %d bytes exceeds %d byte limit for this file type", stat.Size(), limit)
	}

	if typ.MediaType == 9 {
		f, err := os.Open(abs)
		if err != nil {
			return FileMeta{}, err
		}
		defer f.Close()
		header := make([]byte, 16)
		n, _ := io.ReadFull(f, header)
		detected := detectImageType(header[:n])
		if detected == "" {
			return FileMeta{}, fmt.Errorf("invalid or unsupported image signature; PNG, JPEG and WebP are supported")
		}
		if detected != typ.ContentType {
			return FileMeta{}, fmt.Errorf("image signature %s does not match declared type %s", detected, typ.ContentType)
		}
		if extType, exists := extensionTypes[ext]; exists && extType.MediaType == 9 && extType.ContentType != detected {
			return FileMeta{}, fmt.Errorf("image extension .%s does not match actual signature %s", ext, detected)
		}
	}

	return FileMeta{
		Path: abs, FileName: fileName, FileExt: ext, FileSize: stat.Size(), MediaType: typ.MediaType,
		ContentType: typ.ContentType, ModTimeUnix: stat.ModTime().Unix(),
	}, nil
}

func normalizeContentType(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}
	if mt, _, err := mime.ParseMediaType(value); err == nil {
		return strings.ToLower(mt)
	}
	if i := strings.IndexByte(value, ';'); i >= 0 {
		value = value[:i]
	}
	return strings.TrimSpace(value)
}

func detectImageType(b []byte) string {
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	if len(b) >= len(png) && bytes.Equal(b[:len(png)], png) {
		return "image/png"
	}
	if len(b) >= 3 && b[0] == 0xff && b[1] == 0xd8 && b[2] == 0xff {
		return "image/jpeg"
	}
	if len(b) >= 12 && string(b[:4]) == "RIFF" && string(b[8:12]) == "WEBP" {
		return "image/webp"
	}
	return ""
}

func sanitizeFileName(name string) string {
	name = strings.TrimSpace(filepath.Base(strings.ReplaceAll(name, "\x00", "")))
	if name == "." || name == string(filepath.Separator) {
		return ""
	}
	if len(name) > 1024 {
		name = name[:1024]
	}
	return name
}
