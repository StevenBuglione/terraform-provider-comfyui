package client

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
				Outputs: map[string]interface{}{
					"9": map[string]interface{}{
						"images": []interface{}{
							map[string]interface{}{"filename": "output_0001.png", "subfolder": "", "type": "output"},
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
	nodeOut, ok := entry.Outputs["9"].(map[string]interface{})
	if !ok {
		t.Fatal("expected output for node 9")
	}
	images, ok := nodeOut["images"].([]interface{})
	if !ok {
		t.Fatal("expected images output slice for node 9")
	}
	if len(images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(images))
	}
	image, ok := images[0].(map[string]interface{})
	if !ok {
		t.Fatal("expected first image to be an object")
	}
	if image["filename"] != "output_0001.png" {
		t.Errorf("unexpected filename: %v", image["filename"])
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

func TestGetGlobalSubgraphs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/global_subgraphs" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		mustEncodeJSON(t, w, map[string]GlobalSubgraphCatalogEntry{
			"catalog-id": {
				Source: "templates",
				Name:   "Brightness and Contrast",
				Info: GlobalSubgraphInfo{
					NodePack: "comfyui",
				},
			},
		})
	}))
	defer server.Close()

	c := newTestClient(server)
	catalog, err := c.GetGlobalSubgraphs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entry, ok := catalog["catalog-id"]
	if !ok {
		t.Fatalf("expected catalog entry to be present, got %#v", catalog)
	}
	if entry.Source != "templates" {
		t.Fatalf("expected source templates, got %q", entry.Source)
	}
	if entry.Info.NodePack != "comfyui" {
		t.Fatalf("expected node_pack comfyui, got %q", entry.Info.NodePack)
	}
}

func TestGetGlobalSubgraph(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/global_subgraphs/catalog-id" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		mustEncodeJSON(t, w, GlobalSubgraphDefinition{
			Source: "templates",
			Name:   "Brightness and Contrast",
			Info: GlobalSubgraphInfo{
				NodePack: "comfyui",
			},
			Data: `{"revision":0,"last_node_id":140,"last_link_id":0,"nodes":[],"links":[],"groups":[],"definitions":{"subgraphs":[{"id":"916dff42-6166-4d45-b028-04eaf69fbb35","category":"Image Tools/Color adjust"}]},"extra":{},"version":0.4}`,
		})
	}))
	defer server.Close()

	c := newTestClient(server)
	definition, err := c.GetGlobalSubgraph("catalog-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if definition == nil {
		t.Fatal("expected definition response")
	}
	if definition.Name != "Brightness and Contrast" {
		t.Fatalf("expected name to round-trip, got %q", definition.Name)
	}
	if !strings.Contains(definition.Data, `"category":"Image Tools/Color adjust"`) {
		t.Fatalf("expected raw data JSON to round-trip, got %s", definition.Data)
	}
}

func TestGetGlobalSubgraph_NullResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/global_subgraphs/missing-id" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		mustWriteBody(t, w, "null")
	}))
	defer server.Close()

	c := newTestClient(server)
	definition, err := c.GetGlobalSubgraph("missing-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if definition != nil {
		t.Fatalf("expected nil definition for null response, got %#v", definition)
	}
}

func TestGetGlobalSubgraphs_PreservesUnknownInfoFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/global_subgraphs" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		mustWriteBody(t, w, `{"catalog-id":{"source":"templates","name":"Brightness and Contrast","info":{"node_pack":"comfyui","category":"Image Tools","experimental":true}}}`)
	}))
	defer server.Close()

	c := newTestClient(server)
	catalog, err := c.GetGlobalSubgraphs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entry := catalog["catalog-id"]
	infoJSON, err := json.Marshal(entry.Info)
	if err != nil {
		t.Fatalf("marshal info: %v", err)
	}
	if string(infoJSON) != `{"node_pack":"comfyui","category":"Image Tools","experimental":true}` {
		t.Fatalf("expected raw info JSON to preserve unknown fields, got %s", infoJSON)
	}
}

