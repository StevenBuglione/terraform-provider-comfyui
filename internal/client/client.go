package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	Host                             string
	Port                             int64
	APIKey                           string
	HTTPClient                       *http.Client
	BaseURL                          string
	DefaultWorkflowExtraData         map[string]interface{}
	ComfyOrgAuthToken                string
	ComfyOrgAPIKey                   string
	UnsupportedDynamicValidationMode string
}

func NewClient(host string, port int64, apiKey string) *Client {
	normalizedHost, normalizedPort, baseURL := normalizeConnection(host, port)
	return &Client{
		Host:   normalizedHost,
		Port:   normalizedPort,
		APIKey: apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		BaseURL: baseURL,
	}
}

func normalizeConnection(host string, port int64) (string, int64, string) {
	host = strings.TrimSpace(host)
	if host == "" {
		host = "localhost"
	}

	var parsed *url.URL
	switch {
	case strings.Contains(host, "://"):
		if candidate, err := url.Parse(host); err == nil && candidate.Host != "" {
			parsed = candidate
		}
	case strings.Contains(host, ":"):
		if candidate, err := url.Parse("//" + host); err == nil && candidate.Host != "" {
			parsed = candidate
		}
	}

	if parsed == nil {
		return host, port, fmt.Sprintf("http://%s:%d", host, port)
	}

	normalizedHost := parsed.Hostname()
	normalizedPort := port
	if parsedPort := parsed.Port(); parsedPort != "" {
		if parsedValue, err := strconv.ParseInt(parsedPort, 10, 64); err == nil {
			normalizedPort = parsedValue
		}
	} else if strings.Contains(host, "://") || strings.Contains(host, "/") {
		switch parsed.Scheme {
		case "https":
			normalizedPort = 443
		default:
			normalizedPort = 80
		}
	} else if normalizedPort == 0 {
		normalizedPort = 80
	}

	scheme := parsed.Scheme
	if scheme == "" {
		scheme = "http"
	}

	return normalizedHost, normalizedPort, fmt.Sprintf("%s://%s:%d", scheme, normalizedHost, normalizedPort)
}

// QueuePrompt submits a workflow for execution.
func (c *Client) QueuePrompt(request QueuePromptRequest) (*QueueResponse, error) {
	bodyMap := map[string]interface{}{
		"prompt": request.Prompt,
	}
	if request.PromptID != "" {
		bodyMap["prompt_id"] = request.PromptID
	}
	if request.ClientID != "" {
		bodyMap["client_id"] = request.ClientID
	}
	if len(request.ExtraData) > 0 {
		bodyMap["extra_data"] = request.ExtraData
	}
	if len(request.PartialExecutionTargets) > 0 {
		bodyMap["partial_execution_targets"] = request.PartialExecutionTargets
	}

	body, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal prompt: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/prompt", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	var result QueueResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode queue response: %w", err)
	}

	if len(result.NodeErrors) > 0 {
		errJSON, _ := json.Marshal(result.NodeErrors)
		return &result, fmt.Errorf("node errors: %s", string(errJSON))
	}

	return &result, nil
}

// WaitForCompletion polls /history/{promptID} until completion or timeout.
func (c *Client) WaitForCompletion(promptID string, timeout time.Duration) (*HistoryEntry, error) {
	deadline := time.Now().Add(timeout)
	pollInterval := 2 * time.Second

	for time.Now().Before(deadline) {
		history, err := c.GetHistory(promptID)
		if err == nil {
			if entry, ok := (*history)[promptID]; ok {
				if entry.Status.Completed {
					return &entry, nil
				}
			}
		}

		if job, err := c.GetJob(promptID); err == nil && job != nil {
			switch strings.ToLower(strings.TrimSpace(job.Status)) {
			case "failed", "error", "cancelled", "canceled":
				return nil, formatJobExecutionError(promptID, job)
			}
		}

		time.Sleep(pollInterval)
	}

	return nil, fmt.Errorf("timeout waiting for prompt %s to complete after %v", promptID, timeout)
}

