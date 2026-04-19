package terraform

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// LSPClient manages a terraform-ls process and communicates via JSON-RPC
type LSPClient struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	reader  *bufio.Reader
	nextID  int
	running bool
}

// LSPDiagnostic represents a diagnostic from terraform-ls
type LSPDiagnostic struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Col      int    `json:"col"`
	EndLine  int    `json:"endLine"`
	EndCol   int    `json:"endCol"`
	Severity int    `json:"severity"` // 1=error, 2=warning, 3=info, 4=hint
	Message  string `json:"message"`
	Source   string `json:"source"`
}

// LSPHoverResult holds hover information
type LSPHoverResult struct {
	Contents string `json:"contents"`
}

// StartLSP starts a terraform-ls process for the given workspace
func StartLSP(workspacePath string) (*LSPClient, error) {
	lsPath, err := exec.LookPath("terraform-ls")
	if err != nil {
		return nil, fmt.Errorf("terraform-ls not found: %w", err)
	}

	cmd := exec.CommandContext(context.Background(), lsPath, "serve")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting terraform-ls: %w", err)
	}

	client := &LSPClient{
		cmd:     cmd,
		stdin:   stdin,
		reader:  bufio.NewReader(stdout),
		nextID:  1,
		running: true,
	}

	// Initialize the LSP connection
	if err := client.initialize(workspacePath); err != nil {
		client.Close()
		return nil, fmt.Errorf("LSP initialize failed: %w", err)
	}

	return client, nil
}

func (c *LSPClient) initialize(workspacePath string) error {
	_, err := c.sendRequest("initialize", map[string]interface{}{
		"processId": nil,
		"rootUri":   "file://" + workspacePath,
		"capabilities": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"hover": map[string]interface{}{
					"contentFormat": []string{"plaintext"},
				},
				"completion": map[string]interface{}{
					"completionItem": map[string]interface{}{
						"snippetSupport": false,
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}

	// Send initialized notification
	return c.sendNotification("initialized", map[string]interface{}{})
}

// GetDiagnostics opens a file and returns diagnostics from terraform-ls
func (c *LSPClient) GetDiagnostics(filePath string, content string) ([]LSPDiagnostic, error) {
	uri := "file://" + filePath

	// Open the document
	err := c.sendNotification("textDocument/didOpen", map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":        uri,
			"languageId": "terraform",
			"version":    1,
			"text":       content,
		},
	})
	if err != nil {
		return nil, err
	}

	// Read notifications until we get publishDiagnostics
	var diagnostics []LSPDiagnostic
	for i := 0; i < 20; i++ {
		msg, err := c.readMessage()
		if err != nil {
			break
		}

		var notification struct {
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal(msg, &notification); err != nil {
			continue
		}

		if notification.Method == "textDocument/publishDiagnostics" {
			var params struct {
				URI         string `json:"uri"`
				Diagnostics []struct {
					Range struct {
						Start struct{ Line, Character int } `json:"start"`
						End   struct{ Line, Character int } `json:"end"`
					} `json:"range"`
					Severity int    `json:"severity"`
					Message  string `json:"message"`
					Source   string `json:"source"`
				} `json:"diagnostics"`
			}
			if err := json.Unmarshal(notification.Params, &params); err != nil {
				continue
			}

			for _, d := range params.Diagnostics {
				diagnostics = append(diagnostics, LSPDiagnostic{
					File:     filePath,
					Line:     d.Range.Start.Line + 1,
					Col:      d.Range.Start.Character + 1,
					EndLine:  d.Range.End.Line + 1,
					EndCol:   d.Range.End.Character + 1,
					Severity: d.Severity,
					Message:  d.Message,
					Source:   d.Source,
				})
			}
			break
		}
	}

	return diagnostics, nil
}

// Hover requests hover information for a position
func (c *LSPClient) Hover(filePath string, line, col int) (*LSPHoverResult, error) {
	uri := "file://" + filePath

	resp, err := c.sendRequest("textDocument/hover", map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": uri},
		"position":     map[string]interface{}{"line": line - 1, "character": col - 1},
	})
	if err != nil {
		return nil, err
	}

	if resp == nil {
		return nil, nil
	}

	var result struct {
		Contents interface{} `json:"contents"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	var text string
	switch v := result.Contents.(type) {
	case string:
		text = v
	case map[string]interface{}:
		if val, ok := v["value"]; ok {
			text = fmt.Sprint(val)
		}
	}

	if text == "" {
		return nil, nil
	}

	return &LSPHoverResult{Contents: text}, nil
}

func (c *LSPClient) sendRequest(method string, params interface{}) (json.RawMessage, error) {
	c.mu.Lock()
	id := c.nextID
	c.nextID++
	c.mu.Unlock()

	paramsJSON, _ := json.Marshal(params)
	msg := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  json.RawMessage(paramsJSON),
	}

	if err := c.writeMessage(msg); err != nil {
		return nil, err
	}

	// Read response
	for i := 0; i < 30; i++ {
		raw, err := c.readMessage()
		if err != nil {
			return nil, err
		}

		var resp struct {
			ID     *int            `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(raw, &resp); err != nil {
			continue
		}

		if resp.ID != nil && *resp.ID == id {
			if resp.Error != nil {
				return nil, fmt.Errorf("LSP error: %s", resp.Error.Message)
			}
			return resp.Result, nil
		}
	}

	return nil, fmt.Errorf("no response for request %d", id)
}

func (c *LSPClient) sendNotification(method string, params interface{}) error {
	paramsJSON, _ := json.Marshal(params)
	msg := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  json.RawMessage(paramsJSON),
	}
	return c.writeMessage(msg)
}

func (c *LSPClient) writeMessage(msg interface{}) error {
	body, _ := json.Marshal(msg)
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, err := c.stdin.Write([]byte(header)); err != nil {
		return err
	}
	_, err := c.stdin.Write(body)
	return err
}

func (c *LSPClient) readMessage() ([]byte, error) {
	// Read headers
	var contentLength int
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "Content-Length:") {
			lenStr := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			contentLength, _ = strconv.Atoi(lenStr)
		}
	}

	if contentLength == 0 {
		return nil, fmt.Errorf("no content length")
	}

	body := make([]byte, contentLength)
	_, err := io.ReadFull(c.reader, body)
	return body, err
}

// Close shuts down the LSP server
func (c *LSPClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return
	}
	c.running = false
	c.stdin.Close()
	c.cmd.Process.Kill()
	c.cmd.Wait()
}

// IsLSPAvailable checks if terraform-ls is installed
func IsLSPAvailable() bool {
	_, err := exec.LookPath("terraform-ls")
	return err == nil
}
