package util

import (
	"bytes"
	"html/template"
)

type TemplateRenderer struct {
	tmpl *template.Template
}

func NewTemplateRenderer(tmpl *template.Template) *TemplateRenderer {
	return &TemplateRenderer{tmpl: tmpl}
}

func (tr *TemplateRenderer) Render(name string, data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := tr.tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (tr *TemplateRenderer) RenderHTML(name string, data interface{}) (template.HTML, error) {
	content, err := tr.Render(name, data)
	return template.HTML(content), err
}
