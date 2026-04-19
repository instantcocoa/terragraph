package api

import (
	"net/http"
	"os/exec"
	"runtime"
	"strings"
)

func (s *Server) handlePickFolder(w http.ResponseWriter, r *http.Request) {
	path, err := pickFolder()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if path == "" {
		writeError(w, http.StatusBadRequest, "no folder selected")
		return
	}
	writeJSON(w, map[string]string{"path": path})
}

func pickFolder() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		// Use osascript to open a native folder picker
		cmd := exec.Command("osascript", "-e",
			`POSIX path of (choose folder with prompt "Select Terraform workspace")`)
		out, err := cmd.Output()
		if err != nil {
			return "", err
		}
		path := strings.TrimSpace(string(out))
		// Remove trailing slash
		path = strings.TrimRight(path, "/")
		return path, nil
	case "linux":
		// Try zenity first, then kdialog
		cmd := exec.Command("zenity", "--file-selection", "--directory",
			"--title=Select Terraform workspace")
		out, err := cmd.Output()
		if err != nil {
			// Try kdialog
			cmd = exec.Command("kdialog", "--getexistingdirectory", ".",
				"--title", "Select Terraform workspace")
			out, err = cmd.Output()
			if err != nil {
				return "", err
			}
		}
		return strings.TrimSpace(string(out)), nil
	default:
		return "", nil
	}
}

