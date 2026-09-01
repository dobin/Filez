package server

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// entry is a single row in the listing.
type entry struct {
	Name    string
	Size    int64
	ModTime time.Time
	IsDir   bool
}

// indexData is passed to index.html.
type indexData struct {
	Path     string // relative path with leading slash, "" for root
	Entries  []entry
	Parent   string // relative path of parent, "" if at root
	HasError bool
	Error    string
}

// resolvePath cleans rel against root and ensures the result stays within root.
func (s *Server) resolvePath(rel string) (string, error) {
	cleaned := filepath.Clean("/" + strings.TrimPrefix(rel, "/"))
	abs := filepath.Join(s.root, cleaned)
	// filepath.Join collapses leading slash; verify containment.
	rel2, err := filepath.Rel(s.root, abs)
	if err != nil || strings.HasPrefix(rel2, "..") {
		return "", errors.New("path escapes root")
	}
	return abs, nil
}

// relFromRoot returns the cleaned relative path (uses "/" separators) for a given abs.
func (s *Server) relFromRoot(abs string) string {
	rel, err := filepath.Rel(s.root, abs)
	if err != nil {
		return ""
	}
	rel = filepath.ToSlash(rel)
	if rel == "." {
		return ""
	}
	return "/" + rel
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	rel := r.URL.Query().Get("path")
	abs, err := s.resolvePath(rel)
	if err != nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	info, err := os.Stat(abs)
	if err != nil {
		http.Error(w, "cannot read path", http.StatusBadRequest)
		return
	}
	if !info.IsDir() {
		http.Error(w, "not a directory", http.StatusBadRequest)
		return
	}

	dirEntries, err := os.ReadDir(abs)
	if err != nil {
		http.Error(w, "cannot read directory", http.StatusInternalServerError)
		return
	}

	data := indexData{Path: s.relFromRoot(abs)}
	if data.Path != "" {
		parent := filepath.ToSlash(filepath.Dir(strings.TrimPrefix(data.Path, "/")))
		if parent == "." || parent == "" {
			data.Parent = ""
		} else {
			data.Parent = "/" + parent
		}
	}

	for _, de := range dirEntries {
		fi, err := de.Info()
		if err != nil {
			continue
		}
		data.Entries = append(data.Entries, entry{
			Name:    de.Name(),
			Size:    fi.Size(),
			ModTime: fi.ModTime(),
			IsDir:   de.IsDir(),
		})
	}
	sort.Slice(data.Entries, func(i, j int) bool {
		if data.Entries[i].IsDir != data.Entries[j].IsDir {
			return data.Entries[i].IsDir
		}
		return data.Entries[i].Name < data.Entries[j].Name
	})

	s.renderIndex(w, data)
}

func (s *Server) renderIndex(w http.ResponseWriter, data indexData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = s.render.index.Execute(w, data)
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	rel := r.URL.Query().Get("path")
	abs, err := s.resolvePath(rel)
	if err != nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	if info, err := os.Stat(abs); err != nil || !info.IsDir() {
		http.Error(w, "not a directory", http.StatusBadRequest)
		return
	}
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "bad multipart: "+err.Error(), http.StatusBadRequest)
		return
	}
	files := r.MultipartForm.File["files[]"]
	if len(files) == 0 {
		http.Error(w, "no files", http.StatusBadRequest)
		return
	}
	for _, fh := range files {
		src, err := fh.Open()
		if err != nil {
			http.Error(w, "open upload: "+err.Error(), http.StatusBadRequest)
			return
		}
		dstPath := filepath.Join(abs, filepath.Base(fh.Filename))
		dst, err := os.Create(dstPath)
		if err != nil {
			src.Close()
			http.Error(w, "create file: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if _, err := io.Copy(dst, src); err != nil {
			src.Close()
			dst.Close()
			http.Error(w, "write file: "+err.Error(), http.StatusInternalServerError)
			return
		}
		src.Close()
		dst.Close()
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	rel := r.URL.Query().Get("path")
	abs, err := s.resolvePath(rel)
	if err != nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	info, err := os.Stat(abs)
	if err != nil || info.IsDir() {
		http.Error(w, "not a file", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, filepath.Base(abs)))
	http.ServeFile(w, r, abs)
}

// handleBlob returns the raw bytes; JS wraps them in a Blob and triggers a download.
func (s *Server) handleBlob(w http.ResponseWriter, r *http.Request) {
	rel := r.URL.Query().Get("path")
	abs, err := s.resolvePath(rel)
	if err != nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	info, err := os.Stat(abs)
	if err != nil || info.IsDir() {
		http.Error(w, "not a file", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, filepath.Base(abs)))
	http.ServeFile(w, r, abs)
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	rel := r.URL.Query().Get("path")
	abs, err := s.resolvePath(rel)
	if err != nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	if abs == s.root {
		http.Error(w, "cannot delete root", http.StatusBadRequest)
		return
	}
	if err := os.RemoveAll(abs); err != nil {
		http.Error(w, "delete: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// isEditable reports whether a file may be opened in the text editor.
func isEditable(name string) bool {
	n := strings.ToLower(name)
	return strings.HasSuffix(n, ".txt") || strings.HasSuffix(n, ".md")
}

// handleRead returns the raw text of an editable file.
func (s *Server) handleRead(w http.ResponseWriter, r *http.Request) {
	rel := r.URL.Query().Get("path")
	abs, err := s.resolvePath(rel)
	if err != nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	info, err := os.Stat(abs)
	if err != nil || info.IsDir() {
		http.Error(w, "not a file", http.StatusBadRequest)
		return
	}
	if !isEditable(filepath.Base(abs)) {
		http.Error(w, "file type not editable", http.StatusBadRequest)
		return
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		http.Error(w, "read: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(data)
}

// handleSave overwrites an editable file with the request body, atomically.
func (s *Server) handleSave(w http.ResponseWriter, r *http.Request) {
	rel := r.URL.Query().Get("path")
	abs, err := s.resolvePath(rel)
	if err != nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	info, err := os.Stat(abs)
	if err != nil || info.IsDir() {
		http.Error(w, "not a file", http.StatusBadRequest)
		return
	}
	if !isEditable(filepath.Base(abs)) {
		http.Error(w, "file type not editable", http.StatusBadRequest)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
		return
	}
	dir := filepath.Dir(abs)
	tmp, err := os.CreateTemp(dir, ".filez-edit-*")
	if err != nil {
		http.Error(w, "create temp: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		http.Error(w, "write temp: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		http.Error(w, "close temp: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		os.Remove(tmpName)
		http.Error(w, "chmod temp: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := os.Rename(tmpName, abs); err != nil {
		os.Remove(tmpName)
		http.Error(w, "rename: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMkdir(w http.ResponseWriter, r *http.Request) {
	rel := r.URL.Query().Get("path")
	name := r.URL.Query().Get("name")
	if name == "" || strings.ContainsAny(name, `/\`) {
		http.Error(w, "invalid folder name", http.StatusBadRequest)
		return
	}
	abs, err := s.resolvePath(rel)
	if err != nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	if err := os.Mkdir(filepath.Join(abs, filepath.Base(name)), 0o755); err != nil {
		http.Error(w, "mkdir: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
