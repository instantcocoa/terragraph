package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// AddBlockRequest describes a new block to add
type AddBlockRequest struct {
	WorkspacePath string            `json:"workspacePath"`
	File          string            `json:"file"`                   // target file, e.g. "main.tf"
	BlockType     string            `json:"blockType"`              // "resource", "data", "variable", "output"
	ResourceType  string            `json:"resourceType,omitempty"` // e.g. "aws_instance" (for resource/data)
	Name          string            `json:"name"`                   // e.g. "web"
	Attributes    map[string]string `json:"attributes,omitempty"`   // initial attribute values as raw HCL
}

// RemoveBlockRequest describes a block to remove
type RemoveBlockRequest struct {
	WorkspacePath string `json:"workspacePath"`
	File          string `json:"file"`
	Address       string `json:"address"` // e.g. "aws_instance.web", "var.name", "output.id"
}

// BlockResult holds the result of a block add/remove operation
type BlockResult struct {
	File    string `json:"file"`
	Content string `json:"content"`
}

// AddBlock adds a new Terraform block to a file.
func AddBlock(req AddBlockRequest) (*BlockResult, error) {
	filePath := filepath.Join(req.WorkspacePath, req.File)

	var f *hclwrite.File

	src, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			f = hclwrite.NewEmptyFile()
		} else {
			return nil, fmt.Errorf("reading file %s: %w", req.File, err)
		}
	} else {
		var diags hcl.Diagnostics
		f, diags = hclwrite.ParseConfig(src, req.File, hcl.Pos{Line: 1, Column: 1})
		if diags.HasErrors() {
			return nil, fmt.Errorf("parsing HCL: %s", diags.Error())
		}
	}

	body := f.Body()

	// Add a blank line before the new block for readability
	body.AppendNewline()

	var block *hclwrite.Block
	switch req.BlockType {
	case "resource":
		if req.ResourceType == "" {
			return nil, fmt.Errorf("resourceType is required for resource blocks")
		}
		block = body.AppendNewBlock("resource", []string{req.ResourceType, req.Name})
	case "data":
		if req.ResourceType == "" {
			return nil, fmt.Errorf("resourceType is required for data blocks")
		}
		block = body.AppendNewBlock("data", []string{req.ResourceType, req.Name})
	case "variable":
		block = body.AppendNewBlock("variable", []string{req.Name})
		// Set default type if no attributes override it
		if _, hasType := req.Attributes["type"]; !hasType {
			setAttributeValue(block.Body(), "type", "string")
		}
	case "output":
		block = body.AppendNewBlock("output", []string{req.Name})
		// Set default value if no attributes override it
		if _, hasValue := req.Attributes["value"]; !hasValue {
			setAttributeValue(block.Body(), "value", `""`)
		}
	default:
		return nil, fmt.Errorf("unsupported block type: %s", req.BlockType)
	}

	// Set any provided initial attributes
	for name, value := range req.Attributes {
		setAttributeValue(block.Body(), name, value)
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

// RemoveBlock removes a Terraform block from a file.
func RemoveBlock(req RemoveBlockRequest) (*BlockResult, error) {
	filePath := filepath.Join(req.WorkspacePath, req.File)

	src, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading file %s: %w", req.File, err)
	}

	f, diags := hclwrite.ParseConfig(src, req.File, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("parsing HCL: %s", diags.Error())
	}

	body := f.Body()

	blockType, labels, err := parseAddress(req.Address)
	if err != nil {
		return nil, err
	}

	// Special case: locals - remove just the attribute, not the entire block
	if blockType == "locals" {
		parts := strings.SplitN(req.Address, ".", 3)
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid locals address %q", req.Address)
		}
		attrName := parts[1]
		localsBlock := findBlock(body, "locals", nil)
		if localsBlock == nil {
			return nil, fmt.Errorf("locals block not found")
		}
		localsBody := localsBlock.Body()
		if localsBody.GetAttribute(attrName) == nil {
			return nil, fmt.Errorf("attribute %q not found in locals block", attrName)
		}
		localsBody.RemoveAttribute(attrName)

		// If the locals block is now empty, remove the entire block
		if len(localsBody.Attributes()) == 0 && len(localsBody.Blocks()) == 0 {
			body.RemoveBlock(localsBlock)
		}
	} else {
		block := findBlock(body, blockType, labels)
		if block == nil {
			return nil, fmt.Errorf("block not found for address %q", req.Address)
		}
		body.RemoveBlock(block)
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
