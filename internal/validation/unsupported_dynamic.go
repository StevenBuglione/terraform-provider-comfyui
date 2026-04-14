package validation

import "strings"

type UnsupportedDynamicValidationMode string

const (
	UnsupportedDynamicValidationModeError   UnsupportedDynamicValidationMode = "error"
	UnsupportedDynamicValidationModeWarning UnsupportedDynamicValidationMode = "warning"
	UnsupportedDynamicValidationModeIgnore  UnsupportedDynamicValidationMode = "ignore"
)

func ResolveUnsupportedDynamicValidationMode(mode string) UnsupportedDynamicValidationMode {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case string(UnsupportedDynamicValidationModeWarning):
		return UnsupportedDynamicValidationModeWarning
	case string(UnsupportedDynamicValidationModeIgnore):
		return UnsupportedDynamicValidationModeIgnore
	default:
		return UnsupportedDynamicValidationModeError
	}
}
