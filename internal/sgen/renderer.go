package sgen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

type Renderer interface {
	ID() string
	Render(map[string]string) (string, error)
}

type JSONRenderer struct{}

func (r *JSONRenderer) ID() string {
	return "<JSON>"
}

func (r *JSONRenderer) Render(data map[string]string) (string, error) {
	buf, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("rendering json: %w", err)
	}
	return string(buf), nil
}

type GoTemplateRenderer struct {
	tmplStr string
	tmpl    *template.Template
}

func NewGoTemplateRenderer(tmpl string) (*GoTemplateRenderer, error) {
	t, err := template.New("").Funcs(sprig.FuncMap()).Parse(tmpl)
	if err != nil {
		return nil, fmt.Errorf("invalid template %q: %w", tmpl, err)
	}
	return &GoTemplateRenderer{
		tmplStr: tmpl,
		tmpl:    t,
	}, nil
}

func (r *GoTemplateRenderer) ID() string {
	return r.tmplStr
}

func (r *GoTemplateRenderer) Render(data map[string]string) (string, error) {
	buf := new(bytes.Buffer)
	if err := r.tmpl.Execute(buf, data); err != nil {
		return "", fmt.Errorf("rendering go template %q: %w", r.tmplStr, err)
	}
	return buf.String(), nil
}
