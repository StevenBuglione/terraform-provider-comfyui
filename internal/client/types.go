package client

import (
	"bytes"
	"encoding/json"
)

// QueuePromptRequest is submitted to POST /prompt.
type QueuePromptRequest struct {
	Prompt                  map[string]interface{} `json:"prompt"`
	PromptID                string                 `json:"prompt_id,omitempty"`
	ClientID                string                 `json:"client_id,omitempty"`
	ExtraData               map[string]interface{} `json:"extra_data,omitempty"`
	PartialExecutionTargets []string               `json:"partial_execution_targets,omitempty"`
}

// QueueResponse is returned by POST /prompt
type QueueResponse struct {
	PromptID   string                 `json:"prompt_id"`
	Number     int                    `json:"number"`
	NodeErrors map[string]interface{} `json:"node_errors"`
}

// HistoryResponse is returned by GET /history/{id}
type HistoryResponse map[string]HistoryEntry

// HistoryEntry represents a single execution history entry
type HistoryEntry struct {
	Prompt  []interface{}          `json:"prompt"`
	Outputs map[string]interface{} `json:"outputs"`
	Status  ExecutionStatus        `json:"status"`
}

// NodeOutput represents the output of a single node execution
type NodeOutput struct {
	Images []ImageOutput `json:"images,omitempty"`
	Audio  []AudioOutput `json:"audio,omitempty"`
}

// ImageOutput represents a generated image
type ImageOutput struct {
	Filename  string `json:"filename"`
	Subfolder string `json:"subfolder"`
	Type      string `json:"type"`
}

// AudioOutput represents generated audio
type AudioOutput struct {
	Filename  string `json:"filename"`
	Subfolder string `json:"subfolder"`
	Type      string `json:"type"`
}

// ExecutionStatus represents execution state
type ExecutionStatus struct {
	StatusStr string          `json:"status_str"`
	Completed bool            `json:"completed"`
	Messages  [][]interface{} `json:"messages,omitempty"`
}

// SystemStats is returned by GET /system_stats
type SystemStats struct {
	System  SystemInfo `json:"system"`
	Devices []Device   `json:"devices"`
}

// SystemInfo contains system-level information
type SystemInfo struct {
	OS             string `json:"os"`
	PythonVersion  string `json:"python_version"`
	ComfyUIVersion string `json:"comfyui_version"`
	EmbeddedPython bool   `json:"embedded_python"`
}

// Device represents a compute device
type Device struct {
	Name           string `json:"name"`
	Type           string `json:"type"`
	Index          int    `json:"index"`
	VRAMTotal      int64  `json:"vram_total"`
	VRAMFree       int64  `json:"vram_free"`
	TorchVRAMTotal int64  `json:"torch_vram_total"`
	TorchVRAMFree  int64  `json:"torch_vram_free"`
}

// QueueStatus is returned by GET /queue
type QueueStatus struct {
	QueueRunning [][]interface{} `json:"queue_running"`
	QueuePending [][]interface{} `json:"queue_pending"`
}

// NodeInfo represents node type information from GET /object_info
type NodeInfo struct {
	Input        NodeInputInfo       `json:"input"`
	InputOrder   map[string][]string `json:"input_order"`
	Output       []string            `json:"output"`
	OutputIsList []bool              `json:"output_is_list"`
	OutputName   []string            `json:"output_name"`
	Name         string              `json:"name"`
	DisplayName  string              `json:"display_name"`
	Description  string              `json:"description"`
	Category     string              `json:"category"`
	OutputNode   bool                `json:"output_node"`
	Deprecated   bool                `json:"deprecated"`
	Experimental bool                `json:"experimental"`
}

// NodeInputInfo contains input specifications
type NodeInputInfo struct {
	Required map[string]interface{} `json:"required"`
	Optional map[string]interface{} `json:"optional"`
	Hidden   map[string]interface{} `json:"hidden"`
}

type UploadResponse struct {
	Name      string `json:"name"`
	Subfolder string `json:"subfolder"`
	Type      string `json:"type"`
}

type RemoteFileReference struct {
	Filename  string `json:"filename"`
	Subfolder string `json:"subfolder,omitempty"`
	Type      string `json:"type,omitempty"`
}

func (r RemoteFileReference) JSON() (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type DownloadViewResponse struct {
	Content     []byte
	ContentType string
}

type GlobalSubgraphInfo struct {
	NodePack string `json:"node_pack"`
	rawJSON  json.RawMessage
}

func (i *GlobalSubgraphInfo) UnmarshalJSON(data []byte) error {
	i.rawJSON = append(i.rawJSON[:0], data...)

	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		i.NodePack = ""
		return nil
	}

	type alias struct {
		NodePack string `json:"node_pack"`
	}
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	i.NodePack = decoded.NodePack
	return nil
}

func (i GlobalSubgraphInfo) MarshalJSON() ([]byte, error) {
	if len(i.rawJSON) > 0 {
		return append([]byte(nil), i.rawJSON...), nil
	}

	type alias struct {
		NodePack string `json:"node_pack"`
	}
	return json.Marshal(alias{NodePack: i.NodePack})
}

type GlobalSubgraphCatalogEntry struct {
	Source string             `json:"source"`
	Name   string             `json:"name"`
	Info   GlobalSubgraphInfo `json:"info"`
}

type GlobalSubgraphDefinition struct {
	Source string             `json:"source"`
	Name   string             `json:"name"`
	Info   GlobalSubgraphInfo `json:"info"`
	Data   string             `json:"data"`
}

// Job represents a unified job structure from /api/jobs endpoints
type Job struct {
	ID                 string                 `json:"id"`
	Status             string                 `json:"status"`
	Priority           int                    `json:"priority"`
	CreateTime         *int64                 `json:"create_time,omitempty"`
	ExecutionStartTime *int64                 `json:"execution_start_time,omitempty"`
	ExecutionEndTime   *int64                 `json:"execution_end_time,omitempty"`
	ExecutionError     map[string]interface{} `json:"execution_error,omitempty"`
	OutputsCount       *int                   `json:"outputs_count,omitempty"`
	PreviewOutput      map[string]interface{} `json:"preview_output,omitempty"`
	WorkflowID         string                 `json:"workflow_id,omitempty"`
	Outputs            map[string]interface{} `json:"outputs,omitempty"`
	ExecutionStatus    map[string]interface{} `json:"execution_status,omitempty"`
	Workflow           *JobWorkflow           `json:"workflow,omitempty"`
}

// JobWorkflow contains workflow details in a job
type JobWorkflow struct {
	Prompt    map[string]interface{} `json:"prompt"`
	ExtraData map[string]interface{} `json:"extra_data"`
}

// JobsResponse is returned by GET /api/jobs
type JobsResponse struct {
	Jobs    []Job `json:"jobs"`
	HasMore bool  `json:"has_more"`
}

// JobListFilter contains query parameters for GET /api/jobs
type JobListFilter struct {
	Status     []string
	WorkflowID string
	SortBy     string
	SortOrder  string
	Limit      *int
	Offset     *int
}
