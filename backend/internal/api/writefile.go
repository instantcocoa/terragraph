package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (s *Server) handleWriteFile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Path == "" || !strings.HasSuffix(req.Path, ".tf") {
		writeError(w, http.StatusBadRequest, "path must be a .tf file")
		return
	}

	absPath, err := filepath.Abs(req.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid path: %s", err))
		return
	}

	// Save history before writing
	dir := filepath.Dir(absPath)
	base := filepath.Base(absPath)
	s.history.SaveBefore(dir, base, "edit-hcl", fmt.Sprintf("edit %s", base))

	if err := os.WriteFile(absPath, []byte(req.Content), 0644); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("write failed: %s", err))
		return
	}

	writeJSON(w, map[string]string{
		"file": base,
		"path": absPath,
	})
}
