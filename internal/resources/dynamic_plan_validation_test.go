package resources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type checkpointLoaderSimpleTestModel struct {
	ID       types.String `tfsdk:"id"`
	NodeID   types.String `tfsdk:"node_id"`
	CkptName types.String `tfsdk:"ckpt_name"`
}

type basicSchedulerTestModel struct {
	ID        types.String `tfsdk:"id"`
	NodeID    types.String `tfsdk:"node_id"`
	Scheduler types.String `tfsdk:"scheduler"`
}

type loadImageTestModel struct {
	ID     types.String `tfsdk:"id"`
	NodeID types.String `tfsdk:"node_id"`
	Image  types.String `tfsdk:"image"`
}

func TestValidateDynamicInputs_AcceptsPresentInventoryValue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/object_info" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"CheckpointLoaderSimple":{"input":{"required":{"ckpt_name":[["realistic.safetensors"],{}]}}}}`))
	}))
	defer server.Close()

	c := &client.Client{HTTPClient: server.Client(), BaseURL: server.URL}
	model := checkpointLoaderSimpleTestModel{CkptName: types.StringValue("realistic.safetensors")}
	var diags = ValidateDynamicInputsForTest(context.Background(), c, "CheckpointLoaderSimple", model)
	if diags.HasError() {
		t.Fatalf("expected no diagnostics, got %v", diags)
	}
}

func TestValidateDynamicInputs_RejectsMissingInventoryValue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"CheckpointLoaderSimple":{"input":{"required":{"ckpt_name":[["realistic.safetensors"],{}]}}}}`))
	}))
	defer server.Close()

	c := &client.Client{HTTPClient: server.Client(), BaseURL: server.URL}
	model := checkpointLoaderSimpleTestModel{CkptName: types.StringValue("missing.safetensors")}
	var diags = ValidateDynamicInputsForTest(context.Background(), c, "CheckpointLoaderSimple", model)
	if !diags.HasError() {
		t.Fatal("expected diagnostics for missing inventory value")
	}
	if !strings.Contains(diags[0].Detail(), "missing.safetensors") {
		t.Fatalf("unexpected diagnostic detail: %s", diags[0].Detail())
	}
}

func TestValidateDynamicInputs_RejectsUnknownInventoryValueAtPlanTime(t *testing.T) {
	model := checkpointLoaderSimpleTestModel{CkptName: types.StringUnknown()}
	var diags = ValidateDynamicInputsForTest(context.Background(), &client.Client{}, "CheckpointLoaderSimple", model)
	if !diags.HasError() {
		t.Fatal("expected diagnostics for unknown plan-time value")
	}
	if !strings.Contains(diags[0].Summary(), "Unknown Dynamic Inventory Value") {
		t.Fatalf("unexpected diagnostic summary: %s", diags[0].Summary())
	}
}

func TestValidateDynamicInputs_RejectsUnsupportedDynamicExpression(t *testing.T) {
	model := basicSchedulerTestModel{Scheduler: types.StringValue("karras")}
	var diags = ValidateDynamicInputsForTest(context.Background(), &client.Client{}, "BasicScheduler", model)
	if !diags.HasError() {
		t.Fatal("expected diagnostics for unsupported dynamic expression")
	}
	if !strings.Contains(diags[0].Summary(), "Unsupported Dynamic Plan Validation") {
		t.Fatalf("unexpected diagnostic summary: %s", diags[0].Summary())
	}
}

func TestValidateDynamicInputs_WarnsForUnsupportedDynamicExpressionWhenConfigured(t *testing.T) {
	model := basicSchedulerTestModel{Scheduler: types.StringValue("karras")}
	var diags = ValidateDynamicInputsForTest(context.Background(), &client.Client{
		UnsupportedDynamicValidationMode: "warning",
	}, "BasicScheduler", model)
	if diags.HasError() {
		t.Fatalf("expected warning-only diagnostics, got %v", diags)
	}
	if diags.WarningsCount() != 1 {
		t.Fatalf("expected one warning, got %d", diags.WarningsCount())
	}
}

func TestValidateDynamicInputs_IgnoresUnsupportedDynamicExpressionWhenConfigured(t *testing.T) {
	model := basicSchedulerTestModel{Scheduler: types.StringValue("karras")}
	var diags = ValidateDynamicInputsForTest(context.Background(), &client.Client{
		UnsupportedDynamicValidationMode: "ignore",
	}, "BasicScheduler", model)
	if diags.HasError() || diags.WarningsCount() != 0 {
		t.Fatalf("expected no diagnostics, got %v", diags)
	}
}

func TestValidateDynamicInputs_LoadImageUsesUnsupportedDynamicValidationPolicy(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		mode         string
		wantError    bool
		wantWarnings int
		wantSummary  string
	}{
		{
			name:        "default error mode",
			wantError:   true,
			wantSummary: "Unsupported Dynamic Plan Validation",
		},
		{
			name:         "warning mode",
			mode:         "warning",
			wantWarnings: 1,
			wantSummary:  "Unsupported Dynamic Plan Validation",
		},
		{
			name: "ignore mode",
			mode: "ignore",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			model := loadImageTestModel{Image: types.StringValue("reference.png")}
			diags := ValidateDynamicInputsForTest(context.Background(), &client.Client{
				UnsupportedDynamicValidationMode: tc.mode,
			}, "LoadImage", model)

			if diags.HasError() != tc.wantError {
				t.Fatalf("HasError() = %v, want %v; diagnostics: %v", diags.HasError(), tc.wantError, diags)
			}
			if diags.WarningsCount() != tc.wantWarnings {
				t.Fatalf("WarningsCount() = %d, want %d; diagnostics: %v", diags.WarningsCount(), tc.wantWarnings, diags)
			}
			if tc.wantSummary != "" {
				if len(diags) == 0 {
					t.Fatalf("expected diagnostics containing %q, got none", tc.wantSummary)
				}
				if !strings.Contains(diags[0].Summary(), tc.wantSummary) {
					t.Fatalf("unexpected diagnostic summary: %s", diags[0].Summary())
				}
			}
		})
	}
}
