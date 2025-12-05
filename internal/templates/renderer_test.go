package templates

import "testing"

func TestRendererRender(t *testing.T) {
	r := Renderer{}
	out, err := r.Render("Hello {{.name}}", map[string]interface{}{"name": "MyESI"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "Hello MyESI" {
		t.Fatalf("unexpected render output: %s", out)
	}
}
