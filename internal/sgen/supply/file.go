package supply

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/scnewma/sgen/internal/fsutil"
	"gopkg.in/yaml.v3"
)

type File struct {
	path string
}

func NewFileSupply(path string) (*File, error) {
	if !fsutil.Exists(path) {
		return nil, fmt.Errorf("file not found: %q", path)
	}
	return &File{path}, nil
}

func (s *File) Supply(_ context.Context) ([]map[string]string, error) {
	ext := filepath.Ext(s.path)
	if ext == "" {
		return nil, fmt.Errorf("cannot determine encoding for file %q because there is no extension", s.path)
	}

	contents, err := os.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("cannot read %q: %w", s.path, err)
	}

	var data []map[string]string
	// ext[1:] trims the leading "."
	switch ext[1:] {
	case "json":
		err = json.Unmarshal(contents, &data)
	case "yml", "yaml":
		err = yaml.Unmarshal(contents, &data)
	default:
		return nil, fmt.Errorf("unsupported file extension %q", ext)
	}
	if err != nil {
		return nil, fmt.Errorf("decoding %q: %w", s.path, err)
	}

	return data, nil
}

func (s *File) ShouldCache() bool {
	// we don't need to cache files because we would just be copying the data
	// anyway, it makes it less work to just read the source
	return false
}
