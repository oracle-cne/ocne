package template

import (
	"embed"
	"fmt"
)

//go:embed all:templates
var templates embed.FS

func ReadTemplate(name string) ([]byte, error) {
	return templates.ReadFile(fmt.Sprintf("templates/%s", name))
}