func TestGetGlobalSubgraph_EscapesIDPathSegment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI != "/global_subgraphs/catalog%2Fid%3Fwith%23chars" {
			t.Fatalf("unexpected request URI: %s", r.RequestURI)
		}

		mustWriteBody(t, w, `{"source":"templates","name":"Brightness and Contrast","info":{"node_pack":"comfyui"},"data":"{}"}`)
	}))
	defer server.Close()

	c := newTestClient(server)
	definition, err := c.GetGlobalSubgraph("catalog/id?with#chars")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if definition == nil {
		t.Fatal("expected definition response")
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

func TestUploadImage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "source.png")
	if err := os.WriteFile(path, []byte("image-bytes"), 0644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/upload/image" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("failed to parse multipart form: %v", err)
		}
		if got := r.FormValue("type"); got != "input" {
			t.Fatalf("expected type=input, got %q", got)
		}
		if got := r.FormValue("subfolder"); got != "fixtures" {
			t.Fatalf("expected subfolder=fixtures, got %q", got)
		}
		if got := r.FormValue("overwrite"); got != "true" {
			t.Fatalf("expected overwrite=true, got %q", got)
		}

		file, header, err := r.FormFile("image")
		if err != nil {
			t.Fatalf("missing image form file: %v", err)
		}
		defer file.Close()
		if header.Filename != "renamed.png" {
			t.Fatalf("expected uploaded filename renamed.png, got %q", header.Filename)
		}
		body, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("failed to read uploaded file: %v", err)
		}
		if string(body) != "image-bytes" {
			t.Fatalf("unexpected uploaded body: %q", string(body))
		}

		mustEncodeJSON(t, w, UploadResponse{
			Name:      "renamed.png",
			Subfolder: "fixtures",
			Type:      "input",
		})
	}))
	defer server.Close()

	c := newTestClient(server)
	resp, err := c.UploadImage(path, "renamed.png", "fixtures", "input", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Name != "renamed.png" || resp.Subfolder != "fixtures" || resp.Type != "input" {
		t.Fatalf("unexpected upload response: %#v", resp)
	}
}

func TestUploadMask(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mask.png")
	if err := os.WriteFile(path, []byte("mask-bytes"), 0644); err != nil {
		t.Fatalf("failed to write mask file: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/upload/mask" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("failed to parse multipart form: %v", err)
		}
		if got := r.FormValue("overwrite"); got != "true" {
			t.Fatalf("expected overwrite=true, got %q", got)
		}
		if got := r.FormValue("original_ref"); got != `{"filename":"original.png","subfolder":"masks","type":"output"}` {
			t.Fatalf("unexpected original_ref payload: %q", got)
		}

		file, _, err := r.FormFile("image")
		if err != nil {
			t.Fatalf("missing image form file: %v", err)
		}
		defer file.Close()
		body, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("failed to read uploaded file: %v", err)
		}
		if string(body) != "mask-bytes" {
			t.Fatalf("unexpected uploaded body: %q", string(body))
		}

		mustEncodeJSON(t, w, UploadResponse{
			Name: "mask.png",
			Type: "input",
		})
	}))
	defer server.Close()

	c := newTestClient(server)
	resp, err := c.UploadMask(path, "", "", "input", true, RemoteFileReference{
		Filename:  "original.png",
		Subfolder: "masks",
		Type:      "output",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Name != "mask.png" || resp.Type != "input" {
		t.Fatalf("unexpected upload response: %#v", resp)
	}
}

func TestDownloadView(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/view" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		query := r.URL.Query()
		if query.Get("filename") != "image.png" || query.Get("subfolder") != "outputs" || query.Get("type") != "output" {
			t.Fatalf("unexpected query params: %v", query)
		}
		w.Header().Set("Content-Type", "image/png")
		mustWriteBody(t, w, "png-bytes")
	}))
	defer server.Close()

	c := newTestClient(server)
	resp, err := c.DownloadView("image.png", "outputs", "output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(resp.Content) != "png-bytes" {
		t.Fatalf("unexpected download body: %q", string(resp.Content))
	}
	if resp.ContentType != "image/png" {
		t.Fatalf("expected content_type=image/png, got %q", resp.ContentType)
	}
}

