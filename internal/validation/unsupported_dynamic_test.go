package validation

import "testing"

func TestResolveUnsupportedDynamicValidationMode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		in   string
		want UnsupportedDynamicValidationMode
	}{
		{name: "default", in: "", want: UnsupportedDynamicValidationModeError},
		{name: "warning trimmed", in: " warning ", want: UnsupportedDynamicValidationModeWarning},
		{name: "ignore uppercase", in: "IGNORE", want: UnsupportedDynamicValidationModeIgnore},
		{name: "invalid defaults to error", in: "explode", want: UnsupportedDynamicValidationModeError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := ResolveUnsupportedDynamicValidationMode(tc.in); got != tc.want {
				t.Fatalf("ResolveUnsupportedDynamicValidationMode(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
