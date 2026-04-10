package resources

import (
	"fmt"
	"strings"
)

// ValidationError describes a broken inter-node connection reference.
type ValidationError struct {
	SourceNodeID    string // UUID of the node with the broken ref
	SourceClassType string // Class type of the source node
	InputName       string // Name of the input with the broken ref
	ReferencedUUID  string // The UUID that was referenced but not found
	SlotIndex       int    // The slot index referenced
}

func (e ValidationError) Error() string {
	return fmt.Sprintf(
		"node %q (%s) input %q references unknown node %q (slot %d)",
		e.SourceNodeID, e.SourceClassType, e.InputName, e.ReferencedUUID, e.SlotIndex,
	)
}

// knownOutputTypes lists ComfyUI node class types that produce final output.
var knownOutputTypes = map[string]bool{
	"SaveImage":        true,
	"PreviewImage":     true,
	"SaveAnimatedWEBP": true,
	"SaveAnimatedPNG":  true,
	"SaveLatent":       true,
	"SaveAudio":        true,
	"SaveVideo":        true,
}

// isOutputNodeType returns true if classType is a known output node or contains
// "Save" or "Preview" in its name.
func isOutputNodeType(classType string) bool {
	if knownOutputTypes[classType] {
		return true
	}
	return strings.Contains(classType, "Save") || strings.Contains(classType, "Preview")
}

// ValidateNodeConnections checks that all inter-node connection references
// in the given nodes point to nodes that exist in the provided set.
// Returns a list of ValidationErrors for any broken references.
func ValidateNodeConnections(nodes []NodeState) []ValidationError {
	uuids := make(map[string]struct{}, len(nodes))
	for _, n := range nodes {
		uuids[n.ID] = struct{}{}
	}

	var errs []ValidationError
	for _, node := range nodes {
		for inputName, value := range node.Inputs {
			strVal, ok := value.(string)
			if !ok {
				continue
			}
			if !isConnectionRef(strVal) {
				continue
			}
			refUUID, slotIndex, err := parseConnectionRef(strVal)
			if err != nil {
				// Malformed ref — treat as broken.
				errs = append(errs, ValidationError{
					SourceNodeID:    node.ID,
					SourceClassType: node.ClassType,
					InputName:       inputName,
					ReferencedUUID:  strVal,
					SlotIndex:       -1,
				})
				continue
			}
			if _, exists := uuids[refUUID]; !exists {
				errs = append(errs, ValidationError{
					SourceNodeID:    node.ID,
					SourceClassType: node.ClassType,
					InputName:       inputName,
					ReferencedUUID:  refUUID,
					SlotIndex:       slotIndex,
				})
			}
		}
	}
	return errs
}

// ValidateHasOutputNode checks that at least one node in the workflow is a
// known output node type (SaveImage, PreviewImage, etc.).
func ValidateHasOutputNode(nodes []NodeState) bool {
	for _, n := range nodes {
		if isOutputNodeType(n.ClassType) {
			return true
		}
	}
	return false
}

// ValidateWorkflow runs all validations and returns combined diagnostics.
func ValidateWorkflow(nodes []NodeState) []ValidationError {
	errs := ValidateNodeConnections(nodes)

	if !ValidateHasOutputNode(nodes) {
		errs = append(errs, ValidationError{
			SourceNodeID:    "",
			SourceClassType: "",
			InputName:       "",
			ReferencedUUID:  "",
			SlotIndex:       -1,
		})
		// Override the last entry with a descriptive message via a dedicated
		// sentinel; callers can detect it by the empty SourceNodeID.
		errs[len(errs)-1] = ValidationError{
			SourceNodeID:    "workflow",
			SourceClassType: "workflow",
			InputName:       "output",
			ReferencedUUID:  "none",
			SlotIndex:       -1,
		}
	}

	return errs
}