// TestGetJob tests GET /api/jobs/{id}
func TestGetJob(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/jobs/test-job-123" {
			t.Errorf("expected path /api/jobs/test-job-123, got %s", r.URL.Path)
		}
		mustEncodeJSON(t, w, map[string]interface{}{
			"id":                   "test-job-123",
			"status":               "completed",
			"priority":             0,
			"create_time":          1234567890,
			"execution_start_time": 1234567900,
			"execution_end_time":   1234567950,
			"outputs_count":        2,
			"workflow_id":          "wf-abc",
			"preview_output": map[string]interface{}{
				"type":     "image",
				"filename": "preview.png",
			},
			"outputs": map[string]interface{}{
				"3": map[string]interface{}{
					"images": []map[string]interface{}{
						{"filename": "output_01.png", "subfolder": "", "type": "output"},
					},
				},
			},
			"execution_status": map[string]interface{}{
				"status_str": "success",
				"completed":  true,
			},
			"workflow": map[string]interface{}{
				"prompt": map[string]interface{}{
					"1": map[string]interface{}{"class_type": "LoadImage"},
				},
				"extra_data": map[string]interface{}{
					"client_id": "test-client",
				},
			},
		})
	}))
	defer server.Close()

	c := newTestClient(server)
	job, err := c.GetJob("test-job-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if job.ID != "test-job-123" {
		t.Errorf("expected ID=test-job-123, got %s", job.ID)
	}
	if job.Status != "completed" {
		t.Errorf("expected status=completed, got %s", job.Status)
	}
	if job.WorkflowID != "wf-abc" {
		t.Errorf("expected workflow_id=wf-abc, got %s", job.WorkflowID)
	}
	if job.OutputsCount == nil || *job.OutputsCount != 2 {
		t.Errorf("expected outputs_count=2, got %v", job.OutputsCount)
	}
	if job.CreateTime == nil || *job.CreateTime != 1234567890 {
		t.Errorf("expected create_time=1234567890, got %v", job.CreateTime)
	}
	if job.ExecutionStartTime == nil || *job.ExecutionStartTime != 1234567900 {
		t.Errorf("expected execution_start_time=1234567900, got %v", job.ExecutionStartTime)
	}
	if job.ExecutionEndTime == nil || *job.ExecutionEndTime != 1234567950 {
		t.Errorf("expected execution_end_time=1234567950, got %v", job.ExecutionEndTime)
	}
	if job.PreviewOutput == nil {
		t.Fatal("expected preview_output to be non-nil")
	}
	if job.Outputs == nil {
		t.Fatal("expected outputs to be non-nil")
	}
	if job.ExecutionStatus == nil {
		t.Fatal("expected execution_status to be non-nil")
	}
	if job.Workflow == nil {
		t.Fatal("expected workflow to be non-nil")
	}
	if job.Workflow.Prompt == nil {
		t.Fatal("expected workflow.prompt to be non-nil")
	}
	if job.Workflow.ExtraData == nil {
		t.Fatal("expected workflow.extra_data to be non-nil")
	}
}

// TestGetJob_PathEscaping tests special characters in job ID
func TestGetJob_PathEscaping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The URL path is automatically decoded by the server
		if r.URL.Path != "/api/jobs/job/with/slashes" {
			t.Errorf("expected path /api/jobs/job/with/slashes, got %s", r.URL.Path)
		}
		mustEncodeJSON(t, w, map[string]interface{}{
			"id":            "job/with/slashes",
			"status":        "queued",
			"outputs_count": 0,
		})
	}))
	defer server.Close()

	c := newTestClient(server)
	job, err := c.GetJob("job/with/slashes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if job.ID != "job/with/slashes" {
		t.Errorf("expected ID=job/with/slashes, got %s", job.ID)
	}
}

// TestGetJob_EmptyID tests validation for empty job ID
func TestGetJob_EmptyID(t *testing.T) {
	c := NewClient("localhost", 8188, "")
	_, err := c.GetJob("")
	if err == nil {
		t.Fatal("expected error for empty job ID, got nil")
	}
	if !strings.Contains(err.Error(), "cannot be empty") {
		t.Errorf("expected error about empty ID, got %v", err)
	}
}

