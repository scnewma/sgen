package cmd

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
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
	path, err := ConfigFile()
	if err != nil {
		return err
	}
	config, diags := hclconfig.Parse(path)
	if err := writeDiags(diags, config.Files); err != nil {
		return err
	}

	// flags
	var (
		sync          bool
		template      string
		namedTemplate string
	)

	root := &cobra.Command{
		Use:           "sgen [SOURCE ...]",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !sync && len(args) == 0 {
				return cmd.Usage()
			}

			template = strings.TrimSpace(template)
			namedTemplate = strings.TrimSpace(namedTemplate)
			if template != "" && namedTemplate != "" {
				return fmt.Errorf("--template and --template-name are mutually exclusive")
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

			opts := []GenerateOption{}
			if template != "" {
				renderer, err := sgen.NewGoTemplateRenderer(template)
				if err != nil {
					return err
				}
				opts = append(opts, WithRenderer(renderer))
			} else if namedTemplate != "" {
				opts = append(opts, WithNamedRenderer(namedTemplate))
			}

			bw := bufio.NewWriter(os.Stdout)
			defer bw.Flush()
			return app.Generate(bw, opts...)
		},
	}

	root.Flags().BoolVarP(&sync, "sync", "S", false, "update sources")
	root.Flags().StringVarP(&template, "template", "t", "", "go template for rendering each source item, see: http://golang.org/pkg/text/template/#pkg-overview")
	root.Flags().StringVarP(&namedTemplate, "template-name", "n", "", "name of the template defined in config.hcl to use for rendering each source item")

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

		rndrs := map[string]sgen.Renderer{}
		for name, tpl := range cs.GetTemplates() {
			rndrs[name], err = sgen.NewGoTemplateRenderer(tpl)
			if err != nil {
				return nil, err
			}
		}
		if _, found := rndrs["default"]; !found {
			rndrs["default"] = &sgen.JSONRenderer{}
		}

		supplier, err := cs.ToSupplier()
		if err != nil {
			return nil, err
		}

		srcs = append(srcs, sgen.Source{
			Name:      cs.GetName(),
			Renderers: rndrs,
			Supplier:  supplier,
		})
	}
	return &SGen{
		Sources:  srcs,
		TplCache: tplcache.New(),
	}, nil
}

type generateOptions struct {
	renderer      sgen.Renderer
	namedRenderer string
}

func (o generateOptions) Renderer(src sgen.Source) sgen.Renderer {
	if o.renderer != nil {
		return o.renderer
	}
	if o.namedRenderer != "" {
		if rndr, found := src.Renderers[o.namedRenderer]; found {
			return rndr
		}
		// fallthrough
	}
	return src.Renderers["default"]
}

type GenerateOption func(*generateOptions)

func WithRenderer(r sgen.Renderer) GenerateOption {
	return func(opts *generateOptions) {
		opts.renderer = r
	}
}

func WithNamedRenderer(name string) GenerateOption {
	return func(opts *generateOptions) {
		opts.namedRenderer = name
	}
}

func (s *SGen) Generate(out io.Writer, opts ...GenerateOption) error {
	var options generateOptions
	for _, opt := range opts {
		opt(&options)
	}

	ctx := context.Background()
	for _, src := range s.Sources {
		rndr := options.Renderer(src)

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
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("generation requested for source without cached data, re-run with --sync to load data")
		} else if err != nil {
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
