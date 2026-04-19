package terraform

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-exec/tfexec"
)

// EvalDataResult holds the output of evaluating a data source
type EvalDataResult struct {
	Address   string                 `json:"address"`
	Values    map[string]interface{} `json:"values,omitempty"`
	Valid     bool                   `json:"valid"`
	Error     string                 `json:"error,omitempty"`
}

// EvalData runs a targeted plan for a data source to fetch its values.
// Data sources are evaluated during plan, so the values appear in the plan output.
func EvalData(workspacePath string, address string) (*EvalDataResult, error) {
	tf, err := newTF(workspacePath)
	if err != nil {
		return nil, err
	}

	// Auto-init
	if err := tf.Init(context.Background(), tfexec.Backend(false)); err != nil {
		return &EvalDataResult{
			Address: address,
			Valid:   false,
			Error:   fmt.Sprintf("init failed: %s", err),
		}, nil
	}

	// Run a full plan (data sources need their dependencies resolved)
	planFile := filepath.Join(workspacePath, "tfplan.eval")
	_, planErr := tf.Plan(context.Background(),
		tfexec.Out(planFile),
	)

	// Plan might fail but still produce output with data sources evaluated
	plan, err := tf.ShowPlanFile(context.Background(), planFile)
	if err != nil {
		errMsg := "reading plan failed"
		if planErr != nil {
			errMsg = fmt.Sprintf("plan failed: %s", planErr)
		}
		return &EvalDataResult{
			Address: address,
			Valid:   false,
			Error:   errMsg,
		}, nil
	}

	// 1. Check resource_changes for the data source
	if plan.ResourceChanges != nil {
		for _, rc := range plan.ResourceChanges {
			if rc.Address == address {
				result := &EvalDataResult{Address: address, Valid: true}
				if rc.Change.After != nil {
					if m, ok := rc.Change.After.(map[string]interface{}); ok {
						result.Values = m
					}
				}
				return result, nil
			}
		}
	}

	// 2. Check planned_values.root_module.resources
	if plan.PlannedValues != nil && plan.PlannedValues.RootModule != nil {
		for _, r := range plan.PlannedValues.RootModule.Resources {
			if r.Address == address {
				result := &EvalDataResult{Address: address, Valid: true}
				if r.AttributeValues != nil {
					result.Values = r.AttributeValues
				}
				return result, nil
			}
		}
	}

	// 3. Check prior_state.values.root_module.resources
	if plan.PriorState != nil && plan.PriorState.Values != nil && plan.PriorState.Values.RootModule != nil {
		for _, r := range plan.PriorState.Values.RootModule.Resources {
			if r.Address == address {
				result := &EvalDataResult{Address: address, Valid: true}
				if r.AttributeValues != nil {
					result.Values = r.AttributeValues
				}
				return result, nil
			}
		}
	}

	errMsg := "data source not found in plan output"
	if planErr != nil {
		errMsg = fmt.Sprintf("plan had errors: %s", planErr)
	}

	return &EvalDataResult{
		Address: address,
		Valid:   false,
		Error:   errMsg,
	}, nil
}

// EvalExpression uses terraform console to evaluate an arbitrary expression
func EvalExpression(workspacePath string, expression string) (string, error) {
	execPath, err := exec.LookPath("terraform")
	if err != nil {
		return "", fmt.Errorf("terraform binary not found: %w", err)
	}

	cmd := exec.Command(execPath, "console")
	cmd.Dir = workspacePath
	cmd.Stdin = strings.NewReader(expression + "\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("terraform console failed: %s\nOutput: %s", err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}
