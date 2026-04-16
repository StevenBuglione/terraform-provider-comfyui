package resources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/inventory"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/nodeschema"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
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

type dcTestNodeModel struct {
	ID     types.String `tfsdk:"id"`
	NodeID types.String `tfsdk:"node_id"`
	Combo  types.Object `tfsdk:"combo"`
}

var dcSubcomboAttributeTypes = map[string]attr.Type{
	"selection": types.StringType,
	"float_x":   types.Float64Type,
	"float_y":   types.Float64Type,
	"mask1":     types.StringType,
}

var dcComboAttributeTypes = map[string]attr.Type{
	"selection": types.StringType,
	"image":     types.StringType,
	"integer":   types.Int64Type,
	"string":    types.StringType,
	"subcombo": types.ObjectType{
		AttrTypes: dcSubcomboAttributeTypes,
	},
}

func TestValidateDynamicInputs_RejectsInvalidDynamicComboSelection(t *testing.T) {
	model := dcTestNodeModel{
		Combo: dcComboObject(map[string]attr.Value{
			"selection": types.StringValue("invalid"),
		}),
	}

	diags := ValidateDynamicInputsForTest(context.Background(), &client.Client{}, "DCTestNode", model)
	if !diags.HasError() {
		t.Fatal("expected diagnostics for invalid DynamicCombo selection")
	}
	if !strings.Contains(diags[0].Summary(), "Invalid DynamicCombo Selection") {
		t.Fatalf("unexpected diagnostic summary: %s", diags[0].Summary())
	}
	assertDiagnosticPath(t, diags[0], path.Root("combo").AtName("selection"))
}

// TestValidateDynamicInputs_DoesNotEnforceChildrenOfNonStrictNestedDynamicCombo checks that a
// nested DynamicCombo with SupportsStrictPlanValidation == false also does not enforce its required
// children at plan time.  DCTestNode.combo.option4.subcombo has SupportsStrictPlanValidation:false,
// so float_y must not be required even though the user explicitly set float_x.
func TestValidateDynamicInputs_DoesNotEnforceChildrenOfNonStrictNestedDynamicCombo(t *testing.T) {
	model := dcTestNodeModel{
		Combo: dcComboObject(map[string]attr.Value{
			"selection": types.StringValue("option4"),
			"subcombo": dcSubcomboObject(map[string]attr.Value{
				"selection": types.StringValue("opt1"),
				"float_x":   types.Float64Value(1.5),
				// float_y intentionally omitted: subcombo has SupportsStrictPlanValidation: false
			}),
		}),
	}

	diags := ValidateDynamicInputsForTest(context.Background(), &client.Client{}, "DCTestNode", model)
	if diags.HasError() {
		t.Fatalf("expected no diagnostics for nested non-strict DynamicCombo with missing child, got: %v", diags)
	}
}

func TestValidateDynamicInputs_RejectsInvalidNestedDynamicComboSelection(t *testing.T) {
	model := dcTestNodeModel{
		Combo: dcComboObject(map[string]attr.Value{
			"selection": types.StringValue("option4"),
			"subcombo": dcSubcomboObject(map[string]attr.Value{
				"selection": types.StringValue("invalid"),
			}),
		}),
	}

	diags := ValidateDynamicInputsForTest(context.Background(), &client.Client{}, "DCTestNode", model)
	if !diags.HasError() {
		t.Fatal("expected diagnostics for invalid nested DynamicCombo selection")
	}
	if !strings.Contains(diags[0].Summary(), "Invalid DynamicCombo Selection") {
		t.Fatalf("unexpected diagnostic summary: %s", diags[0].Summary())
	}
	assertDiagnosticPath(t, diags[0], path.Root("combo").AtName("subcombo").AtName("selection"))
}

func TestValidateDynamicInputs_AcceptsValidNestedDynamicComboSelection(t *testing.T) {
	model := dcTestNodeModel{
		Combo: dcComboObject(map[string]attr.Value{
			"selection": types.StringValue("option4"),
			"subcombo": dcSubcomboObject(map[string]attr.Value{
				"selection": types.StringValue("opt1"),
				"float_x":   types.Float64Value(1.5),
				"float_y":   types.Float64Value(2.5),
			}),
		}),
	}

	diags := ValidateDynamicInputsForTest(context.Background(), &client.Client{}, "DCTestNode", model)
	if diags.HasError() || diags.WarningsCount() != 0 {
		t.Fatalf("expected no diagnostics, got %v", diags)
	}
}

func TestValidateDynamicInputs_RejectsFieldsFromOtherDynamicComboOptions(t *testing.T) {
	model := dcTestNodeModel{
		Combo: dcComboObject(map[string]attr.Value{
			"selection": types.StringValue("option1"),
			"string":    types.StringValue("hello"),
			"integer":   types.Int64Value(7),
		}),
	}

	diags := ValidateDynamicInputsForTest(context.Background(), &client.Client{}, "DCTestNode", model)
	if !diags.HasError() {
		t.Fatal("expected diagnostics for invalid DynamicCombo field")
	}
	if !strings.Contains(diags[0].Summary(), "Invalid DynamicCombo Field") {
		t.Fatalf("unexpected diagnostic summary: %s", diags[0].Summary())
	}
	assertDiagnosticPath(t, diags[0], path.Root("combo").AtName("integer"))
}

