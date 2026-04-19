package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/terragraph/backend/internal/parser"
)

func (s *Server) handleAddBlock(w http.ResponseWriter, r *http.Request) {
	var req parser.AddBlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.WorkspacePath == "" || req.File == "" || req.BlockType == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "workspacePath, file, blockType, and name are required")
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

	detail := fmt.Sprintf("add %s %s", req.BlockType, req.Name)
	if req.ResourceType != "" {
		detail = fmt.Sprintf("add %s %s.%s", req.BlockType, req.ResourceType, req.Name)
	}
	s.history.SaveBefore(absPath, req.File, "add-block", detail)

	result, err := parser.AddBlock(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("add block failed: %s", err))
		return
	}

	writeJSON(w, result)
}

func (s *Server) handleRemoveBlock(w http.ResponseWriter, r *http.Request) {
	var req parser.RemoveBlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.WorkspacePath == "" || req.File == "" || req.Address == "" {
		writeError(w, http.StatusBadRequest, "workspacePath, file, and address are required")
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

	s.history.SaveBefore(absPath, req.File, "remove-block", fmt.Sprintf("remove %s", req.Address))

	result, err := parser.RemoveBlock(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("remove block failed: %s", err))
		return
	}

	writeJSON(w, result)
}
