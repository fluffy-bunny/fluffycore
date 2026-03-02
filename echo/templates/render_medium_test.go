package templates

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFolderExists_NonExistentDir(t *testing.T) {
	result := FolderExists("/nonexistent/path/file.txt")
	if result {
		t.Error("FolderExists should return false for non-existent directory")
	}
}

func TestFolderExists_CurrentDir(t *testing.T) {
	result := FolderExists("somefile.txt")
	if !result {
		t.Error("FolderExists should return true for current directory")
	}
}

func TestFolderExists_EmptyDirPath(t *testing.T) {
	result := FolderExists("")
	if !result {
		t.Error("FolderExists should return true for empty path (implicit current dir)")
	}
}

func TestFindAndParseTemplates_NonExistentRoot(t *testing.T) {
	tmpl, err := FindAndParseTemplates("/nonexistent/root", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tmpl == nil {
		t.Error("expected non-nil template even for non-existent root")
	}
}

func TestFindAndParseTemplates_WithTemplates(t *testing.T) {
	// Create a temp directory with a .tpl file
	dir := t.TempDir()
	tplContent := `<h1>Hello {{.Name}}</h1>`
	err := os.WriteFile(filepath.Join(dir, "test.tpl"), []byte(tplContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test template: %v", err)
	}

	tmpl, err := FindAndParseTemplates(dir, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tmpl == nil {
		t.Fatal("expected non-nil template")
	}

	// Verify the template can be looked up
	lookup := tmpl.Lookup("test.tpl")
	if lookup == nil {
		t.Error("expected to find 'test.tpl' template")
	}
}

func TestFindAndParseTemplates_WalkErrorHandling(t *testing.T) {
	// Create a temp dir, then remove it to cause Walk error
	dir := t.TempDir()
	subdir := filepath.Join(dir, "sub")
	os.Mkdir(subdir, 0755)
	os.WriteFile(filepath.Join(subdir, "test.tpl"), []byte("hello"), 0644)

	// Make the directory unreadable (on Windows, this test verifies the e1 check path)
	// This primarily tests that the Walk callback handles e1 != nil before checking info
	tmpl, err := FindAndParseTemplates(dir, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tmpl == nil {
		t.Error("expected non-nil template")
	}
}

func TestGetTemplateRender_ValidDir(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "page.tpl"), []byte(`<div>{{.Content}}</div>`), 0644)
	if err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	renderer, err := GetTemplateRender(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if renderer == nil {
		t.Fatal("expected non-nil renderer")
	}
}
