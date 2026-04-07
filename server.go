package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
)

//go:embed web
var webFS embed.FS

func newHandler(kubeconfigPath string, disc *Discovery) http.Handler {
	mux := http.NewServeMux()

	// Serve frontend assets from embedded filesystem
	webContent, _ := fs.Sub(webFS, "web")
	fileServer := http.FileServer(http.FS(webContent))
	mux.Handle("/", fileServer)

	// WebSocket terminal endpoint
	mux.Handle("/ws", handleTerminal(kubeconfigPath))

	// Cluster list API
	mux.HandleFunc("/api/clusters", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(disc.Clusters())
	})

	return mux
}
