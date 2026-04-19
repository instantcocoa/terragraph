package graph

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// Extractor builds a WorkspaceGraph from parsed HCL files
type Extractor struct {
	nodes []GraphNode
	edges []GraphEdge
	// maps address -> node ID for edge resolution
	addressIndex map[string]string
}

// NewExtractor creates a new graph extractor
func NewExtractor() *Extractor {
	return &Extractor{
		addressIndex: make(map[string]string),
	}
}

// ExtractFile processes a single parsed file and adds nodes/edges
func (e *Extractor) ExtractFile(filename string, body *hclsyntax.Body, source []byte) {
	for _, block := range body.Blocks {
		switch block.Type {
		case "resource":
			e.extractResource(filename, block, source)
		case "data":
			e.extractDataSource(filename, block, source)
		case "variable":
			e.extractVariable(filename, block, source)
		case "output":
			e.extractOutput(filename, block, source)
		case "locals":
			e.extractLocals(filename, block, source)
		case "module":
			e.extractModule(filename, block, source)
		case "provider":
			e.extractProvider(filename, block, source)
		case "terraform":
			e.extractTerraform(filename, block, source)
		}
	}
}

// Build finalizes the graph, resolving edges from references
func (e *Extractor) Build() WorkspaceGraph {
	e.resolveEdges()
	return WorkspaceGraph{
		Nodes: e.nodes,
		Edges: e.edges,
	}
}

func (e *Extractor) extractResource(file string, block *hclsyntax.Block, source []byte) {
	if len(block.Labels) < 2 {
		return
	}
	resType := block.Labels[0]
	name := block.Labels[1]
	address := fmt.Sprintf("%s.%s", resType, name)
	id := fmt.Sprintf("resource.%s", address)

	node := GraphNode{
		ID:           id,
		Kind:         KindResource,
		ResourceType: resType,
		Name:         name,
		Address:      address,
		Provider:     inferProvider(resType),
		Source:       makeSpan(file, block),
		RawHCL:       extractBlockSource(source, block),
	}

	node.Attributes, node.NestedBlocks, node.DependsOn = e.extractBlockContents(block.Body, source)
	e.nodes = append(e.nodes, node)
	e.addressIndex[address] = id
}

func (e *Extractor) extractDataSource(file string, block *hclsyntax.Block, source []byte) {
	if len(block.Labels) < 2 {
		return
	}
	resType := block.Labels[0]
	name := block.Labels[1]
	address := fmt.Sprintf("data.%s.%s", resType, name)
	id := fmt.Sprintf("data.%s.%s", resType, name)

	node := GraphNode{
		ID:           id,
		Kind:         KindData,
		ResourceType: resType,
		Name:         name,
		Address:      address,
		Provider:     inferProvider(resType),
		Source:       makeSpan(file, block),
		RawHCL:       extractBlockSource(source, block),
	}

	node.Attributes, node.NestedBlocks, node.DependsOn = e.extractBlockContents(block.Body, source)
	e.nodes = append(e.nodes, node)
	e.addressIndex[address] = id
}

func (e *Extractor) extractVariable(file string, block *hclsyntax.Block, source []byte) {
	if len(block.Labels) < 1 {
		return
	}
	name := block.Labels[0]
	address := fmt.Sprintf("var.%s", name)
	id := fmt.Sprintf("variable.%s", name)

	node := GraphNode{
		ID:      id,
		Kind:    KindVariable,
		Name:    name,
		Address: address,
		Source:  makeSpan(file, block),
		RawHCL:  extractBlockSource(source, block),
	}

	// Extract description, type, default
	for _, attr := range block.Body.Attributes {
		switch attr.Name {
		case "description":
			node.Description = evalStringExpr(attr.Expr)
		case "type":
			node.VarType = exprToString(attr.Expr, source)
		case "default":
			node.Default = evalExpr(attr.Expr)
		}
	}

	e.nodes = append(e.nodes, node)
	e.addressIndex[address] = id
}

func (e *Extractor) extractOutput(file string, block *hclsyntax.Block, source []byte) {
	if len(block.Labels) < 1 {
		return
	}
	name := block.Labels[0]
	address := fmt.Sprintf("output.%s", name)
	id := fmt.Sprintf("output.%s", name)

	node := GraphNode{
		ID:      id,
		Kind:    KindOutput,
		Name:    name,
		Address: address,
		Source:  makeSpan(file, block),
		RawHCL:  extractBlockSource(source, block),
	}

	for _, attr := range block.Body.Attributes {
		switch attr.Name {
		case "description":
			node.Description = evalStringExpr(attr.Expr)
		case "value":
			node.Attributes = append(node.Attributes, Attribute{
				Name:       "value",
				Expression: exprToString(attr.Expr, source),
				References: extractReferences(attr.Expr),
			})
		}
	}

	e.nodes = append(e.nodes, node)
	e.addressIndex[address] = id
}

