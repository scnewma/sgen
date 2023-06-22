package supply

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFileSync(t *testing.T) {
	expect := []map[string]string{
		{"name": "bob"},
		{"name": "alice"},
	}

	paths := []string{
		"testdata/people.json",
		"testdata/people.yaml",
		"testdata/people.yml",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			s := File{path: path}
			data, err := s.Supply(context.Background())
			if err != nil {
				t.Fatalf("file sync error: %v", err)
			}
			t.Logf("%+v\n", data)

			if diff := cmp.Diff(expect, data); diff != "" {
				t.Errorf("File.Sync() data mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
