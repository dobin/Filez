package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"filez/web"
)

// New builds the configured chi router.
func New(root, token string) http.Handler {
	s := &Server{root: root, token: token, render: newRender()}
	r := chi.NewRouter()

	r.Get("/login", s.handleLoginPage)
	r.Post("/login", s.handleLogin)
	r.Post("/logout", s.handleLogout)

	r.Group(func(r chi.Router) {
		r.Use(s.tokenAuth)
		r.Get("/", s.handleIndex)
		r.Post("/api/upload", s.handleUpload)
		r.Get("/api/download", s.handleDownload)
		r.Get("/api/blob", s.handleBlob)
		r.Get("/api/read", s.handleRead)
		r.Post("/api/save", s.handleSave)
		r.Post("/api/delete", s.handleDelete)
		r.Post("/api/mkdir", s.handleMkdir)
		r.Post("/api/newfile", s.handleNewFile)
	})

	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(web.StaticSubFS()))))
	return r
}

// Server holds shared dependencies for handlers.
type Server struct {
	root   string
	token  string
	render *render
}