// TestGetJob_DecodeError tests error handling for malformed JSON
func TestGetJob_DecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mustWriteBody(t, w, "{invalid json")
	}))
	defer server.Close()

	c := newTestClient(server)
	_, err := c.GetJob("test-job-123")
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to decode job response") {
		t.Errorf("expected decode error message, got %v", err)
	}
	if !strings.Contains(err.Error(), "test-job-123") {
		t.Errorf("expected job ID in error message, got %v", err)
	}
}

// TestGetJob_HTTPError tests handling of non-200 response
func TestGetJob_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		mustWriteBody(t, w, "job not found")
	}))
	defer server.Close()

	c := newTestClient(server)
	_, err := c.GetJob("nonexistent")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected 404 in error, got %v", err)
	}
}

// TestListJobs tests GET /api/jobs with filters
func TestListJobs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/jobs" {
			t.Errorf("expected path /api/jobs, got %s", r.URL.Path)
		}
		// Check query parameters
		q := r.URL.Query()
		if q.Get("status") != "completed,failed" {
			t.Errorf("expected status=completed,failed, got %s", q.Get("status"))
		}
		if q.Get("sort_by") != "created_at" {
			t.Errorf("expected sort_by=created_at, got %s", q.Get("sort_by"))
		}
		if q.Get("limit") != "10" {
			t.Errorf("expected limit=10, got %s", q.Get("limit"))
		}

		mustEncodeJSON(t, w, map[string]interface{}{
			"jobs": []map[string]interface{}{
				{
					"id":            "job-1",
					"status":        "completed",
					"outputs_count": 2,
				},
				{
					"id":            "job-2",
					"status":        "failed",
					"outputs_count": 0,
				},
			},
			"has_more": false,
		})
	}))
	defer server.Close()

	c := newTestClient(server)
	filter := JobListFilter{
		Status: []string{"completed", "failed"},
		SortBy: "created_at",
		Limit:  intPtr(10),
	}
	result, err := c.ListJobs(filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Jobs) != 2 {
		t.Errorf("expected 2 jobs, got %d", len(result.Jobs))
	}
	if result.Jobs[0].ID != "job-1" {
		t.Errorf("expected first job ID=job-1, got %s", result.Jobs[0].ID)
	}
	if result.HasMore {
		t.Errorf("expected has_more=false, got true")
	}
}

// TestInterruptPrompt tests POST /interrupt
func TestInterruptPrompt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/interrupt" {
			t.Errorf("expected path /interrupt, got %s", r.URL.Path)
		}

		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if reqBody["prompt_id"] != "prompt-to-interrupt" {
			t.Errorf("expected prompt_id=prompt-to-interrupt, got %v", reqBody["prompt_id"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := newTestClient(server)
	err := c.InterruptPrompt("prompt-to-interrupt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestInterruptPrompt_GlobalInterrupt tests POST /interrupt with empty promptID (global interrupt)
func TestInterruptPrompt_GlobalInterrupt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/interrupt" {
			t.Errorf("expected path /interrupt, got %s", r.URL.Path)
		}

		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		// Global interrupt should not have prompt_id field
		if _, exists := reqBody["prompt_id"]; exists {
			t.Errorf("expected no prompt_id field for global interrupt, got %v", reqBody["prompt_id"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := newTestClient(server)
	err := c.InterruptPrompt("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestInterruptPrompt_HTTPError tests error handling for non-200 response
func TestInterruptPrompt_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		mustWriteBody(t, w, "internal server error")
	}))
	defer server.Close()

	c := newTestClient(server)
	err := c.InterruptPrompt("prompt-123")
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected 500 in error, got %v", err)
	}
}

// TestDeleteQueuedPrompt tests POST /queue with delete
func TestDeleteQueuedPrompt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/queue" {
			t.Errorf("expected path /queue, got %s", r.URL.Path)
		}

		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		deleteList, ok := reqBody["delete"].([]interface{})
		if !ok {
			t.Fatalf("expected delete to be an array")
		}
		if len(deleteList) != 1 {
			t.Errorf("expected 1 item to delete, got %d", len(deleteList))
		}
		if deleteList[0] != "item-1" {
			t.Errorf("expected delete item=item-1, got %v", deleteList[0])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := newTestClient(server)
	err := c.DeleteQueuedPrompt("item-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestDeleteQueuedPrompt_HTTPError tests error handling for non-200 response
func TestDeleteQueuedPrompt_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		mustWriteBody(t, w, "bad request")
	}))
	defer server.Close()

	c := newTestClient(server)
	err := c.DeleteQueuedPrompt("item-1")
	if err == nil {
		t.Fatal("expected error for 400, got nil")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("expected 400 in error, got %v", err)
	}
}

