package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCreatesLogFileAndWrites(t *testing.T) {
	// create temp dir
	dir := t.TempDir()
	// initialize logger
	if err := Init("debug", dir); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	// write different level logs
	Debug("debug message %s", "d")
	Info("info message %s", "i")
	Warn("warn message %s", "w")
	Error("error message %s", "e")

	// check log file exists
	// Since logger uses current date in filename, match the prefix instead
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read log dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("no log files created in dir: %s", dir)
	}
	var content string
	for _, e := range entries {
		if !e.IsDir() {
			b, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				t.Fatalf("failed to read log file: %v", err)
			}
			content = string(b)
			break
		}
	}

	if !strings.Contains(content, "DEBUG") || !strings.Contains(content, "INFO") || !strings.Contains(content, "WARN") || !strings.Contains(content, "ERROR") {
		t.Fatalf("log file does not contain expected level strings; got: %s", content)
	}
}

func TestLevelFiltering(t *testing.T) {
	dir := t.TempDir()
	if err := Init("warn", dir); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	// Reset logger output capture by creating and reading file
	Debug("should not appear")
	Info("should not appear")
	Warn("should appear")
	Error("should appear")

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read log dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("no log files created in dir: %s", dir)
	}
	var content string
	for _, e := range entries {
		if !e.IsDir() {
			b, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				t.Fatalf("failed to read log file: %v", err)
			}
			content = string(b)
			break
		}
	}

	if strings.Contains(content, "DEBUG") || strings.Contains(content, "INFO") {
		t.Fatalf("log file contains filtered levels when level is WARN: %s", content)
	}
	if !strings.Contains(content, "WARN") || !strings.Contains(content, "ERROR") {
		t.Fatalf("log file missing WARN/ERROR entries: %s", content)
	}
}
