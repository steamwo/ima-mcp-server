package ima

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPreflightPNG(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "image.png")
	data := append([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}, make([]byte, 32)...)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	meta, err := PreflightFile(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if meta.MediaType != 9 || meta.ContentType != "image/png" || meta.FileExt != "png" {
		t.Fatalf("unexpected meta: %+v", meta)
	}
}

func TestPreflightRejectsImageExtensionMismatch(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "image.jpg")
	data := append([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}, make([]byte, 32)...)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := PreflightFile(path, ""); err == nil {
		t.Fatal("expected mismatch error")
	}
}

func TestPreflightRejectsOversizeText(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "large.txt")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(10*MiB + 1); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	if _, err := PreflightFile(path, ""); err == nil {
		t.Fatal("expected size error")
	}
}