// TestDeleteHistoryPrompt tests POST /history with delete
func TestDeleteHistoryPrompt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/history" {
			t.Errorf("expected path /history, got %s", r.URL.Path)
		}

		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		deleteList, ok := reqBody["delete"].([]interface{})
		if !ok {
			t.Fatalf("expected delete to be an array")
		}
		if len(deleteList) != 1 {
			t.Errorf("expected 1 item to delete, got %d", len(deleteList))
		}
		if deleteList[0] != "history-1" {
			t.Errorf("unexpected delete item: %v", deleteList[0])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := newTestClient(server)
	err := c.DeleteHistoryPrompt("history-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestDeleteHistoryPrompt_HTTPError tests error handling for non-200 response
func TestDeleteHistoryPrompt_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		mustWriteBody(t, w, "not found")
	}))
	defer server.Close()

	c := newTestClient(server)
	err := c.DeleteHistoryPrompt("nonexistent")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected 404 in error, got %v", err)
	}
}

// TestDeleteQueuedPrompt_EmptyID tests that empty prompt ID is rejected
func TestDeleteQueuedPrompt_EmptyID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called for empty ID")
	}))
	defer server.Close()

	c := newTestClient(server)
	err := c.DeleteQueuedPrompt("")
	if err == nil {
		t.Fatal("expected error for empty ID, got nil")
	}
	if !strings.Contains(err.Error(), "prompt ID cannot be empty") {
		t.Errorf("expected 'prompt ID cannot be empty' error, got %v", err)
	}
}

// TestDeleteHistoryPrompt_EmptyID tests that empty prompt ID is rejected
func TestDeleteHistoryPrompt_EmptyID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called for empty ID")
	}))
	defer server.Close()

	c := newTestClient(server)
	err := c.DeleteHistoryPrompt("")
	if err == nil {
		t.Fatal("expected error for empty ID, got nil")
	}
	if !strings.Contains(err.Error(), "prompt ID cannot be empty") {
		t.Errorf("expected 'prompt ID cannot be empty' error, got %v", err)
	}
}

// TestListJobs_EmptyFilter tests that empty filter doesn't send query parameters
func TestListJobs_EmptyFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" {
			t.Errorf("expected path /api/jobs, got %s", r.URL.Path)
		}
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query parameters, got %s", r.URL.RawQuery)
		}
		mustEncodeJSON(t, w, JobsResponse{
			Jobs:    []Job{},
			HasMore: false,
		})
	}))
	defer server.Close()

	c := newTestClient(server)
	_, err := c.ListJobs(JobListFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestListJobs_HTTPError tests that HTTP errors are surfaced via doGet
func TestListJobs_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		mustWriteBody(t, w, "internal error")
	}))
	defer server.Close()

	c := newTestClient(server)
	_, err := c.ListJobs(JobListFilter{})
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected 500 in error, got %v", err)
	}
}

// TestListJobs_DecodeError tests error handling for malformed JSON
func TestListJobs_DecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mustWriteBody(t, w, "{malformed json")
	}))
	defer server.Close()

	c := newTestClient(server)
	_, err := c.ListJobs(JobListFilter{})
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to decode jobs response") {
		t.Errorf("expected decode error message, got %v", err)
	}
}

func intPtr(i int) *int {
	return &i
}
