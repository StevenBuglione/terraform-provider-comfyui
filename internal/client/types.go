package client

import "encoding/json"

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
	Prompt  []interface{}         `json:"prompt"`
	Outputs map[string]NodeOutput `json:"outputs"`
	Status  ExecutionStatus       `json:"status"`
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
	StatusStr string `json:"status_str"`
	Completed bool   `json:"completed"`
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