// TestValidateGeneratedInput_DoesNotEnforceChildrenOfNonStrictDynamicComboInList checks the
// list variant: a DynamicCombo with SupportsStrictPlanValidation==false inside a list also does
// not enforce required children at plan time.
func TestValidateGeneratedInput_DoesNotEnforceChildrenOfNonStrictDynamicComboInList(t *testing.T) {
	schema, ok := nodeschema.LookupGeneratedNodeSchema("DCTestNode")
	if !ok {
		t.Fatal("expected generated schema for DCTestNode")
	}
	input, ok := generatedInputByName(schema.RequiredInputs, "combo")
	if !ok {
		t.Fatal("expected generated combo input for DCTestNode")
	}

	combos := types.ListValueMust(types.ObjectType{AttrTypes: dcComboAttributeTypes}, []attr.Value{
		dcComboObject(map[string]attr.Value{
			"selection": types.StringValue("option4"),
			"subcombo": dcSubcomboObject(map[string]attr.Value{
				"selection": types.StringValue("opt1"),
				"float_x":   types.Float64Value(1.5),
				// float_y omitted: subcombo has SupportsStrictPlanValidation: false
			}),
		}),
	})

	var diags diag.Diagnostics
	validateGeneratedInput(context.Background(), inventory.NewService(&client.Client{}), "error", input, combos, path.Root("combo"), &diags)

	if diags.HasError() {
		t.Fatalf("expected no diagnostics for non-strict nested DynamicCombo in list, got: %v", diags)
	}
}

func TestValidateDynamicInputs_IgnoresUnknownDynamicComboSelectionAtPlanTime(t *testing.T) {
	model := dcTestNodeModel{
		Combo: dcComboObject(map[string]attr.Value{
			"selection": types.StringUnknown(),
		}),
	}

	diags := ValidateDynamicInputsForTest(context.Background(), &client.Client{}, "DCTestNode", model)
	if diags.HasError() || diags.WarningsCount() != 0 {
		t.Fatalf("expected no diagnostics, got %v", diags)
	}
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

// TestValidateDynamicInputs_DoesNotEnforceChildrenWhenParentStrictPlanValidationDisabled
// documents the regression: when a DynamicCombo parent input has
// SupportsStrictPlanValidation == false, required child fields must NOT be enforced
// at plan time, because their values are only resolved at runtime.
//
// DCTestNode.combo already has SupportsStrictPlanValidation: false in the generated schema,
// so this test exercises the same code path as the WAN2 nodes.
//
// This test FAILS before the fix: the validator currently checks required children
// regardless of the parent's SupportsStrictPlanValidation flag.
func TestValidateDynamicInputs_DoesNotEnforceChildrenWhenParentStrictPlanValidationDisabled(t *testing.T) {
	// option1 requires the "string" child field, but since the parent combo has
	// SupportsStrictPlanValidation == false, omitting it should produce no errors.
	model := dcTestNodeModel{
		Combo: dcComboObject(map[string]attr.Value{
			"selection": types.StringValue("option1"),
			// "string" is intentionally omitted (null)
		}),
	}

	diags := ValidateDynamicInputsForTest(context.Background(), &client.Client{}, "DCTestNode", model)
	if diags.HasError() {
		t.Fatalf("expected no diagnostics when parent SupportsStrictPlanValidation is false, got: %v", diags)
	}
}

func dcComboObject(overrides map[string]attr.Value) types.Object {
	attributes := map[string]attr.Value{
		"selection": types.StringNull(),
		"image":     types.StringNull(),
		"integer":   types.Int64Null(),
		"string":    types.StringNull(),
		"subcombo":  types.ObjectNull(dcSubcomboAttributeTypes),
	}
	for key, value := range overrides {
		attributes[key] = value
	}
	return types.ObjectValueMust(dcComboAttributeTypes, attributes)
}

func dcSubcomboObject(overrides map[string]attr.Value) types.Object {
	attributes := map[string]attr.Value{
		"selection": types.StringNull(),
		"float_x":   types.Float64Null(),
		"float_y":   types.Float64Null(),
		"mask1":     types.StringNull(),
	}
	for key, value := range overrides {
		attributes[key] = value
	}
	return types.ObjectValueMust(dcSubcomboAttributeTypes, attributes)
}

func assertDiagnosticPath(t *testing.T, diagnostic diag.Diagnostic, want path.Path) {
	t.Helper()

	withPath, ok := diagnostic.(diag.DiagnosticWithPath)
	if !ok {
		t.Fatalf("expected diagnostic with path, got %T", diagnostic)
	}
	if !withPath.Path().Equal(want) {
		t.Fatalf("unexpected diagnostic path: got %s want %s", withPath.Path(), want)
	}
}

func generatedInputByName(inputs []nodeschema.GeneratedNodeSchemaInput, name string) (nodeschema.GeneratedNodeSchemaInput, bool) {
	for _, input := range inputs {
		if input.Name == name {
			return input, true
		}
	}
	return nodeschema.GeneratedNodeSchemaInput{}, false
}
