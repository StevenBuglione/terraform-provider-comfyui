package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	Host       string
	Port       int64
	APIKey     string
	HTTPClient *http.Client
	BaseURL    string
}

func NewClient(host string, port int64, apiKey string) *Client {
	return &Client{
		Host:   host,
		Port:   port,
		APIKey: apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		BaseURL: fmt.Sprintf("http://%s:%d", host, port),
	}
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
		if err != nil {
			time.Sleep(pollInterval)
			continue
		}

		if entry, ok := (*history)[promptID]; ok {
			if entry.Status.Completed {
				return &entry, nil
			}
		}

		time.Sleep(pollInterval)
	}

	return nil, fmt.Errorf("timeout waiting for prompt %s to complete after %v", promptID, timeout)
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

// GetViewURL constructs the URL for viewing an output file
func (c *Client) GetViewURL(filename, subfolder, outputType string) string {
	return fmt.Sprintf("%s/view?filename=%s&subfolder=%s&type=%s",
		c.BaseURL, filename, subfolder, outputType)
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

func (c *Client) doGet(path string) (*http.Response, error) {
	url := c.BaseURL + path
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
