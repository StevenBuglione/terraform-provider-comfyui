package datasources

import (
	"context"
	"testing"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestQueueDataSource_ReadWithoutClientReturnsDiagnostic(t *testing.T) {
	d := &QueueDataSource{}
	var resp datasource.ReadResponse

	d.Read(context.Background(), datasource.ReadRequest{}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected missing client to add diagnostics")
	}
}

func TestBuildQueueModel_EnrichesQueueItems(t *testing.T) {
	queue := &client.QueueStatus{
		QueueRunning: [][]interface{}{
			{
				float64(9),
				"run-123",
				map[string]interface{}{
					"1": map[string]interface{}{"class_type": "KSampler"},
				},
				map[string]interface{}{
					"create_time": float64(1712345678),
					"extra_pnginfo": map[string]interface{}{
						"workflow": map[string]interface{}{"id": "wf-running"},
					},
				},
				[]interface{}{"5"},
			},
		},
		QueuePending: [][]interface{}{
			{
				float64(11),
				"pending-456",
				map[string]interface{}{
					"2": map[string]interface{}{"class_type": "SaveImage"},
				},
				map[string]interface{}{},
				[]interface{}{"9"},
			},
		},
	}

	model, err := buildQueueModel(queue)
	if err != nil {
		t.Fatalf("buildQueueModel failed: %v", err)
	}

	if model.RunningCount.ValueInt64() != 1 {
		t.Fatalf("expected 1 running item, got %d", model.RunningCount.ValueInt64())
	}
	if model.PendingCount.ValueInt64() != 1 {
		t.Fatalf("expected 1 pending item, got %d", model.PendingCount.ValueInt64())
	}
	if len(model.RunningItems) != 1 {
		t.Fatalf("expected 1 structured running item, got %d", len(model.RunningItems))
	}
	if len(model.PendingItems) != 1 {
		t.Fatalf("expected 1 structured pending item, got %d", len(model.PendingItems))
	}

	running := model.RunningItems[0]
	if running.PromptID.ValueString() != "run-123" {
		t.Fatalf("expected running prompt_id run-123, got %q", running.PromptID.ValueString())
	}
	if running.Priority.ValueInt64() != 9 {
		t.Fatalf("expected running priority 9, got %d", running.Priority.ValueInt64())
	}
	if running.WorkflowID.ValueString() != "wf-running" {
		t.Fatalf("expected workflow_id wf-running, got %q", running.WorkflowID.ValueString())
	}
	if running.CreateTime.ValueInt64() != 1712345678 {
		t.Fatalf("expected create_time 1712345678, got %d", running.CreateTime.ValueInt64())
	}
	if running.PromptJSON.IsNull() || running.PromptJSON.ValueString() == "" {
		t.Fatal("expected prompt_json to be populated")
	}
	if running.ExtraDataJSON.IsNull() || running.ExtraDataJSON.ValueString() == "" {
		t.Fatal("expected extra_data_json to be populated")
	}
	if running.OutputsToExecuteJSON.IsNull() || running.OutputsToExecuteJSON.ValueString() == "" {
		t.Fatal("expected outputs_to_execute_json to be populated")
	}
}

func TestBuildQueueModel_AllowsShortQueueItems(t *testing.T) {
	queue := &client.QueueStatus{
		QueueRunning: [][]interface{}{
			{
				float64(3),
				"legacy-run",
				map[string]interface{}{
					"1": map[string]interface{}{"class_type": "SaveImage"},
				},
				map[string]interface{}{
					"create_time": float64(100),
				},
			},
		},
	}

	model, err := buildQueueModel(queue)
	if err != nil {
		t.Fatalf("expected short queue items to remain supported, got error: %v", err)
	}

	if len(model.RunningItems) != 1 {
		t.Fatalf("expected 1 running item, got %d", len(model.RunningItems))
	}
	if !model.RunningItems[0].OutputsToExecuteJSON.IsNull() {
		t.Fatalf("expected outputs_to_execute_json to remain null for short tuples, got %q", model.RunningItems[0].OutputsToExecuteJSON.ValueString())
	}
}

func TestQueueDataSource_SchemaAvoidsNestedDynamicAttributes(t *testing.T) {
	ds := NewQueueDataSource().(*QueueDataSource)
	var resp datasource.SchemaResponse
	ds.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	for _, name := range []string{"running_items", "pending_items"} {
		attr, ok := resp.Schema.Attributes[name].(datasourceschema.ListNestedAttribute)
		if !ok {
			t.Fatalf("expected %s to be a list nested attribute, got %#v", name, resp.Schema.Attributes[name])
		}
		for nestedName, nestedAttr := range attr.NestedObject.Attributes {
			if _, isDynamic := nestedAttr.(datasourceschema.DynamicAttribute); isDynamic {
				t.Fatalf("expected %s.%s to avoid nested dynamic attributes", name, nestedName)
			}
		}
	}
}
