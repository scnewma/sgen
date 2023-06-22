package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/scnewma/sgen/internal/encoding"
	"github.com/scnewma/sgen/internal/sgen"
	"github.com/scnewma/sgen/internal/sgen/supply"
	"github.com/spf13/cobra"
)

func Execute() int {
	if err := execute(); err != nil {
		fmt.Printf("error: %v\n", err)
		return 1
	}
	return 0
}

func execute() error {
	config, err := LoadConfig()
	if err != nil {
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
					sources = append(sources, cs.Name)
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

			return app.Generate(os.Stdout, renderer)
		},
	}

	root.Flags().BoolVarP(&sync, "sync", "S", false, "update sources")
	root.Flags().StringVarP(&template, "template", "t", "", "go template for rendering each source item, see: http://golang.org/pkg/text/template/#pkg-overview")

	return root.Execute()
}

type SGen struct {
	Sources []sgen.Source
}

type SGenOpts struct {
	Config  *Config
	Sources []string
}

func NewSGen(opts SGenOpts) (*SGen, error) {
	var srcs []sgen.Source
	for _, srcName := range opts.Sources {
		var err error

		cs := opts.Config.GetSource(srcName)
		if cs == nil {
			return nil, fmt.Errorf("source %q not configured", srcName)
		}

		var rndr sgen.Renderer
		if cs.DefaultTemplate != "" {
			rndr, err = sgen.NewGoTemplateRenderer(cs.DefaultTemplate)
			if err != nil {
				return nil, err
			}
		} else {
			rndr = &sgen.JSONRenderer{}
		}

		supplier, err := ToSupplier(cs)
		if err != nil {
			return nil, err
		}

		srcs = append(srcs, sgen.Source{
			Name:            cs.Name,
			DefaultRenderer: rndr,
			Supplier:        supplier,
		})
	}
	return &SGen{
		Sources: srcs,
	}, nil
}

func (s *SGen) Generate(w io.Writer, renderer sgen.Renderer) error {
	ctx := context.Background()
	for _, src := range s.Sources {
		data, err := src.Load(ctx)
		if err != nil {
			return fmt.Errorf("syncing %s: %w", src.Name, err)
		}

		rndr := renderer
		if rndr == nil {
			rndr = src.DefaultRenderer
		}

		for _, datum := range data {
			line, err := rndr.Render(datum)
			if err != nil {
				dataStr, err := encoding.EncodeJSONString(datum)
				if err != nil {
					dataStr = "<encoding JSON failure>"
				}

				return fmt.Errorf("render failure with data %q: %w", dataStr, err)
			}

			fmt.Fprintln(w, line)
		}
	}

	return nil
}

func (s *SGen) Sync() error {
	ctx := context.Background()
	for _, src := range s.Sources {
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
