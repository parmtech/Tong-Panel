package main

import (
	"embed"
	"github.com/parmtech/tong-panel/internal/api"
	"io/fs"
	"net/http"
)

//go:embed ui/*
var assets embed.FS

func publicHandler(core *api.Server) http.Handler {
	mux := http.NewServeMux()
	handler := core.Handler()
	mux.Handle("/healthz", handler)
	mux.Handle("/api/", handler)
	root, _ := fs.Sub(assets, "ui")
	static := http.FileServer(http.FS(root))
	uiStatic := http.StripPrefix("/ui/", static)
	mux.HandleFunc("/ui/panel-status.mjs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		uiStatic.ServeHTTP(w, r)
	})
	mux.Handle("/ui/", uiStatic)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", 405)
			return
		}
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; object-src 'none'; frame-ancestors 'none'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		http.ServeFileFS(w, r, assets, "ui/index.html")
	})
	return mux
}
