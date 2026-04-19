package api

import (
	"encoding/json"
	"net/http"

	"github.com/terragraph/backend/internal/terraform"
)

func (s *Server) handleScaffold(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ResourceType string `json:"resourceType"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ResourceType == "" {
		writeError(w, http.StatusBadRequest, "resourceType is required")
		return
	}

	result := terraform.GetScaffold(req.ResourceType)
	writeJSON(w, result)
}
