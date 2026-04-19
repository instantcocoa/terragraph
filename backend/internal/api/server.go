package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/terragraph/backend/internal/graph"
	"github.com/terragraph/backend/internal/history"
	"github.com/terragraph/backend/internal/parser"
	"github.com/terragraph/backend/internal/terraform"
)

// Server is the HTTP API server
type Server struct {
	mux         *http.ServeMux
	schemaCache map[string]*terraform.ProviderSchemas
	history     *history.Store
}

// NewServer creates and configures the API server
func NewServer() *Server {
	s := &Server{
		mux:         http.NewServeMux(),
		schemaCache: make(map[string]*terraform.ProviderSchemas),
		history:     history.NewStore(200),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("POST /api/workspace/load", s.handleLoadWorkspace)
	s.mux.HandleFunc("POST /api/workspace/validate", s.handleValidate)
	s.mux.HandleFunc("POST /api/workspace/plan", s.handlePlan)
	s.mux.HandleFunc("GET /api/workspace/file", s.handleGetFile)
	s.mux.HandleFunc("POST /api/workspace/patch", s.handlePatch)
	s.mux.HandleFunc("POST /api/workspace/schema", s.handleSchema)
	s.mux.HandleFunc("POST /api/workspace/add-block", s.handleAddBlock)
	s.mux.HandleFunc("POST /api/workspace/remove-block", s.handleRemoveBlock)
	s.mux.HandleFunc("POST /api/workspace/undo", s.handleUndo)
	s.mux.HandleFunc("POST /api/workspace/redo", s.handleRedo)
	s.mux.HandleFunc("POST /api/workspace/history", s.handleHistory)
	s.mux.HandleFunc("POST /api/workspace/init-project", s.handleInitProject)
	s.mux.HandleFunc("POST /api/workspace/eval-data", s.handleEvalData)
	s.mux.HandleFunc("POST /api/workspace/scaffold", s.handleScaffold)
	s.mux.HandleFunc("POST /api/workspace/rename-block", s.handleRenameBlock)
	s.mux.HandleFunc("POST /api/workspace/add-nested-block", s.handleAddNestedBlock)
	s.mux.HandleFunc("POST /api/workspace/remove-nested-block", s.handleRemoveNestedBlock)
	s.mux.HandleFunc("POST /api/workspace/remove-attribute", s.handleRemoveAttribute)
	s.mux.HandleFunc("POST /api/workspace/add-provider", s.handleAddProvider)
	s.mux.HandleFunc("POST /api/workspace/write-file", s.handleWriteFile)
	s.mux.HandleFunc("POST /api/pick-folder", s.handlePickFolder)
}

// Handler returns the HTTP handler with CORS middleware
func (s *Server) Handler() http.Handler {
	return corsMiddleware(s.mux)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type loadRequest struct {
	Path string `json:"path"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleLoadWorkspace(w http.ResponseWriter, r *http.Request) {
	var req loadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	absPath, err := filepath.Abs(req.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid path: %s", err))
		return
	}

	info, err := os.Stat(absPath)
	if err != nil || !info.IsDir() {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("path is not a directory: %s", absPath))
		return
	}

	files, diags, err := parser.ParseWorkspace(absPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("parse error: %s", err))
		return
	}

	extractor := graph.NewExtractor()
	var fileNames []string
	for _, f := range files {
		extractor.ExtractFile(f.Path, f.ReadBody, f.Source)
		fileNames = append(fileNames, f.Path)
	}

	result := extractor.Build()
	result.Diagnostics = append(result.Diagnostics, diags...)
	result.Files = fileNames

	writeJSON(w, result)
}

func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	var req loadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	absPath, err := filepath.Abs(req.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid path: %s", err))
		return
	}

	result, err := terraform.Validate(absPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("validation failed: %s", err))
		return
	}

	writeJSON(w, result)
}

func (s *Server) handlePlan(w http.ResponseWriter, r *http.Request) {
	var req loadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	absPath, err := filepath.Abs(req.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid path: %s", err))
		return
	}

	result, err := terraform.Plan(absPath)
	if err != nil {
		// Return the partial result with error info so the UI can display it
		if result != nil && result.RawOutput != "" {
			// Plan ran but failed - return the output so user sees the error
			type planErrorResponse struct {
				Error     string             `json:"planError"`
				Changes   []graph.PlanChange `json:"changes"`
				Summary   terraform.PlanSummary `json:"summary"`
				RawOutput string             `json:"rawOutput,omitempty"`
			}
			writeJSON(w, planErrorResponse{
				Error:     err.Error(),
				Changes:   result.Changes,
				Summary:   result.Summary,
				RawOutput: result.RawOutput,
			})
			return
		}
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("plan failed: %s", err))
		return
	}

	writeJSON(w, result)
}

func (s *Server) handleGetFile(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		writeError(w, http.StatusBadRequest, "path query parameter required")
		return
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid path: %s", err))
		return
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("file not found: %s", err))
		return
	}

	writeJSON(w, map[string]string{
		"path":    absPath,
		"content": string(content),
	})
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON: %v", err)
	}
}

func writeError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (s *Server) handleSchema(w http.ResponseWriter, r *http.Request) {
	var req loadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	absPath, err := filepath.Abs(req.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid path: %s", err))
		return
	}

	// Check cache
	if cached, ok := s.schemaCache[absPath]; ok {
		writeJSON(w, cached)
		return
	}

	schemas, err := terraform.GetProviderSchemas(absPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("schema retrieval failed: %s", err))
		return
	}

	// Cache the result
	s.schemaCache[absPath] = schemas

	writeJSON(w, schemas)
}
