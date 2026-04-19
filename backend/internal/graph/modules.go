package graph

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// ExpandModules iterates through nodes, and for any module node with a local
// source path, parses the module directory and attaches its resources as children.
func ExpandModules(workspacePath string, nodes []GraphNode) []GraphNode {
	for i, node := range nodes {
		if node.Kind != KindModule {
			continue
		}
		if node.ModuleSource == "" {
			continue
		}
		if !isLocalSource(node.ModuleSource) {
			continue
		}

		modulePath := filepath.Join(workspacePath, node.ModuleSource)
		modulePath = filepath.Clean(modulePath)

		info, err := os.Stat(modulePath)
		if err != nil || !info.IsDir() {
			log.Printf("module %s: source path %s is not a directory", node.Address, modulePath)
			continue
		}

		children, err := parseModuleChildren(modulePath, node.Address)
		if err != nil {
			log.Printf("module %s: failed to parse children: %v", node.Address, err)
			continue
		}

		nodes[i].Children = children
	}
	return nodes
}

// isLocalSource returns true if the module source is a relative local path.
func isLocalSource(source string) bool {
	return strings.HasPrefix(source, "./") || strings.HasPrefix(source, "../")
}

// parseModuleChildren parses .tf files in the module directory and returns
// graph nodes with addresses prefixed by the parent module address.
func parseModuleChildren(moduleDir string, parentAddress string) ([]GraphNode, error) {
	entries, err := os.ReadDir(moduleDir)
	if err != nil {
		return nil, fmt.Errorf("reading module dir: %w", err)
	}

	extractor := NewExtractor()

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tf") {
			continue
		}

		path := filepath.Join(moduleDir, entry.Name())
		src, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		file, diags := hclsyntax.ParseConfig(src, entry.Name(), hcl.Pos{Line: 1, Column: 1})
		if diags.HasErrors() {
			continue
		}

		body, ok := file.Body.(*hclsyntax.Body)
		if !ok {
			continue
		}

		extractor.ExtractFile(entry.Name(), body, src)
	}

	result := extractor.Build()

	// Prefix all child node IDs and addresses with the parent module address
	for i := range result.Nodes {
		result.Nodes[i].ID = fmt.Sprintf("%s.%s", parentAddress, result.Nodes[i].ID)
		result.Nodes[i].Address = fmt.Sprintf("%s.%s", parentAddress, result.Nodes[i].Address)
	}

	return result.Nodes, nil
}
