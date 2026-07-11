package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestValidationAndImage(t *testing.T) {
	if validate(Story{Type: "text", Text: "hello"}) != nil {
		t.Fatal("valid text story rejected")
	}
	if validate(Story{Type: "text", Text: "hello", Link: "javascript:alert(1)"}) == nil {
		t.Fatal("unsafe link accepted")
	}
	if validate(Story{Type: "text", Text: "hello", Link: "//evil.example"}) == nil {
		t.Fatal("protocol-relative link accepted")
	}
	dir := t.TempDir()
	png := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 504)...)
	ext, err := saveImage(dir, "test", bytes.NewReader(png), int64(len(png)))
	if err != nil || ext != ".png" {
		t.Fatalf("PNG rejected: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "test.png")); err != nil {
		t.Fatal("PNG was not saved")
	}
}
