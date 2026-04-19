package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/terragraph/backend/internal/parser"
)

func (s *Server) handlePatch(w http.ResponseWriter, r *http.Request) {
	var req parser.PatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.WorkspacePath == "" || req.File == "" || req.Address == "" || req.Attribute == "" {
		writeError(w, http.StatusBadRequest, "workspacePath, file, address, and attribute are required")
		return
	}

	absPath, err := filepath.Abs(req.WorkspacePath)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid workspace path: %s", err))
		return
	}

	info, err := os.Stat(absPath)
	if err != nil || !info.IsDir() {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("workspace path is not a directory: %s", absPath))
		return
	}

	req.WorkspacePath = absPath

	// Save history before mutation
	s.history.SaveBefore(absPath, req.File, "patch", fmt.Sprintf("set %s.%s = %s", req.Address, req.Attribute, req.Value))

	result, err := parser.PatchAttribute(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("patch failed: %s", err))
		return
	}

	writeJSON(w, result)
}
