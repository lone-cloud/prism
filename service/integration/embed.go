package integration

import "embed"

//go:embed templates/*.html
var templates embed.FS

func GetTemplates() embed.FS {
	return templates
}