func (e *Extractor) extractLocals(file string, block *hclsyntax.Block, source []byte) {
	for _, attr := range block.Body.Attributes {
		name := attr.Name
		address := fmt.Sprintf("local.%s", name)
		id := fmt.Sprintf("local.%s", name)

		node := GraphNode{
			ID:      id,
			Kind:    KindLocal,
			Name:    name,
			Address: address,
			Source: SourceSpan{
				File:      file,
				StartLine: attr.SrcRange.Start.Line,
				EndLine:   attr.SrcRange.End.Line,
			},
			Attributes: []Attribute{{
				Name:       "value",
				Expression: exprToString(attr.Expr, source),
				Value:      evalExpr(attr.Expr),
				References: extractReferences(attr.Expr),
			}},
		}

		e.nodes = append(e.nodes, node)
		e.addressIndex[address] = id
	}
}

func (e *Extractor) extractModule(file string, block *hclsyntax.Block, source []byte) {
	if len(block.Labels) < 1 {
		return
	}
	name := block.Labels[0]
	address := fmt.Sprintf("module.%s", name)
	id := fmt.Sprintf("module.%s", name)

	node := GraphNode{
		ID:      id,
		Kind:    KindModule,
		Name:    name,
		Address: address,
		Source:  makeSpan(file, block),
		RawHCL:  extractBlockSource(source, block),
	}

	for _, attr := range block.Body.Attributes {
		switch attr.Name {
		case "source":
			node.ModuleSource = evalStringExpr(attr.Expr)
		case "version":
			node.ModuleVersion = evalStringExpr(attr.Expr)
		default:
			node.Attributes = append(node.Attributes, Attribute{
				Name:       attr.Name,
				Expression: exprToString(attr.Expr, source),
				Value:      evalExpr(attr.Expr),
				References: extractReferences(attr.Expr),
			})
		}
	}

	e.nodes = append(e.nodes, node)
	e.addressIndex[address] = id
}

func (e *Extractor) extractProvider(file string, block *hclsyntax.Block, source []byte) {
	if len(block.Labels) < 1 {
		return
	}
	name := block.Labels[0]
	id := fmt.Sprintf("provider.%s", name)

	node := GraphNode{
		ID:      id,
		Kind:    KindProvider,
		Name:    name,
		Address: fmt.Sprintf("provider.%s", name),
		Source:  makeSpan(file, block),
		RawHCL:  extractBlockSource(source, block),
	}

	node.Attributes, node.NestedBlocks, _ = e.extractBlockContents(block.Body, source)
	e.nodes = append(e.nodes, node)
}

func (e *Extractor) extractTerraform(file string, block *hclsyntax.Block, source []byte) {
	id := "terraform"
	node := GraphNode{
		ID:      id,
		Kind:    KindTerraform,
		Name:    "terraform",
		Address: "terraform",
		Source:  makeSpan(file, block),
		RawHCL:  extractBlockSource(source, block),
	}
	e.nodes = append(e.nodes, node)
}

func (e *Extractor) extractBlockContents(body *hclsyntax.Body, source []byte) ([]Attribute, []NestedBlock, []string) {
	var attrs []Attribute
	var nested []NestedBlock
	var dependsOn []string

	for _, attr := range body.Attributes {
		if attr.Name == "depends_on" {
			// Extract depends_on references
			refs := extractReferences(attr.Expr)
			dependsOn = append(dependsOn, refs...)
			continue
		}

		a := Attribute{
			Name:       attr.Name,
			Expression: exprToString(attr.Expr, source),
			Value:      evalExpr(attr.Expr),
			References: extractReferences(attr.Expr),
		}
		attrs = append(attrs, a)
	}

	for _, block := range body.Blocks {
		nb := NestedBlock{
			Type:   block.Type,
			Labels: block.Labels,
			RawHCL: extractBlockSource(source, block),
		}
		for _, attr := range block.Body.Attributes {
			nb.Attributes = append(nb.Attributes, Attribute{
				Name:       attr.Name,
				Expression: exprToString(attr.Expr, source),
				Value:      evalExpr(attr.Expr),
				References: extractReferences(attr.Expr),
			})
		}
		nested = append(nested, nb)
	}

	return attrs, nested, dependsOn
}

