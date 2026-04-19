package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/terragraph/backend/internal/graph"
	"github.com/zclconf/go-cty/cty"
)

// ValidateResult holds terraform validate output
type ValidateResult struct {
	Valid       bool               `json:"valid"`
	Diagnostics []graph.Diagnostic `json:"diagnostics"`
	ErrorCount  int                `json:"errorCount"`
	WarnCount   int                `json:"warningCount"`
}

// PlanResult holds terraform plan output
type PlanResult struct {
	Changes   []graph.PlanChange `json:"changes"`
	Summary   PlanSummary        `json:"summary"`
	RawOutput string             `json:"rawOutput,omitempty"`
}

// PlanSummary counts of each action type
type PlanSummary struct {
	Create  int `json:"create"`
	Update  int `json:"update"`
	Delete  int `json:"delete"`
	Replace int `json:"replace"`
}

// newTF creates a terraform-exec instance for the given workspace
func newTF(workspacePath string) (*tfexec.Terraform, error) {
	execPath, err := exec.LookPath("terraform")
	if err != nil {
		return nil, fmt.Errorf("terraform binary not found: %w", err)
	}
	tf, err := tfexec.NewTerraform(workspacePath, execPath)
	if err != nil {
		return nil, fmt.Errorf("creating terraform executor: %w", err)
	}
	return tf, nil
}

// Init runs terraform init
func Init(workspacePath string) error {
	tf, err := newTF(workspacePath)
	if err != nil {
		return err
	}
	return tf.Init(context.Background(), tfexec.Backend(false))
}

// Validate runs terraform validate
func Validate(workspacePath string) (*ValidateResult, error) {
	tf, err := newTF(workspacePath)
	if err != nil {
		return nil, err
	}

	// Auto-init
	_ = tf.Init(context.Background(), tfexec.Backend(false))

	output, err := tf.Validate(context.Background())
	if err != nil {
		return nil, fmt.Errorf("terraform validate failed: %w", err)
	}

	result := &ValidateResult{
		Valid:      output.Valid,
		ErrorCount: output.ErrorCount,
		WarnCount:  output.WarningCount,
	}

	for _, d := range output.Diagnostics {
		diag := graph.Diagnostic{
			Severity: string(d.Severity),
			Summary:  d.Summary,
			Detail:   d.Detail,
		}
		if d.Range != nil {
			diag.Range = &graph.SourceSpan{
				File:      d.Range.Filename,
				StartLine: d.Range.Start.Line,
				EndLine:   d.Range.End.Line,
				StartCol:  d.Range.Start.Column,
				EndCol:    d.Range.End.Column,
			}
		}
		result.Diagnostics = append(result.Diagnostics, diag)
	}

	return result, nil
}

// Plan runs terraform plan and returns structured output
func Plan(workspacePath string) (*PlanResult, error) {
	tf, err := newTF(workspacePath)
	if err != nil {
		return nil, err
	}

	// Auto-init
	if err := tf.Init(context.Background(), tfexec.Backend(false)); err != nil {
		return &PlanResult{}, fmt.Errorf("auto-init failed: %w", err)
	}

	planFile := filepath.Join(workspacePath, "tfplan.out")
	_, err = tf.Plan(context.Background(), tfexec.Out(planFile))
	if err != nil {
		// Get raw output for display
		rawOutput := err.Error()
		return &PlanResult{RawOutput: rawOutput}, fmt.Errorf("terraform plan failed: %w", err)
	}

	// Read the plan file as structured JSON
	plan, err := tf.ShowPlanFile(context.Background(), planFile)
	if err != nil {
		return &PlanResult{}, fmt.Errorf("reading plan file: %w", err)
	}

	// Also get raw text output
	rawOutput, _ := tf.ShowPlanFileRaw(context.Background(), planFile)

	return convertPlan(plan, rawOutput), nil
}

func convertPlan(plan *tfjson.Plan, rawOutput string) *PlanResult {
	result := &PlanResult{
		RawOutput: rawOutput,
	}

	if plan.ResourceChanges == nil {
		return result
	}

	for _, rc := range plan.ResourceChanges {
		action := mapActions(rc.Change.Actions)
		change := graph.PlanChange{
			Address: rc.Address,
			Action:  action,
		}

		// Convert before/after to map[string]interface{}
		if rc.Change.Before != nil {
			if m, ok := rc.Change.Before.(map[string]interface{}); ok {
				change.Before = m
			}
		}
		if rc.Change.After != nil {
			if m, ok := rc.Change.After.(map[string]interface{}); ok {
				change.After = m
			}
		}
		if rc.Change.AfterUnknown != nil {
			if m, ok := rc.Change.AfterUnknown.(map[string]interface{}); ok {
				change.AfterUnknown = m
			}
		}

		result.Changes = append(result.Changes, change)

		switch action {
		case graph.PlanCreate:
			result.Summary.Create++
		case graph.PlanUpdate:
			result.Summary.Update++
		case graph.PlanDelete:
			result.Summary.Delete++
		case graph.PlanReplace:
			result.Summary.Replace++
		}
	}

	return result
}

