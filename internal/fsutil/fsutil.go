package fsutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func EnsureDirExists(path string) error {
	return os.MkdirAll(path, 0755)
}

func WriteJSON(path string, v any) error {
	if err := EnsureDirExists(filepath.Dir(path)); err != nil {
		return fmt.Errorf("creating base directory: %w", err)
	}

	buf, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("encoding json data: %w", err)
	}
	return os.WriteFile(path, buf, 0755)
}

func ReadJSON(path string, v any) error {
	buf, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(buf, v)
}
