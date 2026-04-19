package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

func (s *Server) handleUndo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
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

	file, err := s.history.Undo(absPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	status := s.history.GetStatus()
	writeJSON(w, map[string]interface{}{
		"file":      file,
		"undoCount": status.UndoCount,
		"redoCount": status.RedoCount,
	})
}

func (s *Server) handleRedo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
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

	file, err := s.history.Redo(absPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	status := s.history.GetStatus()
	writeJSON(w, map[string]interface{}{
		"file":      file,
		"undoCount": status.UndoCount,
		"redoCount": status.RedoCount,
	})
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	status := s.history.GetStatus()
	writeJSON(w, status)
}

func (s *Server) handleInitProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path     string `json:"path"`
		Provider string `json:"provider"` // e.g. "aws", "hcloud", "google"
		Region   string `json:"region"`   // e.g. "us-east-1"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Path == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	absPath, err := filepath.Abs(req.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid path: %s", err))
		return
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(absPath, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("creating directory: %s", err))
		return
	}

	// Generate provider config based on provider type
	providerSource := providerRegistry(req.Provider)
	region := req.Region
	if region == "" {
		region = defaultRegion(req.Provider)
	}

	mainTF := fmt.Sprintf(`terraform {
  required_version = ">= 1.0"

  required_providers {
    %s = {
      source  = "%s"
      version = "~> %s"
    }
  }
}

provider "%s" {
  region = "%s"
}
`, req.Provider, providerSource, defaultVersion(req.Provider), req.Provider, region)

	mainPath := filepath.Join(absPath, "main.tf")
	if err := os.WriteFile(mainPath, []byte(mainTF), 0644); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("writing main.tf: %s", err))
		return
	}

	writeJSON(w, map[string]string{
		"path": absPath,
		"file": "main.tf",
	})
}

func providerRegistry(provider string) string {
	switch provider {
	case "aws":
		return "hashicorp/aws"
	case "google":
		return "hashicorp/google"
	case "azurerm":
		return "hashicorp/azurerm"
	case "hcloud":
		return "hetznercloud/hcloud"
	case "digitalocean":
		return "digitalocean/digitalocean"
	case "cloudflare":
		return "cloudflare/cloudflare"
	default:
		return "hashicorp/" + provider
	}
}

func defaultVersion(provider string) string {
	switch provider {
	case "aws":
		return "5.0"
	case "google":
		return "5.0"
	case "azurerm":
		return "3.0"
	case "hcloud":
		return "1.0"
	default:
		return "1.0"
	}
}

func defaultRegion(provider string) string {
	switch provider {
	case "aws":
		return "us-east-1"
	case "google":
		return "us-central1"
	case "azurerm":
		return "eastus"
	case "hcloud":
		return "nbg1"
	default:
		return "us-east-1"
	}
}
