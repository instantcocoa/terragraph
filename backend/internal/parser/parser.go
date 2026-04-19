package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/terragraph/backend/internal/graph"
)

// ParsedFile holds the parsed representations of a single .tf file
type ParsedFile struct {
	Path      string
	ReadBody  *hclsyntax.Body
	WriteFile *hclwrite.File
	Source    []byte
}

// ParseWorkspace parses all .tf files in the given directory
func ParseWorkspace(dir string) ([]ParsedFile, []graph.Diagnostic, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("reading workspace dir: %w", err)
	}

	var files []ParsedFile
	var diags []graph.Diagnostic

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tf") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		src, err := os.ReadFile(path)
		if err != nil {
			diags = append(diags, graph.Diagnostic{
				Severity: "error",
				Summary:  fmt.Sprintf("Failed to read file: %s", entry.Name()),
				Detail:   err.Error(),
			})
			continue
		}

		readFile, readDiags := hclsyntax.ParseConfig(src, entry.Name(), hcl.Pos{Line: 1, Column: 1})
		if readDiags.HasErrors() {
			for _, d := range readDiags {
				diag := graph.Diagnostic{
					Severity: "error",
					Summary:  d.Summary,
					Detail:   d.Detail,
				}
				if d.Subject != nil {
					diag.Range = &graph.SourceSpan{
						File:      entry.Name(),
						StartLine: d.Subject.Start.Line,
						EndLine:   d.Subject.End.Line,
						StartCol:  d.Subject.Start.Column,
						EndCol:    d.Subject.End.Column,
					}
				}
				diags = append(diags, diag)
			}
			continue
		}

		writeFile, writeDiags := hclwrite.ParseConfig(src, entry.Name(), hcl.Pos{Line: 1, Column: 1})
		if writeDiags.HasErrors() {
			// Non-fatal: we can still read even if write parse fails
			_ = writeFile
		}

		body, ok := readFile.Body.(*hclsyntax.Body)
		if !ok {
			continue
		}

		files = append(files, ParsedFile{
			Path:      entry.Name(),
			ReadBody:  body,
			WriteFile: writeFile,
			Source:    src,
		})
	}

	return files, diags, nil
}