func formatJobExecutionError(promptID string, job *Job) error {
	if job == nil {
		return fmt.Errorf("prompt %s failed", promptID)
	}
	if len(job.ExecutionError) == 0 {
		return fmt.Errorf("prompt %s failed with status %s", promptID, job.Status)
	}
	if message, ok := job.ExecutionError["exception_message"].(string); ok && strings.TrimSpace(message) != "" {
		if nodeType, ok := job.ExecutionError["node_type"].(string); ok && strings.TrimSpace(nodeType) != "" {
			return fmt.Errorf("prompt %s failed at node %s: %s", promptID, nodeType, message)
		}
		return fmt.Errorf("prompt %s failed: %s", promptID, message)
	}
	raw, err := json.Marshal(job.ExecutionError)
	if err != nil {
		return fmt.Errorf("prompt %s failed with status %s", promptID, job.Status)
	}
	return fmt.Errorf("prompt %s failed with status %s: %s", promptID, job.Status, string(raw))
}

// GetHistory retrieves execution history for a prompt
func (c *Client) GetHistory(promptID string) (*HistoryResponse, error) {
	resp, err := c.doGet(fmt.Sprintf("/history/%s", promptID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result HistoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode history response: %w", err)
	}
	return &result, nil
}

// GetSystemStats retrieves system information
func (c *Client) GetSystemStats() (*SystemStats, error) {
	resp, err := c.doGet("/system_stats")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result SystemStats
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode system stats: %w", err)
	}
	return &result, nil
}

// GetQueue retrieves queue status
func (c *Client) GetQueue() (*QueueStatus, error) {
	resp, err := c.doGet("/queue")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result QueueStatus
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode queue status: %w", err)
	}
	return &result, nil
}

// GetObjectInfo retrieves available node information
func (c *Client) GetObjectInfo() (map[string]NodeInfo, error) {
	resp, err := c.doGet("/object_info")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]NodeInfo
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode object info: %w", err)
	}
	return result, nil
}

// GetObjectInfoSingle retrieves information for a single node type
func (c *Client) GetObjectInfoSingle(nodeType string) (*NodeInfo, error) {
	allInfo, err := c.GetObjectInfo()
	if err != nil {
		return nil, err
	}

	info, ok := allInfo[nodeType]
	if !ok {
		return nil, fmt.Errorf("node type %q not found", nodeType)
	}
	return &info, nil
}

// GetGlobalSubgraphs retrieves the sanitized /global_subgraphs catalog.
func (c *Client) GetGlobalSubgraphs() (map[string]GlobalSubgraphCatalogEntry, error) {
	resp, err := c.doGet("/global_subgraphs")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]GlobalSubgraphCatalogEntry
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode global_subgraphs catalog: %w", err)
	}
	return result, nil
}

