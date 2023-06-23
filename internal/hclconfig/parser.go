package hclconfig

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/scnewma/sgen/internal/sgen"
	"github.com/scnewma/sgen/internal/sgen/supply"
)

type Config struct {
	Sources map[string]Source
	Files   map[string]*hcl.File
}

type Source interface {
	GetName() string
	GetDefaultTemplate() string
	ToSupplier() (sgen.Supplier, error)
}

type SourceBlock struct {
	Name            string
	DefaultTemplate string
}

func (b *SourceBlock) GetName() string {
	return b.Name
}

func (b *SourceBlock) GetDefaultTemplate() string {
	return b.DefaultTemplate
}

type FileSourceBlock struct {
	SourceBlock
	Path string
}

func (b *FileSourceBlock) ToSupplier() (sgen.Supplier, error) {
	return supply.NewFileSupply(b.Path)
}

type CommandSourceBlock struct {
	SourceBlock
	Command string
}

func (b *CommandSourceBlock) ToSupplier() (sgen.Supplier, error) {
	return supply.NewCommandSupply(b.Command)
}

var configSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "source", LabelNames: []string{"type", "name"}},
	},
}

func Parse(filename string) (*Config, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	config := &Config{
		Sources: make(map[string]Source),
		Files:   make(map[string]*hcl.File),
	}

	parser := hclparse.NewParser()
	f, moreDiags := parser.ParseHCLFile(filename)
	diags = append(diags, moreDiags...)
	if moreDiags.HasErrors() {
		return config, diags
	}

	config.Files[filename] = f

	content, moreDiags := f.Body.Content(configSchema)
	diags = append(diags, moreDiags...)
	for _, block := range content.Blocks {
		switch block.Type {
		case "source":
			typ := block.Labels[0]
			name := block.Labels[1]
			switch typ {
			case "file":
				source, moreDiags := decodeFileSource(name, block)
				diags = append(diags, moreDiags...)
				if moreDiags.HasErrors() {
					continue
				}
				config.Sources[source.Name] = source
			case "command":
				source, moreDiags := decodeCommandSource(name, block)
				diags = append(diags, moreDiags...)
				if moreDiags.HasErrors() {
					continue
				}
				config.Sources[source.Name] = source
			default:
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("Source type %q unknown", typ),
				})
			}
		}
	}
	return config, diags
}

func decodeFileSource(name string, block *hcl.Block) (*FileSourceBlock, hcl.Diagnostics) {
	source := &FileSourceBlock{
		SourceBlock: SourceBlock{Name: name},
	}
	var b struct {
		DefaultTemplate string `hcl:"default_template,optional"`
		Path            string `hcl:"path"`
	}
	diags := gohcl.DecodeBody(block.Body, nil, &b)
	if diags.HasErrors() {
		return source, diags
	}
	source.DefaultTemplate = b.DefaultTemplate
	source.Path = b.Path
	return source, diags
}

func decodeCommandSource(name string, block *hcl.Block) (*CommandSourceBlock, hcl.Diagnostics) {
	source := &CommandSourceBlock{
		SourceBlock: SourceBlock{Name: name},
	}
	var b struct {
		DefaultTemplate string `hcl:"default_template,optional"`
		Command         string `hcl:"command"`
	}
	diags := gohcl.DecodeBody(block.Body, nil, &b)
	if diags.HasErrors() {
		return source, diags
	}
	source.Command = b.Command
	source.DefaultTemplate = b.DefaultTemplate
	return source, diags
}
