package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/terragraph/backend/internal/terraform"
)

func (s *Server) handleEvalData(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path    string `json:"path"`
		Address string `json:"address"` // e.g. "data.aws_ami.ubuntu"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Path == "" || req.Address == "" {
		writeError(w, http.StatusBadRequest, "path and address are required")
		return
	}

	absPath, err := filepath.Abs(req.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid path: %s", err))
		return
	}

	result, err := terraform.EvalData(absPath, req.Address)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("eval failed: %s", err))
		return
	}

	writeJSON(w, result)
}
