package templating

import (
	"fmt"
	"strings"

	"github.com/flosch/pongo2/v6"
)

const MaxTemplateLength = 256 * 1024
const MaxRenderedLength = 512 * 1024

type Renderer interface {
	Render(template string, data map[string]interface{}) (string, error)
}

type PongoRenderer struct{}

func NewPongoRenderer() *PongoRenderer { return &PongoRenderer{} }

func (r *PongoRenderer) Render(templateText string, data map[string]interface{}) (string, error) {
	if strings.TrimSpace(templateText) == "" {
		return "", nil
	}
	if len([]byte(templateText)) > MaxTemplateLength {
		return "", fmt.Errorf("template too large: %d bytes", len([]byte(templateText)))
	}
	tpl, err := pongo2.FromString(templateText)
	if err != nil {
		return "", err
	}
	ctx := pongo2.Context{}
	for k, v := range data {
		ctx[k] = v
	}
	out, err := tpl.Execute(ctx)
	if err != nil {
		return "", err
	}
	if len([]byte(out)) > MaxRenderedLength {
		return "", fmt.Errorf("rendered output too large: %d bytes", len([]byte(out)))
	}
	return out, nil
}

func SafeRender(templateText string, data map[string]interface{}, fallback string) string {
	out, err := NewPongoRenderer().Render(templateText, data)
	if err != nil || strings.TrimSpace(out) == "" {
		return fallback
	}
	return out
}
