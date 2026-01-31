package internal_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func projectRoot(t *testing.T) string {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "go", "list", "-m", "-f", "{{.Dir}}")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to get module root: %v", err)
	}
	return strings.TrimSpace(string(out))
}

func TestProjectStructure(t *testing.T) {
	root := projectRoot(t)

	expectedDirs := []string{
		"cmd/generator",
		"internal/parser",
		"internal/content",
		"internal/site",
		"internal/feed",
		"content",
		"dist",
	}

	for _, dir := range expectedDirs {
		t.Run("directory_"+dir, func(t *testing.T) {
			path := filepath.Join(root, dir)
			info, err := os.Stat(path)
			if os.IsNotExist(err) {
				t.Errorf("expected directory %s to exist", dir)
				return
			}
			if err != nil {
				t.Errorf("error checking directory %s: %v", dir, err)
				return
			}
			if !info.IsDir() {
				t.Errorf("%s is not a directory", dir)
			}
		})
	}
}

func TestMakefileExists(t *testing.T) {
	root := projectRoot(t)
	path := filepath.Join(root, "Makefile")

	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		t.Error("Makefile should exist")
		return
	}
	if err != nil {
		t.Errorf("error checking Makefile: %v", err)
		return
	}
	if info.IsDir() {
		t.Error("Makefile should be a file, not a directory")
	}
}

func TestGeneratorEntryPoint(t *testing.T) {
	root := projectRoot(t)
	path := filepath.Join(root, "cmd/generator/main.go")

	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		t.Error("cmd/generator/main.go should exist")
		return
	}
	if err != nil {
		t.Errorf("error checking cmd/generator/main.go: %v", err)
		return
	}
	if info.IsDir() {
		t.Error("cmd/generator/main.go should be a file, not a directory")
	}
}
