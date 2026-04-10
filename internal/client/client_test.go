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
		json.NewEncoder(w).Encode(SystemStats{
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
		json.NewEncoder(w).Encode(QueueStatus{
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

		json.NewEncoder(w).Encode(QueueResponse{
			PromptID:   "test-123",
			Number:     1,
			NodeErrors: map[string]interface{}{},
		})
	}))
	defer server.Close()

	c := newTestClient(server)
	resp, err := c.QueuePrompt(map[string]interface{}{
		"1": map[string]interface{}{
			"class_type": "KSampler",
			"inputs":     map[string]interface{}{},
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

func TestQueuePromptNodeErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(QueueResponse{
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
	resp, err := c.QueuePrompt(map[string]interface{}{})
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
		json.NewEncoder(w).Encode(resp)
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
		json.NewEncoder(w).Encode(resp)
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
		json.NewEncoder(w).Encode(resp)
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
		w.Write([]byte("internal error"))
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
		w.Write([]byte("service unavailable"))
	}))
	defer server.Close()

	c := newTestClient(server)
	_, err := c.QueuePrompt(map[string]interface{}{})
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
			json.NewEncoder(w).Encode(SystemStats{})
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
			json.NewEncoder(w).Encode(SystemStats{})
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
		json.NewEncoder(w).Encode(QueueResponse{
			PromptID:   "ok",
			NodeErrors: map[string]interface{}{},
		})
	}))
	defer server.Close()

	c := &Client{HTTPClient: server.Client(), BaseURL: server.URL, APIKey: "post-key"}
	_, err := c.QueuePrompt(map[string]interface{}{})
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
