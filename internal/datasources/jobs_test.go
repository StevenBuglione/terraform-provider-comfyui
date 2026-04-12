package datasources

import (
	"context"
	"testing"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildJobsModel_EmptyList(t *testing.T) {
	resp := &client.JobsResponse{
		Jobs:    []client.Job{},
		HasMore: false,
	}

	model, err := buildJobsModel(resp, nil)
	if err != nil {
		t.Fatalf("buildJobsModel failed: %v", err)
	}

	if model.HasMore.ValueBool() != false {
		t.Errorf("expected HasMore false, got %v", model.HasMore.ValueBool())
	}

	if model.JobCount.ValueInt64() != 0 {
		t.Errorf("expected JobCount 0, got %d", model.JobCount.ValueInt64())
	}

	if len(model.Jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(model.Jobs))
	}
}

func TestBuildJobsModel_MultipleJobs(t *testing.T) {
	resp := &client.JobsResponse{
		Jobs: []client.Job{
			{ID: "job-1", Status: "completed", Priority: 0},
			{ID: "job-2", Status: "running", Priority: 1},
			{ID: "job-3", Status: "pending", Priority: 2},
		},
		HasMore: true,
	}

	model, err := buildJobsModel(resp, nil)
	if err != nil {
		t.Fatalf("buildJobsModel failed: %v", err)
	}

	if model.HasMore.ValueBool() != true {
		t.Errorf("expected HasMore true, got %v", model.HasMore.ValueBool())
	}

	if model.JobCount.ValueInt64() != 3 {
		t.Errorf("expected JobCount 3, got %d", model.JobCount.ValueInt64())
	}

	if len(model.Jobs) != 3 {
		t.Fatalf("expected 3 jobs, got %d", len(model.Jobs))
	}

	if model.Jobs[0].ID.ValueString() != "job-1" {
		t.Errorf("expected first job ID 'job-1', got %q", model.Jobs[0].ID.ValueString())
	}

	if model.Jobs[1].Status.ValueString() != "running" {
		t.Errorf("expected second job status 'running', got %q", model.Jobs[1].Status.ValueString())
	}

	if model.Jobs[2].Priority.ValueInt64() != 2 {
		t.Errorf("expected third job priority 2, got %d", model.Jobs[2].Priority.ValueInt64())
	}
}

func TestBuildJobsModel_WithRichFields(t *testing.T) {
	resp := &client.JobsResponse{
		Jobs: []client.Job{
			{
				ID:           "job-rich",
				Status:       "completed",
				Priority:     5,
				WorkflowID:   "wf-123",
				OutputsCount: intPtr(2),
				CreateTime:   int64Ptr(1700000000),
				PreviewOutput: map[string]interface{}{
					"node_id": "9",
				},
			},
		},
		HasMore: false,
	}

	model, err := buildJobsModel(resp, nil)
	if err != nil {
		t.Fatalf("buildJobsModel failed: %v", err)
	}

	if len(model.Jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(model.Jobs))
	}

	job := model.Jobs[0]

	if job.WorkflowID.ValueString() != "wf-123" {
		t.Errorf("expected WorkflowID 'wf-123', got %q", job.WorkflowID.ValueString())
	}

	if job.OutputsCount.ValueInt64() != 2 {
		t.Errorf("expected OutputsCount 2, got %d", job.OutputsCount.ValueInt64())
	}

	if job.CreateTime.ValueInt64() != 1700000000 {
		t.Errorf("expected CreateTime 1700000000, got %d", job.CreateTime.ValueInt64())
	}

	if job.PreviewOutputJSON.IsNull() {
		t.Error("expected PreviewOutputJSON to be set")
	}
}

func TestBuildJobsModel_PreservesNullNumericFieldsWhenOmitted(t *testing.T) {
	resp := &client.JobsResponse{
		Jobs: []client.Job{
			{
				ID:     "job-null-numerics",
				Status: "pending",
			},
		},
		HasMore: false,
	}

	model, err := buildJobsModel(resp, nil)
	if err != nil {
		t.Fatalf("buildJobsModel failed: %v", err)
	}

	if len(model.Jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(model.Jobs))
	}

	job := model.Jobs[0]
	if !job.CreateTime.IsNull() {
		t.Fatalf("expected create_time to be null when omitted, got %d", job.CreateTime.ValueInt64())
	}
	if !job.ExecutionStartTime.IsNull() {
		t.Fatalf("expected execution_start_time to be null when omitted, got %d", job.ExecutionStartTime.ValueInt64())
	}
	if !job.ExecutionEndTime.IsNull() {
		t.Fatalf("expected execution_end_time to be null when omitted, got %d", job.ExecutionEndTime.ValueInt64())
	}
	if !job.OutputsCount.IsNull() {
		t.Fatalf("expected outputs_count to be null when omitted, got %d", job.OutputsCount.ValueInt64())
	}
}

