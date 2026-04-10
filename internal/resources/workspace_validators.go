package resources

import (
	"fmt"
)

type workspaceNodeLayoutConfig struct {
	Mode      string
	Direction string
	ColumnGap float64
	RowGap    float64
}

// validateWorkspaceNodeLayout performs lightweight validation for node_layout config.
// This helper is intentionally minimal and is not yet wired into the resource schema.
func validateWorkspaceNodeLayout(l workspaceNodeLayoutConfig) error {
	if l.Mode != "" && l.Mode != "dag" {
		return fmt.Errorf("mode must be dag")
	}
	if l.Direction != "" && l.Direction != "left_to_right" && l.Direction != "right_to_left" {
		return fmt.Errorf("direction must be left_to_right or right_to_left")
	}
	return nil
}

type workflowStyleConfig struct {
	GroupColor    string
	TitleFontSize int
}

// validateWorkflowStyle performs minimal validation for workflow style.
// This helper is intentionally minimal and is not yet wired into the resource schema.
func validateWorkflowStyle(s workflowStyleConfig) error {
	if s.GroupColor == "" {
		return fmt.Errorf("group_color must not be empty")
	}
	return nil
}
