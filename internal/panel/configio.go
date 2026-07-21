package panel

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"gopkg.in/yaml.v3"
)

// writeMu serializes all config writes so concurrent panel submissions cannot
// interleave or race on temp-file names. Reads are safe without the lock
// because every write is an atomic temp-file-then-rename.
var writeMu sync.Mutex

// atomicWriteFile writes data to path atomically by writing a temp file in the
// same directory and renaming it over the target. Same-dir rename is atomic on
// POSIX and avoids cross-device EXDEV.
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		cleanup()
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}

// SaveYAML marshals v to YAML and writes it to path atomically.
func SaveYAML(path string, v any) error {
	writeMu.Lock()
	defer writeMu.Unlock()

	out, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal yaml: %w", err)
	}
	return atomicWriteFile(path, out, 0o644)
}

// SaveJSON marshals v to indented JSON and writes it to path atomically.
func SaveJSON(path string, v any) error {
	writeMu.Lock()
	defer writeMu.Unlock()

	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	out = append(out, '\n')
	return atomicWriteFile(path, out, 0o644)
}

var (
	reBlockComment = regexp.MustCompile(`/\*[\s\S]*?\*/`)
	reLineComment  = regexp.MustCompile(`(?m)//.*$`)
)

// stripComments removes // and /* */ comments from JSONC input so it can be
// parsed as plain JSON. Mirrors internal/config.stripJSONCComments.
func stripComments(s string) string {
	s = reBlockComment.ReplaceAllString(s, "")
	s = reLineComment.ReplaceAllString(s, "")
	return s
}

// loadJSONC reads a JSONC file, strips comments, and unmarshals into out.
func loadJSONC(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(stripComments(string(data))), out)
}