func TestBuildJobListFilter_EmptyFilter(t *testing.T) {
	model := &JobsModel{
		ID:         types.StringValue("jobs"),
		Statuses:   types.ListNull(types.StringType),
		WorkflowID: types.StringNull(),
		SortBy:     types.StringNull(),
		SortOrder:  types.StringNull(),
		Limit:      types.Int64Null(),
		Offset:     types.Int64Null(),
	}

	filter, diags := buildJobListFilter(model)
	if diags.HasError() {
		t.Fatalf("buildJobListFilter returned diagnostics: %v", diags)
	}

	if len(filter.Status) != 0 {
		t.Errorf("expected empty Status slice, got %v", filter.Status)
	}

	if filter.WorkflowID != "" {
		t.Errorf("expected empty WorkflowID, got %q", filter.WorkflowID)
	}

	if filter.SortBy != "" {
		t.Errorf("expected empty SortBy, got %q", filter.SortBy)
	}

	if filter.SortOrder != "" {
		t.Errorf("expected empty SortOrder, got %q", filter.SortOrder)
	}

	if filter.Limit != nil {
		t.Errorf("expected nil Limit, got %v", *filter.Limit)
	}

	if filter.Offset != nil {
		t.Errorf("expected nil Offset, got %v", *filter.Offset)
	}
}

func TestBuildJobListFilter_WithStatuses(t *testing.T) {
	statusList, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"completed", "running"})

	model := &JobsModel{
		ID:         types.StringValue("jobs"),
		Statuses:   statusList,
		WorkflowID: types.StringNull(),
		SortBy:     types.StringNull(),
		SortOrder:  types.StringNull(),
		Limit:      types.Int64Null(),
		Offset:     types.Int64Null(),
	}

	filter, diags := buildJobListFilter(model)
	if diags.HasError() {
		t.Fatalf("buildJobListFilter returned diagnostics: %v", diags)
	}

	if len(filter.Status) != 2 {
		t.Fatalf("expected 2 statuses, got %d", len(filter.Status))
	}

	if filter.Status[0] != "completed" {
		t.Errorf("expected first status 'completed', got %q", filter.Status[0])
	}

	if filter.Status[1] != "running" {
		t.Errorf("expected second status 'running', got %q", filter.Status[1])
	}
}

func TestBuildJobListFilter_WithWorkflowID(t *testing.T) {
	model := &JobsModel{
		ID:         types.StringValue("jobs"),
		Statuses:   types.ListNull(types.StringType),
		WorkflowID: types.StringValue("workflow-abc"),
		SortBy:     types.StringNull(),
		SortOrder:  types.StringNull(),
		Limit:      types.Int64Null(),
		Offset:     types.Int64Null(),
	}

	filter, diags := buildJobListFilter(model)
	if diags.HasError() {
		t.Fatalf("buildJobListFilter returned diagnostics: %v", diags)
	}

	if filter.WorkflowID != "workflow-abc" {
		t.Errorf("expected WorkflowID 'workflow-abc', got %q", filter.WorkflowID)
	}
}

func TestBuildJobListFilter_WithSorting(t *testing.T) {
	model := &JobsModel{
		ID:         types.StringValue("jobs"),
		Statuses:   types.ListNull(types.StringType),
		WorkflowID: types.StringNull(),
		SortBy:     types.StringValue("create_time"),
		SortOrder:  types.StringValue("desc"),
		Limit:      types.Int64Null(),
		Offset:     types.Int64Null(),
	}

	filter, diags := buildJobListFilter(model)
	if diags.HasError() {
		t.Fatalf("buildJobListFilter returned diagnostics: %v", diags)
	}

	if filter.SortBy != "create_time" {
		t.Errorf("expected SortBy 'create_time', got %q", filter.SortBy)
	}

	if filter.SortOrder != "desc" {
		t.Errorf("expected SortOrder 'desc', got %q", filter.SortOrder)
	}
}

