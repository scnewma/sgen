package hclconfig

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParse(t *testing.T) {
	expect := &Config{
		Sources: map[string]Source{
			"gh": &CommandSourceBlock{
				SourceBlock: SourceBlock{
					Name:      "gh",
					Templates: map[string]string{},
				},
				Command: "gh repo list --json nameWithOwner",
			},
			"gh_w_template": &CommandSourceBlock{
				SourceBlock: SourceBlock{
					Name: "gh_w_template",
					Templates: map[string]string{
						"default": "{{.nameWithOwner}}",
						"name":    "{{.name}}",
					},
				},
				Command: "gh repo list --json nameWithOwner",
			},
			"static": &FileSourceBlock{
				SourceBlock: SourceBlock{
					Name:      "static",
					Templates: map[string]string{},
				},
				Path: "/data.json",
			},
			"static_w_template": &FileSourceBlock{
				SourceBlock: SourceBlock{
					Name: "static_w_template",
					Templates: map[string]string{
						"default": "{{.nameWithOwner}}",
						"name":    "{{.name}}",
					},
				},
				Path: "/data.json",
			},
		},
	}

	parsed, diags := Parse("testdata/basic.hcl")
	if diags.HasErrors() {
		t.Fatalf("Unexpected diagnostics: %s", diags)
	}

	if diff := cmp.Diff(expect.Sources, parsed.Sources); diff != "" {
		t.Errorf("Parse() data mismatch (-want +got):\n%s", diff)
	}
}