func (e *Extractor) resolveEdges() {
	edgeID := 0

	for _, node := range e.nodes {
		// Collect all references from attributes
		var allRefs []string
		for _, attr := range node.Attributes {
			allRefs = append(allRefs, attr.References...)
		}
		for _, nb := range node.NestedBlocks {
			for _, attr := range nb.Attributes {
				allRefs = append(allRefs, attr.References...)
			}
		}

		seen := make(map[string]bool)
		for _, ref := range allRefs {
			targetID := e.resolveRef(ref)
			if targetID == "" || targetID == node.ID || seen[targetID] {
				continue
			}
			seen[targetID] = true
			edgeID++
			e.edges = append(e.edges, GraphEdge{
				ID:     fmt.Sprintf("e%d", edgeID),
				Source: node.ID,
				Target: targetID,
				Kind:   EdgeReference,
			})
		}

		// depends_on edges
		for _, dep := range node.DependsOn {
			targetID := e.resolveRef(dep)
			if targetID == "" || targetID == node.ID || seen[targetID] {
				continue
			}
			seen[targetID] = true
			edgeID++
			e.edges = append(e.edges, GraphEdge{
				ID:     fmt.Sprintf("e%d", edgeID),
				Source: node.ID,
				Target: targetID,
				Kind:   EdgeDependsOn,
			})
		}

		// Provider edge
		if node.Provider != "" && node.Kind == KindResource || node.Kind == KindData {
			providerID := fmt.Sprintf("provider.%s", node.Provider)
			if _, exists := e.addressIndex[providerID]; exists {
				// Only add if provider node was explicitly defined
				// (not just inferred)
			}
		}
	}
}

func (e *Extractor) resolveRef(ref string) string {
	// Try direct address lookup
	if id, ok := e.addressIndex[ref]; ok {
		return id
	}
	// Try trimming attribute access: aws_instance.web.id -> aws_instance.web
	parts := strings.Split(ref, ".")
	for i := len(parts); i >= 2; i-- {
		candidate := strings.Join(parts[:i], ".")
		if id, ok := e.addressIndex[candidate]; ok {
			return id
		}
	}
	return ""
}

// Helper functions

func makeSpan(file string, block *hclsyntax.Block) SourceSpan {
	return SourceSpan{
		File:      file,
		StartLine: block.Range().Start.Line,
		EndLine:   block.Range().End.Line,
		StartCol:  block.Range().Start.Column,
		EndCol:    block.Range().End.Column,
	}
}

func extractBlockSource(source []byte, block *hclsyntax.Block) string {
	start := block.Range().Start.Byte
	end := block.Range().End.Byte
	if start >= 0 && end <= len(source) && start < end {
		return string(source[start:end])
	}
	return ""
}

func inferProvider(resourceType string) string {
	parts := strings.SplitN(resourceType, "_", 2)
	if len(parts) >= 1 {
		return parts[0]
	}
	return ""
}

func extractReferences(expr hclsyntax.Expression) []string {
	var refs []string
	for _, traversal := range expr.Variables() {
		var parts []string
		for _, t := range traversal {
			switch tt := t.(type) {
			case hcl.TraverseRoot:
				parts = append(parts, tt.Name)
			case hcl.TraverseAttr:
				parts = append(parts, tt.Name)
			}
		}
		if len(parts) > 0 {
			refs = append(refs, strings.Join(parts, "."))
		}
	}
	return refs
}

func evalStringExpr(expr hclsyntax.Expression) string {
	val, diags := expr.Value(nil)
	if diags.HasErrors() || val.Type() != cty.String {
		return ""
	}
	return val.AsString()
}

func evalExpr(expr hclsyntax.Expression) interface{} {
	val, diags := expr.Value(nil)
	if diags.HasErrors() {
		return nil
	}
	return ctyToInterface(val)
}

func ctyToInterface(val cty.Value) interface{} {
	if val.IsNull() {
		return nil
	}
	ty := val.Type()
	switch {
	case ty == cty.String:
		return val.AsString()
	case ty == cty.Number:
		bf := val.AsBigFloat()
		f, _ := bf.Float64()
		return f
	case ty == cty.Bool:
		return val.True()
	case ty.IsListType() || ty.IsTupleType() || ty.IsSetType():
		var items []interface{}
		it := val.ElementIterator()
		for it.Next() {
			_, v := it.Element()
			items = append(items, ctyToInterface(v))
		}
		return items
	case ty.IsMapType() || ty.IsObjectType():
		m := make(map[string]interface{})
		it := val.ElementIterator()
		for it.Next() {
			k, v := it.Element()
			m[k.AsString()] = ctyToInterface(v)
		}
		return m
	default:
		return nil
	}
}

func exprToString(expr hclsyntax.Expression, source []byte) string {
	rng := expr.Range()
	if rng.Start.Byte >= 0 && rng.End.Byte <= len(source) && rng.Start.Byte < rng.End.Byte {
		return string(source[rng.Start.Byte:rng.End.Byte])
	}
	return ""
}
