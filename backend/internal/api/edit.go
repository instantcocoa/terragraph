package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/terragraph/backend/internal/parser"
)

func (s *Server) handleRenameBlock(w http.ResponseWriter, r *http.Request) {
	var req parser.RenameBlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.WorkspacePath == "" || req.File == "" || req.Address == "" || req.NewName == "" {
		writeError(w, http.StatusBadRequest, "workspacePath, file, address, and newName are required")
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

	s.history.SaveBefore(absPath, req.File, "rename-block", fmt.Sprintf("rename %s to %s", req.Address, req.NewName))

	result, err := parser.RenameBlock(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("rename block failed: %s", err))
		return
	}

	writeJSON(w, result)
}

func (s *Server) handleAddNestedBlock(w http.ResponseWriter, r *http.Request) {
	var req parser.AddNestedBlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.WorkspacePath == "" || req.File == "" || req.Address == "" || req.BlockType == "" {
		writeError(w, http.StatusBadRequest, "workspacePath, file, address, and blockType are required")
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

	s.history.SaveBefore(absPath, req.File, "add-nested-block", fmt.Sprintf("add %s to %s", req.BlockType, req.Address))

	result, err := parser.AddNestedBlock(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("add nested block failed: %s", err))
		return
	}

	writeJSON(w, result)
}

func (s *Server) handleRemoveNestedBlock(w http.ResponseWriter, r *http.Request) {
	var req parser.RemoveNestedBlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.WorkspacePath == "" || req.File == "" || req.Address == "" || req.BlockType == "" {
		writeError(w, http.StatusBadRequest, "workspacePath, file, address, and blockType are required")
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

	s.history.SaveBefore(absPath, req.File, "remove-nested-block", fmt.Sprintf("remove %s[%d] from %s", req.BlockType, req.Index, req.Address))

	result, err := parser.RemoveNestedBlock(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("remove nested block failed: %s", err))
		return
	}

	writeJSON(w, result)
}

func (s *Server) handleRemoveAttribute(w http.ResponseWriter, r *http.Request) {
	var req parser.RemoveAttributeRequest
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

	s.history.SaveBefore(absPath, req.File, "remove-attribute", fmt.Sprintf("remove %s.%s", req.Address, req.Attribute))

	result, err := parser.RemoveAttribute(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("remove attribute failed: %s", err))
		return
	}

	writeJSON(w, result)
}