func TestBuildJobListFilter_WithPagination(t *testing.T) {
	model := &JobsModel{
		ID:         types.StringValue("jobs"),
		Statuses:   types.ListNull(types.StringType),
		WorkflowID: types.StringNull(),
		SortBy:     types.StringNull(),
		SortOrder:  types.StringNull(),
		Limit:      types.Int64Value(50),
		Offset:     types.Int64Value(100),
	}

	filter, diags := buildJobListFilter(model)
	if diags.HasError() {
		t.Fatalf("buildJobListFilter returned diagnostics: %v", diags)
	}

	if filter.Limit == nil {
		t.Fatal("expected Limit to be set")
	}

	if *filter.Limit != 50 {
		t.Errorf("expected Limit 50, got %d", *filter.Limit)
	}

	if filter.Offset == nil {
		t.Fatal("expected Offset to be set")
	}

	if *filter.Offset != 100 {
		t.Errorf("expected Offset 100, got %d", *filter.Offset)
	}
}

func TestBuildJobListFilter_AllFilters(t *testing.T) {
	statusList, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"pending", "running"})

	model := &JobsModel{
		ID:         types.StringValue("jobs"),
		Statuses:   statusList,
		WorkflowID: types.StringValue("wf-xyz"),
		SortBy:     types.StringValue("priority"),
		SortOrder:  types.StringValue("asc"),
		Limit:      types.Int64Value(25),
		Offset:     types.Int64Value(0),
	}

	filter, diags := buildJobListFilter(model)
	if diags.HasError() {
		t.Fatalf("buildJobListFilter returned diagnostics: %v", diags)
	}

	if len(filter.Status) != 2 {
		t.Errorf("expected 2 statuses, got %d", len(filter.Status))
	}

	if filter.WorkflowID != "wf-xyz" {
		t.Errorf("expected WorkflowID 'wf-xyz', got %q", filter.WorkflowID)
	}

	if filter.SortBy != "priority" {
		t.Errorf("expected SortBy 'priority', got %q", filter.SortBy)
	}

	if filter.SortOrder != "asc" {
		t.Errorf("expected SortOrder 'asc', got %q", filter.SortOrder)
	}

	if filter.Limit == nil || *filter.Limit != 25 {
		t.Errorf("expected Limit 25, got %v", filter.Limit)
	}

	if filter.Offset == nil || *filter.Offset != 0 {
		t.Errorf("expected Offset 0, got %v", filter.Offset)
	}
}

func TestJobsDataSource_Factory(t *testing.T) {
	ds := NewJobsDataSource()
	if ds == nil {
		t.Fatal("factory returned nil data source")
	}

	jobsDS, ok := ds.(*JobsDataSource)
	if !ok {
		t.Fatal("factory returned wrong type")
	}

	if jobsDS.client != nil {
		t.Error("expected client to be nil before Configure")
	}
}

func TestBuildJobsModel_JSONMarshalFailureInNestedJob(t *testing.T) {
	// Create a response with one invalid job that cannot be marshaled
	resp := &client.JobsResponse{
		Jobs: []client.Job{
			{
				ID:       "job-good",
				Status:   "completed",
				Priority: 0,
			},
			{
				ID:       "job-bad",
				Status:   "error",
				Priority: 0,
				Outputs: map[string]interface{}{
					"broken": make(chan int), // channels cannot be marshaled
				},
			},
		},
		HasMore: false,
	}

	_, err := buildJobsModel(resp, nil)
	if err == nil {
		t.Fatal("expected error from buildJobsModel with un-marshalable nested job, got nil")
	}

	if err.Error() == "" {
		t.Error("expected non-empty error message")
	}
}

func TestBuildJobsModel_PreservesInputFilters(t *testing.T) {
	statusList, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"completed"})

	inputModel := &JobsModel{
		Statuses:   statusList,
		WorkflowID: types.StringValue("wf-preserve"),
		SortBy:     types.StringValue("create_time"),
		SortOrder:  types.StringValue("desc"),
		Limit:      types.Int64Value(10),
		Offset:     types.Int64Value(5),
	}

	resp := &client.JobsResponse{
		Jobs: []client.Job{
			{ID: "job-1", Status: "completed", Priority: 0},
		},
		HasMore: false,
	}

	model, err := buildJobsModel(resp, inputModel)
	if err != nil {
		t.Fatalf("buildJobsModel failed: %v", err)
	}

	// Verify all input fields are preserved
	if model.WorkflowID.ValueString() != "wf-preserve" {
		t.Errorf("expected preserved WorkflowID 'wf-preserve', got %q", model.WorkflowID.ValueString())
	}

	if model.SortBy.ValueString() != "create_time" {
		t.Errorf("expected preserved SortBy 'create_time', got %q", model.SortBy.ValueString())
	}

	if model.SortOrder.ValueString() != "desc" {
		t.Errorf("expected preserved SortOrder 'desc', got %q", model.SortOrder.ValueString())
	}

	if model.Limit.ValueInt64() != 10 {
		t.Errorf("expected preserved Limit 10, got %d", model.Limit.ValueInt64())
	}

	if model.Offset.ValueInt64() != 5 {
		t.Errorf("expected preserved Offset 5, got %d", model.Offset.ValueInt64())
	}

	var statuses []string
	model.Statuses.ElementsAs(context.Background(), &statuses, false)
	if len(statuses) != 1 || statuses[0] != "completed" {
		t.Errorf("expected preserved Statuses ['completed'], got %v", statuses)
	}
}

