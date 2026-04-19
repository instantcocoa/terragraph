package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// RenameBlockRequest describes a block rename operation
type RenameBlockRequest struct {
	WorkspacePath string `json:"workspacePath"`
	File          string `json:"file"`
	Address       string `json:"address"` // current address e.g. "aws_instance.web"
	NewName       string `json:"newName"` // new name e.g. "app_server"
}

// AddNestedBlockRequest describes a nested block to add
type AddNestedBlockRequest struct {
	WorkspacePath string            `json:"workspacePath"`
	File          string            `json:"file"`
	Address       string            `json:"address"`   // parent block address
	BlockType     string            `json:"blockType"` // nested block type e.g. "ingress"
	Attributes    map[string]string `json:"attributes"`
}

// RemoveNestedBlockRequest describes a nested block to remove
type RemoveNestedBlockRequest struct {
	WorkspacePath string `json:"workspacePath"`
	File          string `json:"file"`
	Address       string `json:"address"`   // parent block address
	BlockType     string `json:"blockType"` // nested block type to remove
	Index         int    `json:"index"`     // which one (0-based) if multiple
}

// RemoveAttributeRequest describes an attribute to remove from a block
type RemoveAttributeRequest struct {
	WorkspacePath string `json:"workspacePath"`
	File          string `json:"file"`
	Address       string `json:"address"`
	Attribute     string `json:"attribute"`
}

// RenameBlock renames a block by changing its last label and updates references
// across all .tf files in the workspace.
func RenameBlock(req RenameBlockRequest) (*BlockResult, error) {
	filePath := filepath.Join(req.WorkspacePath, req.File)

	src, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading file %s: %w", req.File, err)
	}

	f, diags := hclwrite.ParseConfig(src, req.File, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("parsing HCL: %s", diags.Error())
	}

	blockType, labels, err := parseAddress(req.Address)
	if err != nil {
		return nil, err
	}

	block := findBlock(f.Body(), blockType, labels)
	if block == nil {
		return nil, fmt.Errorf("block not found for address %q", req.Address)
	}

	// Build new labels: replace the last label with the new name
	newLabels := make([]string, len(labels))
	copy(newLabels, labels)
	newLabels[len(newLabels)-1] = req.NewName
	block.SetLabels(newLabels)

	result := f.Bytes()
	if err := os.WriteFile(filePath, result, 0644); err != nil {
		return nil, fmt.Errorf("writing file: %w", err)
	}

	// Build old and new reference strings for cross-file updates
	oldRef, newRef := buildRefStrings(blockType, labels, newLabels)
	if oldRef != "" && newRef != "" {
		updateReferencesInWorkspace(req.WorkspacePath, oldRef, newRef)
	}

	return &BlockResult{
		File:    req.File,
		Content: string(result),
	}, nil
}

// buildRefStrings returns the old and new Terraform reference strings for a rename.
func buildRefStrings(blockType string, oldLabels, newLabels []string) (string, string) {
	switch blockType {
	case "resource":
		if len(oldLabels) == 2 && len(newLabels) == 2 {
			return oldLabels[0] + "." + oldLabels[1], newLabels[0] + "." + newLabels[1]
		}
	case "data":
		if len(oldLabels) == 2 && len(newLabels) == 2 {
			return "data." + oldLabels[0] + "." + oldLabels[1], "data." + newLabels[0] + "." + newLabels[1]
		}
	case "variable":
		if len(oldLabels) == 1 && len(newLabels) == 1 {
			return "var." + oldLabels[0], "var." + newLabels[0]
		}
	case "output":
		if len(oldLabels) == 1 && len(newLabels) == 1 {
			return "output." + oldLabels[0], "output." + newLabels[0]
		}
	case "module":
		if len(oldLabels) == 1 && len(newLabels) == 1 {
			return "module." + oldLabels[0], "module." + newLabels[0]
		}
	}
	return "", ""
}

