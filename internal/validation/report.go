package validation

import "encoding/json"

type Report struct {
	Valid              bool     `json:"valid"`
	Errors             []string `json:"errors"`
	Warnings           []string `json:"warnings"`
	ValidatedNodeCount int      `json:"validated_node_count"`
	ErrorCount         int      `json:"error_count"`
	WarningCount       int      `json:"warning_count"`
}

func NewReport(validatedNodeCount int) Report {
	return Report{
		Valid:              true,
		Errors:             []string{},
		Warnings:           []string{},
		ValidatedNodeCount: validatedNodeCount,
	}
}

func (r *Report) AddError(message string) {
	r.Errors = append(r.Errors, message)
	r.ErrorCount = len(r.Errors)
	r.Valid = false
}

func (r *Report) AddWarning(message string) {
	r.Warnings = append(r.Warnings, message)
	r.WarningCount = len(r.Warnings)
}

func (r Report) JSON() (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