func TestBuildJobsModel_NestedJobsExposeJSONFields(t *testing.T) {
	resp := &client.JobsResponse{
		Jobs: []client.Job{
			{
				ID:       "job-1",
				Status:   "completed",
				Priority: 0,
				PreviewOutput: map[string]interface{}{
					"node_id": "9",
					"images":  []interface{}{"preview.png"},
				},
				Outputs: map[string]interface{}{
					"9": map[string]interface{}{
						"images": []interface{}{
							map[string]interface{}{
								"filename": "output.png",
							},
						},
					},
				},
				ExecutionStatus: map[string]interface{}{
					"status_str": "success",
				},
				Workflow: &client.JobWorkflow{
					Prompt: map[string]interface{}{
						"3": map[string]interface{}{
							"class_type": "KSampler",
						},
					},
					ExtraData: map[string]interface{}{
						"client_id": "test",
					},
				},
			},
		},
		HasMore: false,
	}

	model, err := buildJobsModel(resp, nil)
	if err != nil {
		t.Fatalf("buildJobsModel failed: %v", err)
	}

	if len(model.Jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(model.Jobs))
	}

	job := model.Jobs[0]

	if job.PreviewOutputJSON.IsNull() {
		t.Error("expected nested job PreviewOutputJSON to still be set")
	}

	if job.OutputsJSON.IsNull() {
		t.Error("expected nested job OutputsJSON to still be set")
	}

	if job.ExecutionStatusJSON.IsNull() {
		t.Error("expected nested job ExecutionStatusJSON to still be set")
	}

	if job.WorkflowJSON.IsNull() {
		t.Error("expected nested job WorkflowJSON to still be set")
	}
}

func TestJobsDataSource_SchemaAvoidsNestedDynamicAttributes(t *testing.T) {
	ds := NewJobsDataSource().(*JobsDataSource)
	var resp datasource.SchemaResponse
	ds.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	attr, ok := resp.Schema.Attributes["jobs"].(datasourceschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("expected jobs to be a list nested attribute, got %#v", resp.Schema.Attributes["jobs"])
	}
	for nestedName, nestedAttr := range attr.NestedObject.Attributes {
		if _, isDynamic := nestedAttr.(datasourceschema.DynamicAttribute); isDynamic {
			t.Fatalf("expected jobs.%s to avoid nested dynamic attributes", nestedName)
		}
	}
}

// Regression tests for Fix 2: Harden pagination conversion in buildJobListFilter

func TestBuildJobListFilter_RejectsNegativeLimit(t *testing.T) {
	model := &JobsModel{
		ID:         types.StringValue("jobs"),
		Statuses:   types.ListNull(types.StringType),
		WorkflowID: types.StringNull(),
		SortBy:     types.StringNull(),
		SortOrder:  types.StringNull(),
		Limit:      types.Int64Value(-10), // Invalid: negative
		Offset:     types.Int64Null(),
	}

	_, diags := buildJobListFilter(model)
	if !diags.HasError() {
		t.Fatal("expected error for negative limit, got none")
	}

	// Verify error message mentions "limit"
	errMsg := diags.Errors()[0].Summary()
	if errMsg == "" {
		t.Error("expected non-empty error summary")
	}
}

func TestBuildJobListFilter_RejectsNegativeOffset(t *testing.T) {
	model := &JobsModel{
		ID:         types.StringValue("jobs"),
		Statuses:   types.ListNull(types.StringType),
		WorkflowID: types.StringNull(),
		SortBy:     types.StringNull(),
		SortOrder:  types.StringNull(),
		Limit:      types.Int64Null(),
		Offset:     types.Int64Value(-5), // Invalid: negative
	}

	_, diags := buildJobListFilter(model)
	if !diags.HasError() {
		t.Fatal("expected error for negative offset, got none")
	}

	// Verify error message mentions "offset"
	errMsg := diags.Errors()[0].Summary()
	if errMsg == "" {
		t.Error("expected non-empty error summary")
	}
}