func mapActions(actions tfjson.Actions) graph.PlanAction {
	if actions.NoOp() {
		return graph.PlanNoOp
	}
	if actions.Create() {
		return graph.PlanCreate
	}
	if actions.Delete() {
		return graph.PlanDelete
	}
	if actions.Update() {
		return graph.PlanUpdate
	}
	if actions.Replace() {
		return graph.PlanReplace
	}
	if actions.Read() {
		return graph.PlanRead
	}
	return graph.PlanNoOp
}

// SchemaAttribute represents a single attribute in a provider schema
type SchemaAttribute struct {
	Name        string      `json:"name"`
	Type        interface{} `json:"type"`
	Required    bool        `json:"required"`
	Optional    bool        `json:"optional"`
	Computed    bool        `json:"computed"`
	Description string      `json:"description,omitempty"`
	Sensitive   bool        `json:"sensitive,omitempty"`
}

// SchemaBlockType represents a nested block type
type SchemaBlockType struct {
	Name        string            `json:"name"`
	NestingMode string            `json:"nestingMode"`
	Attributes  []SchemaAttribute `json:"attributes,omitempty"`
	BlockTypes  []SchemaBlockType `json:"blockTypes,omitempty"`
	MinItems    int               `json:"minItems,omitempty"`
	MaxItems    int               `json:"maxItems,omitempty"`
}

// ResourceSchema holds the schema for a resource or data source type
type ResourceSchema struct {
	Name       string            `json:"name"`
	Provider   string            `json:"provider"`
	Attributes []SchemaAttribute `json:"attributes"`
	BlockTypes []SchemaBlockType `json:"blockTypes,omitempty"`
}

// ProviderSchemas holds all schemas keyed by provider
type ProviderSchemas struct {
	Resources   map[string]ResourceSchema `json:"resources"`
	DataSources map[string]ResourceSchema `json:"dataSources"`
}

// GetProviderSchemas runs terraform providers schema and parses with terraform-json
func GetProviderSchemas(workspacePath string) (*ProviderSchemas, error) {
	tf, err := newTF(workspacePath)
	if err != nil {
		return nil, err
	}

	// Auto-init
	if err := tf.Init(context.Background(), tfexec.Backend(false)); err != nil {
		return nil, fmt.Errorf("auto-init failed: %w", err)
	}

	schemas, err := tf.ProvidersSchema(context.Background())
	if err != nil {
		return nil, fmt.Errorf("providers schema failed: %w", err)
	}

	result := &ProviderSchemas{
		Resources:   make(map[string]ResourceSchema),
		DataSources: make(map[string]ResourceSchema),
	}

	for providerAddr, ps := range schemas.Schemas {
		providerName := shortProviderName(providerAddr)

		if ps.ResourceSchemas != nil {
			for resType, rs := range ps.ResourceSchemas {
				attrs, blocks := convertSchemaBlock(rs.Block)
				result.Resources[resType] = ResourceSchema{
					Name:       resType,
					Provider:   providerName,
					Attributes: attrs,
					BlockTypes: blocks,
				}
			}
		}

		if ps.DataSourceSchemas != nil {
			for dsType, ds := range ps.DataSourceSchemas {
				attrs, blocks := convertSchemaBlock(ds.Block)
				result.DataSources[dsType] = ResourceSchema{
					Name:       dsType,
					Provider:   providerName,
					Attributes: attrs,
					BlockTypes: blocks,
				}
			}
		}
	}

	return result, nil
}

// shortProviderName extracts the provider short name from a registry address
func shortProviderName(addr string) string {
	parts := strings.Split(addr, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return addr
}

// convertSchemaBlock converts a tfjson.SchemaBlock to our typed attributes and block types
func convertSchemaBlock(block *tfjson.SchemaBlock) ([]SchemaAttribute, []SchemaBlockType) {
	if block == nil {
		return nil, nil
	}

	var attrs []SchemaAttribute
	for name, a := range block.Attributes {
		sa := SchemaAttribute{
			Name:        name,
			Required:    a.Required,
			Optional:    a.Optional,
			Computed:    a.Computed,
			Description: a.Description,
			Sensitive:   a.Sensitive,
		}
		if !a.AttributeType.Equals(cty.NilType) {
			// Marshal the cty type to JSON for frontend consumption
			typeJSON, err := json.Marshal(a.AttributeType)
			if err == nil {
				var typeVal interface{}
				json.Unmarshal(typeJSON, &typeVal)
				sa.Type = typeVal
			}
		}
		attrs = append(attrs, sa)
	}
	sort.Slice(attrs, func(i, j int) bool {
		return attrs[i].Name < attrs[j].Name
	})

	var blocks []SchemaBlockType
	for name, bt := range block.NestedBlocks {
		childAttrs, childBlocks := convertSchemaBlock(bt.Block)
		blocks = append(blocks, SchemaBlockType{
			Name:        name,
			NestingMode: string(bt.NestingMode),
			Attributes:  childAttrs,
			BlockTypes:  childBlocks,
			MinItems:    int(bt.MinItems),
			MaxItems:    int(bt.MaxItems),
		})
	}
	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].Name < blocks[j].Name
	})

	return attrs, blocks
}
