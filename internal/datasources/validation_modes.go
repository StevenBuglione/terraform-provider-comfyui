package datasources

import (
	"fmt"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/validation"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type validationMode string

const (
	validationModeFragment            validationMode = "fragment"
	validationModeWorkspaceFragment   validationMode = "workspace_fragment"
	validationModeExecutableWorkflow  validationMode = "executable_workflow"
	validationModeExecutableWorkspace validationMode = "executable_workspace"
)

func (m validationMode) toValidationMode() validation.ValidationMode {
	return validation.ValidationMode(m)
}

func parsePromptValidationMode(value types.String) (validationMode, error) {
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return validationModeExecutableWorkflow, nil
	}
	mode := validationMode(value.ValueString())
	switch mode {
	case validationModeFragment, validationModeExecutableWorkflow:
		return mode, nil
	default:
		return "", fmt.Errorf("prompt validation mode must be one of %q or %q", validationModeFragment, validationModeExecutableWorkflow)
	}
}

func parseWorkspaceValidationMode(value types.String) (validationMode, error) {
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return validationModeExecutableWorkspace, nil
	}
	mode := validationMode(value.ValueString())
	switch mode {
	case validationModeWorkspaceFragment, validationModeExecutableWorkspace:
		return mode, nil
	default:
		return "", fmt.Errorf("workspace validation mode must be one of %q or %q", validationModeWorkspaceFragment, validationModeExecutableWorkspace)
	}
}