func TestBuildJobListFilter_RejectsLimitOverflow(t *testing.T) {
	// Use a value that exceeds int32 max to test overflow protection
	// Provider uses 1_000_000_000 as the safe bound
	model := &JobsModel{
		ID:         types.StringValue("jobs"),
		Statuses:   types.ListNull(types.StringType),
		WorkflowID: types.StringNull(),
		SortBy:     types.StringNull(),
		SortOrder:  types.StringNull(),
		Limit:      types.Int64Value(2_000_000_000), // Exceeds safe bound
		Offset:     types.Int64Null(),
	}

	_, diags := buildJobListFilter(model)
	if !diags.HasError() {
		t.Fatal("expected error for limit overflow, got none")
	}
}

func TestBuildJobListFilter_RejectsOffsetOverflow(t *testing.T) {
	// Use a value that exceeds the safe bound
	model := &JobsModel{
		ID:         types.StringValue("jobs"),
		Statuses:   types.ListNull(types.StringType),
		WorkflowID: types.StringNull(),
		SortBy:     types.StringNull(),
		SortOrder:  types.StringNull(),
		Limit:      types.Int64Null(),
		Offset:     types.Int64Value(2_000_000_000), // Exceeds safe bound
	}

	_, diags := buildJobListFilter(model)
	if !diags.HasError() {
		t.Fatal("expected error for offset overflow, got none")
	}
}

func TestBuildJobListFilter_AcceptsValidLargePagination(t *testing.T) {
	// Test a large but valid value within the safe bound
	model := &JobsModel{
		ID:         types.StringValue("jobs"),
		Statuses:   types.ListNull(types.StringType),
		WorkflowID: types.StringNull(),
		SortBy:     types.StringNull(),
		SortOrder:  types.StringNull(),
		Limit:      types.Int64Value(999_999_999), // Within safe bound
		Offset:     types.Int64Value(999_999_999), // Within safe bound
	}

	filter, diags := buildJobListFilter(model)
	if diags.HasError() {
		t.Fatalf("buildJobListFilter failed with valid values: %v", diags)
	}

	if filter.Limit == nil {
		t.Fatal("expected Limit to be set")
	}
	if *filter.Limit != 999_999_999 {
		t.Errorf("expected Limit 999999999, got %d", *filter.Limit)
	}

	if filter.Offset == nil {
		t.Fatal("expected Offset to be set")
	}
	if *filter.Offset != 999_999_999 {
		t.Errorf("expected Offset 999999999, got %d", *filter.Offset)
	}
}

func TestBuildJobListFilter_AcceptsZeroPagination(t *testing.T) {
	// Zero is valid for pagination
	model := &JobsModel{
		ID:         types.StringValue("jobs"),
		Statuses:   types.ListNull(types.StringType),
		WorkflowID: types.StringNull(),
		SortBy:     types.StringNull(),
		SortOrder:  types.StringNull(),
		Limit:      types.Int64Value(0),
		Offset:     types.Int64Value(0),
	}

	filter, diags := buildJobListFilter(model)
	if diags.HasError() {
		t.Fatalf("buildJobListFilter failed with zero values: %v", diags)
	}

	if filter.Limit == nil {
		t.Fatal("expected Limit to be set")
	}
	if *filter.Limit != 0 {
		t.Errorf("expected Limit 0, got %d", *filter.Limit)
	}

	if filter.Offset == nil {
		t.Fatal("expected Offset to be set")
	}
	if *filter.Offset != 0 {
		t.Errorf("expected Offset 0, got %d", *filter.Offset)
	}
}

func TestBuildJobsModel_NestedJobsWithNullJSONFields(t *testing.T) {
	resp := &client.JobsResponse{
		Jobs: []client.Job{
			{
				ID:              "job-pending",
				Status:          "pending",
				Priority:        0,
				PreviewOutput:   nil,
				Outputs:         nil,
				ExecutionStatus: nil,
				ExecutionError:  nil,
				Workflow:        nil,
			},
		},
		HasMore: false,
	}

	model, err := buildJobsModel(resp, nil)
	if err != nil {
		t.Fatalf("buildJobsModel failed: %v", err)
	}

	if len(model.Jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(model.Jobs))
	}

	job := model.Jobs[0]

	if !job.PreviewOutputJSON.IsNull() {
		t.Error("expected nested job PreviewOutputJSON to be null")
	}

	if !job.OutputsJSON.IsNull() {
		t.Error("expected nested job OutputsJSON to be null")
	}

	if !job.ExecutionStatusJSON.IsNull() {
		t.Error("expected nested job ExecutionStatusJSON to be null")
	}

	if !job.ExecutionErrorJSON.IsNull() {
		t.Error("expected nested job ExecutionErrorJSON to be null")
	}

	if !job.WorkflowJSON.IsNull() {
		t.Error("expected nested job WorkflowJSON to be null")
	}
}
