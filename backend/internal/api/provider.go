package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/terragraph/backend/internal/parser"
	"github.com/terragraph/backend/internal/terraform"
)

func (s *Server) handleAddProvider(w http.ResponseWriter, r *http.Request) {
	var req parser.AddProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.WorkspacePath == "" || req.Provider == "" {
		writeError(w, http.StatusBadRequest, "workspacePath and provider are required")
		return
	}

	absPath, err := filepath.Abs(req.WorkspacePath)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid path: %s", err))
		return
	}

	info, err := os.Stat(absPath)
	if err != nil || !info.IsDir() {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("workspace path is not a directory: %s", absPath))
		return
	}

	req.WorkspacePath = absPath
	if req.File == "" {
		req.File = "main.tf"
	}
	if req.Source == "" {
		req.Source = providerRegistry(req.Provider)
	}
	if req.Version == "" {
		req.Version = "~> " + defaultVersion(req.Provider)
	}

	s.history.SaveBefore(absPath, req.File, "add-provider", fmt.Sprintf("add provider %s", req.Provider))

	result, err := parser.AddProvider(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("add provider failed: %s", err))
		return
	}

	// Auto-init after adding provider to download it
	initErr := terraform.Init(absPath)

	// Invalidate schema cache since we have a new provider
	delete(s.schemaCache, absPath)

	response := map[string]interface{}{
		"file":    result.File,
		"content": result.Content,
	}
	if initErr != nil {
		response["initError"] = initErr.Error()
	}

	writeJSON(w, response)
}
