package web

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"mime"
	"net"
	"net/http"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/kpkym/kup/internal/config"
	"github.com/kpkym/kup/internal/runner"
)

//go:embed static/*
var staticFiles embed.FS

// Server serves the web UI and API for browsing restic snapshots.
type Server struct {
	global config.GlobalConfig
	repo   string
	mux    *http.ServeMux
}

// NewServer creates a new web server for the given repo.
func NewServer(global config.GlobalConfig, repo string) *Server {
	s := &Server{global: global, repo: repo, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) routes() {
	// API routes
	s.mux.HandleFunc("GET /api/snapshots", s.handleSnapshots)
	s.mux.HandleFunc("GET /api/snapshots/{id}/ls", s.handleLs)
	s.mux.HandleFunc("GET /api/snapshots/{id}/dump", s.handleDump)

	// Static files
	sub, _ := fs.Sub(staticFiles, "static")
	fileServer := http.FileServer(http.FS(sub))
	s.mux.Handle("GET /", fileServer)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// ListenAndServe starts the HTTP server with graceful shutdown on context cancellation.
func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:    addr,
		Handler: s,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	log.Printf("Listening on %s", addr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) resticCmd(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "restic", args...)
	cmd.Env = runner.ResticEnv(s.global, s.repo)
	return cmd
}

func (s *Server) handleSnapshots(w http.ResponseWriter, r *http.Request) {
	cmd := s.resticCmd(r.Context(), "snapshots", "--json")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("restic snapshots: %v: %s", err, stderr.String())
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("restic snapshots: %v: %s", err, stderr.String()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(stdout.Bytes())
}

func (s *Server) handleLs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	dirPath := r.URL.Query().Get("path")
	if dirPath == "" {
		dirPath = "/"
	}
	// Ensure trailing slash for directory matching
	if !strings.HasSuffix(dirPath, "/") {
		dirPath += "/"
	}

	cmd := s.resticCmd(r.Context(), "ls", "--json", id, dirPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("restic ls: %v: %s", err, stderr.String())
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("restic ls: %v: %s", err, stderr.String()))
		return
	}

	// Parse NDJSON, filter to direct children of dirPath
	type lsEntry struct {
		Name       string `json:"name"`
		Type       string `json:"type"`
		Path       string `json:"path"`
		Size       uint64 `json:"size,omitempty"`
		Mtime      string `json:"mtime,omitempty"`
		StructType string `json:"struct_type"`
	}

	var entries []lsEntry
	for line := range strings.SplitSeq(strings.TrimSpace(stdout.String()), "\n") {
		if line == "" {
			continue
		}
		var entry lsEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		// Skip the snapshot metadata line and the directory itself
		if entry.StructType == "snapshot" || entry.Path == strings.TrimSuffix(dirPath, "/") {
			continue
		}
		// Only include direct children
		rel := strings.TrimPrefix(entry.Path, strings.TrimSuffix(dirPath, "/"))
		rel = strings.TrimPrefix(rel, "/")
		if rel == "" || strings.Contains(rel, "/") {
			continue
		}
		entries = append(entries, entry)
	}

	if entries == nil {
		entries = []lsEntry{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

func (s *Server) handleDump(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		writeError(w, http.StatusBadRequest, "path parameter is required")
		return
	}

	cmd := s.resticCmd(r.Context(), "dump", id, filePath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("pipe: %v", err))
		return
	}

	if err := cmd.Start(); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("restic dump: %v", err))
		return
	}

	// Set content type from extension
	ext := path.Ext(filePath)
	ct := mime.TypeByExtension(ext)
	if ct == "" {
		ct = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Content-Disposition", "inline")

	// Stream stdout to response
	buf := make([]byte, 32*1024)
	for {
		n, readErr := stdout.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
		if readErr != nil {
			break
		}
	}

	if err := cmd.Wait(); err != nil {
		log.Printf("restic dump error for %s: %v: %s", filePath, err, stderr.String())
	}
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
