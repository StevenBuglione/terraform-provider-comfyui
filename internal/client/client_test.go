package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(server *httptest.Server) *Client {
	return &Client{
		HTTPClient: server.Client(),
		BaseURL:    server.URL,
	}
}

func mustEncodeJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("failed to encode JSON response: %v", err)
	}
}

func mustWriteBody(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()
	if _, err := w.Write([]byte(body)); err != nil {
		t.Fatalf("failed to write response body: %v", err)
	}
}

func TestNewClient(t *testing.T) {
	c := NewClient("localhost", 8188, "my-key")
	if c.Host != "localhost" {
		t.Errorf("expected Host=localhost, got %s", c.Host)
	}
	if c.Port != 8188 {
		t.Errorf("expected Port=8188, got %d", c.Port)
	}
	if c.APIKey != "my-key" {
		t.Errorf("expected APIKey=my-key, got %s", c.APIKey)
	}
	if c.BaseURL != "http://localhost:8188" {
		t.Errorf("expected BaseURL=http://localhost:8188, got %s", c.BaseURL)
	}
	if c.HTTPClient == nil {
		t.Fatal("expected HTTPClient to be non-nil")
	}
}

func TestGetSystemStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/system_stats" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		mustEncodeJSON(t, w, SystemStats{
			System: SystemInfo{
				OS:             "linux",
				PythonVersion:  "3.11.0",
				ComfyUIVersion: "0.18.5",
			},
			Devices: []Device{
				{Name: "NVIDIA GeForce RTX 4090", Type: "cuda", VRAMTotal: 24576000000},
			},
		})
	}))
	defer server.Close()

	c := newTestClient(server)
	stats, err := c.GetSystemStats()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.System.OS != "linux" {
		t.Errorf("expected OS=linux, got %s", stats.System.OS)
	}
	if stats.System.PythonVersion != "3.11.0" {
		t.Errorf("expected PythonVersion=3.11.0, got %s", stats.System.PythonVersion)
	}
	if stats.System.ComfyUIVersion != "0.18.5" {
		t.Errorf("expected ComfyUIVersion=0.18.5, got %s", stats.System.ComfyUIVersion)
	}
	if len(stats.Devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(stats.Devices))
	}
	if stats.Devices[0].Name != "NVIDIA GeForce RTX 4090" {
		t.Errorf("unexpected device name: %s", stats.Devices[0].Name)
	}
	if stats.Devices[0].Type != "cuda" {
		t.Errorf("unexpected device type: %s", stats.Devices[0].Type)
	}
}

func TestGetQueue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/queue" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		mustEncodeJSON(t, w, QueueStatus{
			QueueRunning: [][]interface{}{},
			QueuePending: [][]interface{}{},
		})
	}))
	defer server.Close()

	c := newTestClient(server)
	queue, err := c.GetQueue()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(queue.QueueRunning) != 0 {
		t.Errorf("expected empty running queue, got %d", len(queue.QueueRunning))
	}
	if len(queue.QueuePending) != 0 {
		t.Errorf("expected empty pending queue, got %d", len(queue.QueuePending))
	}
}

