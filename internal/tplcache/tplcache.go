package tplcache

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"

	"github.com/scnewma/sgen/internal/sgen"
)

type Cache struct {
	BaseDir string
}

func New() *Cache {
	cacheDir, err := sgen.CacheDir()
	if err != nil {
		panic(err)
	}
	return &Cache{
		BaseDir: cacheDir,
	}
}

func (c *Cache) Clear(src string) error {
	d := c.srcDir(src)
	return os.RemoveAll(d)
}

func (c *Cache) Set(src, tpl string, data []byte) error {
	d := filepath.Join(c.srcDir(src), c.hash(tpl))
	p := filepath.Join(d, "out")
	if err := os.MkdirAll(d, 0755); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0755)
}

func (c *Cache) Open(src, tpl string) (io.WriteCloser, error) {
	d := filepath.Join(c.srcDir(src), c.hash(tpl))
	p := filepath.Join(d, "out")
	if err := os.MkdirAll(d, 0755); err != nil {
		return nil, err
	}
	return os.Create(p)
}

func (c *Cache) Get(src, tpl string) ([]byte, error) {
	d := filepath.Join(c.srcDir(src), c.hash(tpl))
	p := filepath.Join(d, "out")
	return os.ReadFile(p)
}

func (c *Cache) srcDir(src string) string {
	return filepath.Join(c.BaseDir, "templates", "by-source", src)
}

func (c *Cache) hash(s string) string {
	w := sha256.New()
	w.Write([]byte(s))
	return hex.EncodeToString(w.Sum(nil))
}