// updateReferencesInWorkspace scans all .tf files and replaces oldRef with newRef
// in attribute expressions.
func updateReferencesInWorkspace(workspacePath, oldRef, newRef string) {
	entries, err := os.ReadDir(workspacePath)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tf") {
			continue
		}

		filePath := filepath.Join(workspacePath, entry.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		// Only process files that contain the old reference
		if !strings.Contains(string(content), oldRef) {
			continue
		}

		updated := strings.ReplaceAll(string(content), oldRef, newRef)
		if updated != string(content) {
			os.WriteFile(filePath, []byte(updated), 0644)
		}
	}
}

// AddNestedBlock adds a nested block to a parent resource block.
func AddNestedBlock(req AddNestedBlockRequest) (*BlockResult, error) {
	filePath := filepath.Join(req.WorkspacePath, req.File)

	src, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading file %s: %w", req.File, err)
	}

	f, diags := hclwrite.ParseConfig(src, req.File, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("parsing HCL: %s", diags.Error())
	}

	blockType, labels, err := parseAddress(req.Address)
	if err != nil {
		return nil, err
	}

	block := findBlock(f.Body(), blockType, labels)
	if block == nil {
		return nil, fmt.Errorf("block not found for address %q", req.Address)
	}

	nestedBlock := block.Body().AppendNewBlock(req.BlockType, nil)
	for name, value := range req.Attributes {
		setAttributeValue(nestedBlock.Body(), name, value)
	}

	result := f.Bytes()
	if err := os.WriteFile(filePath, result, 0644); err != nil {
		return nil, fmt.Errorf("writing file: %w", err)
	}

	return &BlockResult{
		File:    req.File,
		Content: string(result),
	}, nil
}

// RemoveNestedBlock removes a nested block from a parent block by type and index.
func RemoveNestedBlock(req RemoveNestedBlockRequest) (*BlockResult, error) {
	filePath := filepath.Join(req.WorkspacePath, req.File)

	src, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading file %s: %w", req.File, err)
	}

	f, diags := hclwrite.ParseConfig(src, req.File, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("parsing HCL: %s", diags.Error())
	}

	blockType, labels, err := parseAddress(req.Address)
	if err != nil {
		return nil, err
	}

	block := findBlock(f.Body(), blockType, labels)
	if block == nil {
		return nil, fmt.Errorf("block not found for address %q", req.Address)
	}

	// Find nested blocks matching the type and select by index
	var matched []*hclwrite.Block
	for _, nb := range block.Body().Blocks() {
		if nb.Type() == req.BlockType {
			matched = append(matched, nb)
		}
	}

	if len(matched) == 0 {
		return nil, fmt.Errorf("no nested block of type %q found in %q", req.BlockType, req.Address)
	}
	if req.Index < 0 || req.Index >= len(matched) {
		return nil, fmt.Errorf("nested block index %d out of range (found %d %q blocks)", req.Index, len(matched), req.BlockType)
	}

	block.Body().RemoveBlock(matched[req.Index])

	result := f.Bytes()
	if err := os.WriteFile(filePath, result, 0644); err != nil {
		return nil, fmt.Errorf("writing file: %w", err)
	}

	return &BlockResult{
		File:    req.File,
		Content: string(result),
	}, nil
}

// RemoveAttribute removes an attribute from a block.
func RemoveAttribute(req RemoveAttributeRequest) (*BlockResult, error) {
	filePath := filepath.Join(req.WorkspacePath, req.File)

	src, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading file %s: %w", req.File, err)
	}

	f, diags := hclwrite.ParseConfig(src, req.File, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("parsing HCL: %s", diags.Error())
	}

	blockType, labels, err := parseAddress(req.Address)
	if err != nil {
		return nil, err
	}

	block := findBlock(f.Body(), blockType, labels)
	if block == nil {
		return nil, fmt.Errorf("block not found for address %q", req.Address)
	}

	if block.Body().GetAttribute(req.Attribute) == nil {
		return nil, fmt.Errorf("attribute %q not found in block %q", req.Attribute, req.Address)
	}

	block.Body().RemoveAttribute(req.Attribute)

	result := f.Bytes()
	if err := os.WriteFile(filePath, result, 0644); err != nil {
		return nil, fmt.Errorf("writing file: %w", err)
	}

	return &BlockResult{
		File:    req.File,
		Content: string(result),
	}, nil
}