func TestQueuePrompt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/prompt" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		ct := r.Header.Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("expected Content-Type=application/json, got %s", ct)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if _, ok := body["prompt"]; !ok {
			t.Error("request body missing 'prompt' key")
		}

		mustEncodeJSON(t, w, QueueResponse{
			PromptID:   "test-123",
			Number:     1,
			NodeErrors: map[string]interface{}{},
		})
	}))
	defer server.Close()

	c := newTestClient(server)
	resp, err := c.QueuePrompt(QueuePromptRequest{
		Prompt: map[string]interface{}{
			"1": map[string]interface{}{
				"class_type": "KSampler",
				"inputs":     map[string]interface{}{},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.PromptID != "test-123" {
		t.Errorf("expected prompt_id=test-123, got %s", resp.PromptID)
	}
	if resp.Number != 1 {
		t.Errorf("expected number=1, got %d", resp.Number)
	}
}

func TestQueuePrompt_IncludesWrapperFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if body["prompt_id"] != "prompt-123" {
			t.Fatalf("expected prompt_id to be sent, got %#v", body["prompt_id"])
		}
		if body["client_id"] != "client-456" {
			t.Fatalf("expected client_id to be sent, got %#v", body["client_id"])
		}

		extraData, ok := body["extra_data"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected extra_data object, got %#v", body["extra_data"])
		}
		if extraData["tenant"] != "dev" {
			t.Fatalf("expected extra_data.tenant=dev, got %#v", extraData["tenant"])
		}

		targets, ok := body["partial_execution_targets"].([]interface{})
		if !ok || len(targets) != 2 {
			t.Fatalf("expected partial_execution_targets to be sent, got %#v", body["partial_execution_targets"])
		}

		mustEncodeJSON(t, w, QueueResponse{
			PromptID:   "prompt-123",
			Number:     9,
			NodeErrors: map[string]interface{}{},
		})
	}))
	defer server.Close()

	c := newTestClient(server)
	resp, err := c.QueuePrompt(QueuePromptRequest{
		Prompt: map[string]interface{}{
			"1": map[string]interface{}{
				"class_type": "KSampler",
				"inputs":     map[string]interface{}{},
			},
		},
		PromptID: "prompt-123",
		ClientID: "client-456",
		ExtraData: map[string]interface{}{
			"tenant": "dev",
		},
		PartialExecutionTargets: []string{"3", "5"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Number != 9 {
		t.Fatalf("expected number 9, got %d", resp.Number)
	}
}

func TestQueuePromptNodeErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mustEncodeJSON(t, w, QueueResponse{
			PromptID: "err-456",
			Number:   2,
			NodeErrors: map[string]interface{}{
				"3": map[string]interface{}{
					"error": "missing input",
				},
			},
		})
	}))
	defer server.Close()

	c := newTestClient(server)
	resp, err := c.QueuePrompt(QueuePromptRequest{Prompt: map[string]interface{}{}})
	if err == nil {
		t.Fatal("expected error for node errors response")
	}
	if resp == nil {
		t.Fatal("expected non-nil response even with node errors")
	}
	if resp.PromptID != "err-456" {
		t.Errorf("expected prompt_id=err-456, got %s", resp.PromptID)
	}
}

func TestGetHistory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/history/prompt-abc" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		resp := HistoryResponse{
			"prompt-abc": HistoryEntry{
				Outputs: map[string]NodeOutput{
					"9": {
						Images: []ImageOutput{
							{Filename: "output_0001.png", Subfolder: "", Type: "output"},
						},
					},
				},
				Status: ExecutionStatus{
					StatusStr: "success",
					Completed: true,
				},
			},
		}
		mustEncodeJSON(t, w, resp)
	}))
	defer server.Close()

	c := newTestClient(server)
	history, err := c.GetHistory("prompt-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	entry, ok := (*history)["prompt-abc"]
	if !ok {
		t.Fatal("expected history entry for prompt-abc")
	}
	if !entry.Status.Completed {
		t.Error("expected completed=true")
	}
	if entry.Status.StatusStr != "success" {
		t.Errorf("expected status=success, got %s", entry.Status.StatusStr)
	}
	nodeOut, ok := entry.Outputs["9"]
	if !ok {
		t.Fatal("expected output for node 9")
	}
	if len(nodeOut.Images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(nodeOut.Images))
	}
	if nodeOut.Images[0].Filename != "output_0001.png" {
		t.Errorf("unexpected filename: %s", nodeOut.Images[0].Filename)
	}
}

func TestGetObjectInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/object_info" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		resp := map[string]NodeInfo{
			"KSampler": {
				Name:        "KSampler",
				DisplayName: "KSampler",
				Category:    "sampling",
				Output:      []string{"LATENT"},
				OutputName:  []string{"LATENT"},
			},
		}
		mustEncodeJSON(t, w, resp)
	}))
	defer server.Close()

	c := newTestClient(server)
	info, err := c.GetObjectInfo()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ks, ok := info["KSampler"]
	if !ok {
		t.Fatal("expected KSampler in object info")
	}
	if ks.Category != "sampling" {
		t.Errorf("expected category=sampling, got %s", ks.Category)
	}
}

func TestGetObjectInfoSingle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]NodeInfo{
			"KSampler": {Name: "KSampler", Category: "sampling"},
		}
		mustEncodeJSON(t, w, resp)
	}))
	defer server.Close()

	c := newTestClient(server)

	// Existing node
	info, err := c.GetObjectInfoSingle("KSampler")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Name != "KSampler" {
		t.Errorf("expected Name=KSampler, got %s", info.Name)
	}

	// Non-existent node
	_, err = c.GetObjectInfoSingle("NonExistent")
	if err == nil {
		t.Fatal("expected error for non-existent node type")
	}
}

func TestGetViewURL(t *testing.T) {
	c := &Client{BaseURL: "http://localhost:8188"}
	url := c.GetViewURL("image.png", "outputs", "output")
	expected := "http://localhost:8188/view?filename=image.png&subfolder=outputs&type=output"
	if url != expected {
		t.Errorf("GetViewURL() = %q, want %q", url, expected)
	}
}

func TestCheckOutputExists(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "HEAD" {
				t.Errorf("expected HEAD, got %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		c := newTestClient(server)
		exists, err := c.CheckOutputExists("img.png", "", "output")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !exists {
			t.Error("expected exists=true")
		}
	})

	t.Run("not_found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		c := newTestClient(server)
		exists, err := c.CheckOutputExists("missing.png", "", "output")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exists {
			t.Error("expected exists=false")
		}
	})
}

func TestDoGetError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		mustWriteBody(t, w, "internal error")
	}))
	defer server.Close()

	c := newTestClient(server)
	_, err := c.GetSystemStats()
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestQueuePromptHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		mustWriteBody(t, w, "service unavailable")
	}))
	defer server.Close()

	c := newTestClient(server)
	_, err := c.QueuePrompt(QueuePromptRequest{Prompt: map[string]interface{}{}})
	if err == nil {
		t.Fatal("expected error for 503 response")
	}
}

func TestAPIKeyHeader(t *testing.T) {
	t.Run("with_key", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer test-key" {
				t.Errorf("expected auth header 'Bearer test-key', got '%s'", auth)
			}
			mustEncodeJSON(t, w, SystemStats{})
		}))
		defer server.Close()

		c := &Client{HTTPClient: server.Client(), BaseURL: server.URL, APIKey: "test-key"}
		_, err := c.GetSystemStats()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("without_key", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth != "" {
				t.Errorf("expected no auth header, got '%s'", auth)
			}
			mustEncodeJSON(t, w, SystemStats{})
		}))
		defer server.Close()

		c := newTestClient(server)
		_, err := c.GetSystemStats()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestAPIKeyHeaderOnPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer post-key" {
			t.Errorf("expected auth header 'Bearer post-key', got '%s'", auth)
		}
		mustEncodeJSON(t, w, QueueResponse{
			PromptID:   "ok",
			NodeErrors: map[string]interface{}{},
		})
	}))
	defer server.Close()

	c := &Client{HTTPClient: server.Client(), BaseURL: server.URL, APIKey: "post-key"}
	_, err := c.QueuePrompt(QueuePromptRequest{Prompt: map[string]interface{}{}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAPIKeyHeaderOnHead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer head-key" {
			t.Errorf("expected auth header 'Bearer head-key', got '%s'", auth)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := &Client{HTTPClient: server.Client(), BaseURL: server.URL, APIKey: "head-key"}
	_, err := c.CheckOutputExists("test.png", "", "output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