// GetGlobalSubgraph retrieves a sanitized /global_subgraphs/{id} definition.
func (c *Client) GetGlobalSubgraph(id string) (*GlobalSubgraphDefinition, error) {
	resp, err := c.doGet(fmt.Sprintf("/global_subgraphs/%s", url.PathEscape(id)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result *GlobalSubgraphDefinition
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode global_subgraphs definition: %w", err)
	}
	return result, nil
}

// GetViewURL constructs the URL for viewing an output file
func (c *Client) GetViewURL(filename, subfolder, outputType string) string {
	values := url.Values{}
	values.Set("filename", filename)
	values.Set("subfolder", subfolder)
	values.Set("type", outputType)
	return fmt.Sprintf("%s/view?%s", c.BaseURL, values.Encode())
}

// CheckOutputExists checks if an output file exists by making a HEAD request
func (c *Client) CheckOutputExists(filename, subfolder, outputType string) (bool, error) {
	url := c.GetViewURL(filename, subfolder, outputType)
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

func (c *Client) UploadImage(filePath string, filename string, subfolder string, uploadType string, overwrite bool) (*UploadResponse, error) {
	return c.uploadFile("/upload/image", filePath, filename, subfolder, uploadType, overwrite, "")
}

func (c *Client) UploadMask(filePath string, filename string, subfolder string, uploadType string, overwrite bool, originalRef RemoteFileReference) (*UploadResponse, error) {
	originalRefJSON, err := originalRef.JSON()
	if err != nil {
		return nil, fmt.Errorf("marshal original_ref: %w", err)
	}
	return c.uploadFile("/upload/mask", filePath, filename, subfolder, uploadType, overwrite, originalRefJSON)
}

func (c *Client) DownloadView(filename, subfolder, outputType string) (*DownloadViewResponse, error) {
	resp, err := c.doGetRaw(c.GetViewURL(filename, subfolder, outputType))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &DownloadViewResponse{
		Content:     content,
		ContentType: resp.Header.Get("Content-Type"),
	}, nil
}

func (c *Client) doGet(path string) (*http.Response, error) {
	url := c.BaseURL + path
	return c.doGetRaw(url)
}

func (c *Client) doGetRaw(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

func (c *Client) uploadFile(endpoint string, filePath string, filename string, subfolder string, uploadType string, overwrite bool, originalRefJSON string) (*UploadResponse, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", filePath, err)
	}
	defer file.Close()

	if filename == "" {
		filename = filepath.Base(filePath)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("image", filename)
	if err != nil {
		return nil, fmt.Errorf("create multipart file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("copy multipart file: %w", err)
	}
	if err := writer.WriteField("type", uploadType); err != nil {
		return nil, fmt.Errorf("write type field: %w", err)
	}
	if err := writer.WriteField("subfolder", subfolder); err != nil {
		return nil, fmt.Errorf("write subfolder field: %w", err)
	}
	if overwrite {
		if err := writer.WriteField("overwrite", "true"); err != nil {
			return nil, fmt.Errorf("write overwrite field: %w", err)
		}
	}
	if originalRefJSON != "" {
		if err := writer.WriteField("original_ref", originalRefJSON); err != nil {
			return nil, fmt.Errorf("write original_ref field: %w", err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+endpoint, &body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	var result UploadResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode upload response: %w", err)
	}
	return &result, nil
}

// GetJob retrieves a single job by ID from GET /api/jobs/{id}
func (c *Client) GetJob(jobID string) (*Job, error) {
	if jobID == "" {
		return nil, fmt.Errorf("job ID cannot be empty")
	}

	resp, err := c.doGet(fmt.Sprintf("/api/jobs/%s", url.PathEscape(jobID)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result Job
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode job response for job %q: %w", jobID, err)
	}
	return &result, nil
}

// ListJobs retrieves a list of jobs from GET /api/jobs with optional filters
func (c *Client) ListJobs(filter JobListFilter) (*JobsResponse, error) {
	values := url.Values{}

	if len(filter.Status) > 0 {
		values.Set("status", strings.Join(filter.Status, ","))
	}

	if filter.WorkflowID != "" {
		values.Set("workflow_id", filter.WorkflowID)
	}

	if filter.SortBy != "" {
		values.Set("sort_by", filter.SortBy)
	}

	if filter.SortOrder != "" {
		values.Set("sort_order", filter.SortOrder)
	}

	if filter.Limit != nil {
		values.Set("limit", fmt.Sprintf("%d", *filter.Limit))
	}

	if filter.Offset != nil {
		values.Set("offset", fmt.Sprintf("%d", *filter.Offset))
	}

	path := "/api/jobs"
	if len(values) > 0 {
		path += "?" + values.Encode()
	}

	resp, err := c.doGet(path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result JobsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode jobs response: %w", err)
	}
	return &result, nil
}

// InterruptPrompt sends POST /interrupt to stop a specific prompt's execution.
// If promptID is empty, sends a global interrupt (no prompt_id in request body).
func (c *Client) InterruptPrompt(promptID string) error {
	bodyMap := map[string]interface{}{}
	if promptID != "" {
		bodyMap["prompt_id"] = promptID
	}

	body, err := json.Marshal(bodyMap)
	if err != nil {
		return fmt.Errorf("failed to marshal interrupt request: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/interrupt", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(errBody))
	}

	return nil
}

// DeleteQueuedPrompt removes a single prompt from the queue
func (c *Client) DeleteQueuedPrompt(promptID string) error {
	if promptID == "" {
		return fmt.Errorf("prompt ID cannot be empty")
	}

	bodyMap := map[string]interface{}{
		"delete": []string{promptID},
	}

	body, err := json.Marshal(bodyMap)
	if err != nil {
		return fmt.Errorf("failed to marshal queue delete request: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/queue", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(errBody))
	}

	return nil
}

// DeleteHistoryPrompt removes a single prompt from history
func (c *Client) DeleteHistoryPrompt(promptID string) error {
	if promptID == "" {
		return fmt.Errorf("prompt ID cannot be empty")
	}

	bodyMap := map[string]interface{}{
		"delete": []string{promptID},
	}

	body, err := json.Marshal(bodyMap)
	if err != nil {
		return fmt.Errorf("failed to marshal history delete request: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/history", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(errBody))
	}

	return nil
}
