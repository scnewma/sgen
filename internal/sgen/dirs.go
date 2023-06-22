package sgen

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/scnewma/sgen/internal/fsutil"
)

func CacheDir() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "sgen"), nil
}

type SourceCache struct {
	Dir string
}

func NewSourceCache() (*SourceCache, error) {
	cacheDir, err := CacheDir()
	if err != nil {
		return nil, err
	}

	return &SourceCache{
		Dir: filepath.Join(cacheDir, "sources", "by-name"),
	}, nil
}

func (c *SourceCache) Store(name string, data []map[string]string) error {
	path := filepath.Join(c.Dir, name+".json")
	err := fsutil.WriteJSON(path, data)
	if err != nil {
		return fmt.Errorf("updating cache for %q: %w", name, err)
	}
	return nil
}

func (c *SourceCache) Load(name string) ([]map[string]string, error) {
	path := filepath.Join(c.Dir, name+".json")

	var data []map[string]string
	err := fsutil.ReadJSON(path, &data)
	if err != nil {
		return nil, fmt.Errorf("reading cache for %q: %w", name, err)
	}
	return data, nil
}
