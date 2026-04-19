package parser

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// AddProviderRequest describes adding a provider to the workspace
type AddProviderRequest struct {
	WorkspacePath string            `json:"workspacePath"`
	File          string            `json:"file"`
	Provider      string            `json:"provider"`      // short name e.g. "aws"
	Source        string            `json:"source"`         // registry source e.g. "hashicorp/aws"
	Version       string            `json:"version"`       // version constraint e.g. "~> 5.0"
	Attributes    map[string]string `json:"attributes"`    // provider config e.g. {"region": "\"us-east-1\""}
}

// AddProvider creates a provider block and ensures required_providers is configured
func AddProvider(req AddProviderRequest) (*BlockResult, error) {
	filePath := filepath.Join(req.WorkspacePath, req.File)

	var f *hclwrite.File

	// Read existing file or create new
	src, err := os.ReadFile(filePath)
	if err != nil {
		f = hclwrite.NewEmptyFile()
	} else {
		var diags hcl.Diagnostics
		f, diags = hclwrite.ParseConfig(src, req.File, hcl.Pos{Line: 1, Column: 1})
		if diags.HasErrors() {
			return nil, fmt.Errorf("parsing HCL: %s", diags.Error())
		}
	}

	body := f.Body()

	// 1. Ensure terraform { required_providers { ... } } exists
	ensureRequiredProvider(body, req.Provider, req.Source, req.Version)

	// 2. Add provider block
	body.AppendNewline()
	providerBlock := body.AppendNewBlock("provider", []string{req.Provider})
	providerBody := providerBlock.Body()

	for name, value := range req.Attributes {
		setAttributeValue(providerBody, name, value)
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

// ensureRequiredProvider adds or updates the terraform required_providers block
func ensureRequiredProvider(body *hclwrite.Body, provider, source, version string) {
	// Find existing terraform block
	var terraformBlock *hclwrite.Block
	for _, block := range body.Blocks() {
		if block.Type() == "terraform" {
			terraformBlock = block
			break
		}
	}

	// Create terraform block if it doesn't exist
	if terraformBlock == nil {
		terraformBlock = body.AppendNewBlock("terraform", nil)
		body.AppendNewline()
	}

	terraformBody := terraformBlock.Body()

	// Find or create required_providers block
	var rpBlock *hclwrite.Block
	for _, block := range terraformBody.Blocks() {
		if block.Type() == "required_providers" {
			rpBlock = block
			break
		}
	}

	if rpBlock == nil {
		rpBlock = terraformBody.AppendNewBlock("required_providers", nil)
	}

	rpBody := rpBlock.Body()

	// Set the provider entry
	providerValue := fmt.Sprintf(`{
      source  = "%s"
      version = "%s"
    }`, source, version)
	setAttributeValue(rpBody, provider, providerValue)
}
