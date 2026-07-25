package server

import (
	"fmt"
	"html/template"

	"filez/web"
)

type render struct {
	index *template.Template
	login *template.Template
}

func newRender() *render {
	funcs := template.FuncMap{"humanSize": humanSize}
	idx := template.Must(template.New("index.html").Funcs(funcs).ParseFS(web.TemplatesFS, "templates/index.html"))
	lgn := template.Must(template.ParseFS(web.TemplatesFS, "templates/login.html"))
	return &render{index: idx, login: lgn}
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

