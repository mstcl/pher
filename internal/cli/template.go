package cli

import (
	"embed"
	"html/template"
	"path"
	"path/filepath"

	"github.com/mstcl/pher/v3/internal/state"
)

var EmbedFS embed.FS

func initTemplates(s *state.State) {
	funcMap := getTemplateFuncMap()
	tmpl := template.New("main")
	tmpl = tmpl.Funcs(funcMap)
	s.Templates = template.Must(tmpl.ParseFS(EmbedFS, filepath.Join(relTemplateDir, "*")))
}

func getTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"joinPath": path.Join,
	}
}
