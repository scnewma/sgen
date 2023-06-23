package cmd

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/spf13/cobra"

	"github.com/scnewma/sgen/internal/encoding"
	"github.com/scnewma/sgen/internal/hclconfig"
	"github.com/scnewma/sgen/internal/sgen"
	"github.com/scnewma/sgen/internal/sgen/supply"
	"github.com/scnewma/sgen/internal/tplcache"
)

type ExitCodeError struct {
	ExitCode int
}

func (e ExitCodeError) Error() string {
	return fmt.Sprintf("exit code %d", e.ExitCode)
}

func Execute() int {
	if err := execute(); err != nil {
		var exitCodeErr ExitCodeError
		if errors.As(err, &exitCodeErr) {
			return exitCodeErr.ExitCode
		}

		fmt.Printf("error: %v\n", err)
		return 1
	}
	return 0
}

func execute() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not find HOME: %w", err)
	}
	path := filepath.Join(homeDir, ".config", "sgen", "config.hcl")
	config, diags := hclconfig.Parse(path)
	if err := writeDiags(diags, config.Files); err != nil {
		return err
	}

	// flags
	var (
		sync     bool
		template string
	)

	root := &cobra.Command{
		Use: "sgen [SOURCE ...]",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !sync && len(args) == 0 {
				return cmd.Usage()
			}

			// special case, if the user just specifies -S then we sync all of
			// the sources
			if sync && len(args) == 0 {
				var sources []string
				for _, cs := range config.Sources {
					sources = append(sources, cs.GetName())
				}

				app, err := NewSGen(SGenOpts{
					Config:  config,
					Sources: sources,
				})
				if err != nil {
					return err
				}
				return app.Sync()
			}

			app, err := NewSGen(SGenOpts{
				Config:  config,
				Sources: args,
			})
			if err != nil {
				return err
			}

			if sync {
				if err := app.Sync(); err != nil {
					return err
				}
			}

			var renderer sgen.Renderer
			template = strings.TrimSpace(template)
			if template != "" {
				renderer, err = sgen.NewGoTemplateRenderer(template)
				if err != nil {
					return err
				}
			}

			bw := bufio.NewWriter(os.Stdout)
			defer bw.Flush()
			return app.Generate(bw, renderer)
		},
	}

	root.Flags().BoolVarP(&sync, "sync", "S", false, "update sources")
	root.Flags().StringVarP(&template, "template", "t", "", "go template for rendering each source item, see: http://golang.org/pkg/text/template/#pkg-overview")

	return root.Execute()
}

func writeDiags(diags hcl.Diagnostics, files map[string]*hcl.File) error {
	var b bytes.Buffer
	w := hcl.NewDiagnosticTextWriter(&b, files, 80, true)
	if err := w.WriteDiagnostics(diags); err != nil {
		return fmt.Errorf("writing diagnostics: %w", err)
	}
	if b.Len() != 0 {
		fmt.Print(b.String())
		if diags.HasErrors() {
			return ExitCodeError{ExitCode: 1}
		}
	}
	return nil
}

type SGen struct {
	Sources  []sgen.Source
	TplCache *tplcache.Cache
}

type SGenOpts struct {
	Config  *hclconfig.Config
	Sources []string
}

func NewSGen(opts SGenOpts) (*SGen, error) {
	var srcs []sgen.Source
	for _, srcName := range opts.Sources {
		var err error

		cs, found := opts.Config.Sources[srcName]
		if !found {
			return nil, fmt.Errorf("source %q not configured", srcName)
		}

		var rndr sgen.Renderer
		if cs.GetDefaultTemplate() != "" {
			rndr, err = sgen.NewGoTemplateRenderer(cs.GetDefaultTemplate())
			if err != nil {
				return nil, err
			}
		} else {
			rndr = &sgen.JSONRenderer{}
		}

		supplier, err := cs.ToSupplier()
		if err != nil {
			return nil, err
		}

		srcs = append(srcs, sgen.Source{
			Name:            cs.GetName(),
			DefaultRenderer: rndr,
			Supplier:        supplier,
		})
	}
	return &SGen{
		Sources:  srcs,
		TplCache: tplcache.New(),
	}, nil
}

func (s *SGen) Generate(out io.Writer, renderer sgen.Renderer) error {
	ctx := context.Background()
	for _, src := range s.Sources {
		rndr := renderer
		if rndr == nil {
			rndr = src.DefaultRenderer
		}

		if cache, err := s.TplCache.Get(src.Name, rndr.ID()); err == nil && cache != nil {
			// if an error happens copying the cached date into the writer we
			// can't just fallback to loading the underlying source and using
			// that data since we may have partially written the cached data,
			// which would create corrupted output on the writer
			if _, err := io.Copy(out, bytes.NewBuffer(cache)); err != nil {
				return err
			}
			continue
		}

		data, err := src.Load(ctx)
		if err != nil {
			return fmt.Errorf("syncing %s: %w", src.Name, err)
		}

		cacheW, err := s.TplCache.Open(src.Name, rndr.ID())
		if err != nil {
			return err
		}
		defer cacheW.Close()
		w := io.MultiWriter(out, cacheW)

		for _, datum := range data {
			line, err := rndr.Render(datum)
			if err != nil {
				dataStr, err := encoding.EncodeJSONString(datum)
				if err != nil {
					dataStr = "<encoding JSON failure>"
				}

				return fmt.Errorf("render failure with data %q: %w", dataStr, err)
			}

			if _, err := fmt.Fprintln(w, line); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *SGen) Sync() error {
	ctx := context.Background()

	for _, src := range s.Sources {
		if err := s.TplCache.Clear(src.Name); err != nil {
			return err
		}

		if err := src.Sync(ctx); err != nil {
			return err
		}
	}
	return nil
}

func ToSupplier(cs *ConfigSource) (sgen.Supplier, error) {
	type convertFn func(*ConfigSource) (sgen.Supplier, error)
	supplierConverters := []struct {
		Type    string
		Convert convertFn
	}{
		{
			Type: "file",
			Convert: func(cs *ConfigSource) (sgen.Supplier, error) {
				if cs.File == nil {
					return nil, fmt.Errorf("%s: file sources must define a path", cs.Name)
				}
				return supply.NewFileSupply(cs.File.Path)
			},
		},
		{
			Type: "command",
			Convert: func(cs *ConfigSource) (sgen.Supplier, error) {
				if cs.Command == nil {
					return nil, fmt.Errorf("%s: command sources must define a command", cs.Name)
				}
				return supply.NewCommandSupply(*cs.Command)
			},
		},
	}

	var convert convertFn
	allTypes := []string{}
	for _, conv := range supplierConverters {
		allTypes = append(allTypes, conv.Type)

		if conv.Type == cs.Type {
			convert = conv.Convert
		}
	}

	if convert == nil {
		validTypes := fmt.Sprintf("[%s]", strings.Join(allTypes, ","))
		return nil, fmt.Errorf("%s: invalid source type %q, valid types are %s", cs.Name, cs.Type, validTypes)
	}

	return convert(cs)
}
