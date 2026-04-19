package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/terragraph/backend/internal/terraform"
)

var (
	lspClients   = make(map[string]*terraform.LSPClient)
	lspClientsMu sync.Mutex
)

func getLSPClient(workspacePath string) (*terraform.LSPClient, error) {
	lspClientsMu.Lock()
	defer lspClientsMu.Unlock()

	if client, ok := lspClients[workspacePath]; ok {
		return client, nil
	}

	client, err := terraform.StartLSP(workspacePath)
	if err != nil {
		return nil, err
	}

	lspClients[workspacePath] = client
	return client, nil
}

func (s *Server) handleLSPStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"available": terraform.IsLSPAvailable(),
	})
}

func (s *Server) handleLSPDiagnostics(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"` // workspace path
		File string `json:"file"` // relative file name
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	absPath, err := filepath.Abs(req.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid path: %s", err))
		return
	}

	filePath := filepath.Join(absPath, req.File)
	content, err := os.ReadFile(filePath)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("file not found: %s", err))
		return
	}

	client, err := getLSPClient(absPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("LSP unavailable: %s", err))
		return
	}

	diags, err := client.GetDiagnostics(filePath, string(content))
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("LSP diagnostics failed: %s", err))
		return
	}

	writeJSON(w, map[string]interface{}{
		"diagnostics": diags,
	})
}

func (s *Server) handleLSPHover(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"` // workspace path
		File string `json:"file"` // relative file name
		Line int    `json:"line"`
		Col  int    `json:"col"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	absPath, err := filepath.Abs(req.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid path: %s", err))
		return
	}

	filePath := filepath.Join(absPath, req.File)

	client, err := getLSPClient(absPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("LSP unavailable: %s", err))
		return
	}

	result, err := client.Hover(filePath, req.Line, req.Col)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("LSP hover failed: %s", err))
		return
	}

	if result == nil {
		writeJSON(w, map[string]interface{}{"contents": nil})
		return
	}

	writeJSON(w, result)
}
