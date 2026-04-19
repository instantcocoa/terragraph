package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// PatchRequest describes a single attribute patch operation
type PatchRequest struct {
	WorkspacePath string `json:"workspacePath"`
	File          string `json:"file"`
	Address       string `json:"address"`
	Attribute     string `json:"attribute"`
	Value         string `json:"value"`
}

// PatchResult holds the result of a patch operation
type PatchResult struct {
	File    string `json:"file"`
	Content string `json:"content"`
}

// PatchAttribute patches a single attribute in a Terraform file.
// It parses the address to find the right block, sets the attribute value,
// and writes the file back, preserving formatting.
func PatchAttribute(req PatchRequest) (*PatchResult, error) {
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

	// Special case: locals block has no labels; the attribute name is part of the address
	if blockType == "locals" {
		block := findBlock(body, "locals", nil)
		if block == nil {
			return nil, fmt.Errorf("locals block not found")
		}
		setAttributeValue(block.Body(), req.Attribute, req.Value)
	} else {
		block := findBlock(body, blockType, labels)
		if block == nil {
			return nil, fmt.Errorf("block not found for address %q (type=%s labels=%v)", req.Address, blockType, labels)
		}
		setAttributeValue(block.Body(), req.Attribute, req.Value)
	}

	result := f.Bytes()

	if err := os.WriteFile(filePath, result, 0644); err != nil {
		return nil, fmt.Errorf("writing patched file: %w", err)
	}

	return &PatchResult{
		File:    req.File,
		Content: string(result),
	}, nil
}

// parseAddress converts a Terraform address into block type and labels.
//
//	"aws_instance.web"   -> ("resource", ["aws_instance", "web"])
//	"var.name"           -> ("variable", ["name"])
//	"local.name"         -> ("locals", [])  (attribute is "name")
//	"output.name"        -> ("output", ["name"])
//	"data.aws_ami.latest"-> ("data", ["aws_ami", "latest"])
//	"module.vpc"         -> ("module", ["vpc"])
func parseAddress(addr string) (blockType string, labels []string, err error) {
	parts := strings.SplitN(addr, ".", 3)
	if len(parts) < 2 {
		return "", nil, fmt.Errorf("invalid address %q: expected at least two dot-separated parts", addr)
	}

	switch parts[0] {
	case "var":
		return "variable", []string{parts[1]}, nil
	case "local":
		// locals block has no labels; the attribute is identified by name
		return "locals", nil, nil
	case "output":
		return "output", []string{parts[1]}, nil
	case "data":
		if len(parts) < 3 {
			return "", nil, fmt.Errorf("invalid data address %q: expected data.<type>.<name>", addr)
		}
		return "data", []string{parts[1], parts[2]}, nil
	case "module":
		return "module", []string{parts[1]}, nil
	default:
		// Assume resource: "aws_instance.web" -> resource "aws_instance" "web"
		return "resource", []string{parts[0], parts[1]}, nil
	}
}

// findBlock finds the first block matching the given type and labels in the body.
func findBlock(body *hclwrite.Body, blockType string, labels []string) *hclwrite.Block {
	for _, block := range body.Blocks() {
		if block.Type() != blockType {
			continue
		}
		bl := block.Labels()
		if len(bl) != len(labels) {
			continue
		}
		match := true
		for i := range labels {
			if bl[i] != labels[i] {
				match = false
				break
			}
		}
		if match {
			return block
		}
	}
	return nil
}

// setAttributeValue sets an attribute using raw HCL expression tokens.
func setAttributeValue(body *hclwrite.Body, name string, rawValue string) {
	tokens := hclwrite.TokensForTraversal(nil)
	// Parse the raw value as HCL tokens by wrapping in a temporary config
	tmpSrc := []byte(fmt.Sprintf("v = %s\n", rawValue))
	tmpFile, diags := hclwrite.ParseConfig(tmpSrc, "tmp", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		// Fallback: write as raw tokens directly
		body.SetAttributeRaw(name, hclwrite.Tokens{
			{
				Type:  9, // hclsyntax.TokenIdent
				Bytes: []byte(rawValue),
			},
		})
		return
	}
	_ = tokens
	// Extract the attribute's expression tokens from the parsed tmp file
	attr := tmpFile.Body().GetAttribute("v")
	if attr == nil {
		return
	}
	body.SetAttributeRaw(name, attr.Expr().BuildTokens(nil))
}
