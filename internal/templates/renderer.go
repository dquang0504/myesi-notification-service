package templates

import (
	"bytes"
	"text/template"
)

// Renderer renders notification templates using text/template semantics.
type Renderer struct{}

// Render applies the provided data to a template string.
func (Renderer) Render(tpl string, data map[string]interface{}) (string, error) {
	parsed, err := template.New("tpl").Parse(tpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := parsed.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
