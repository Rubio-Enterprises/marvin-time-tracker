package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const marvinBaseURL = "https://serv.amazingmarvin.com/api"

// TrackedItemResponse represents the response from GET /api/trackedItem.
type TrackedItemResponse struct {
	TaskID  string `json:"taskId"`
	Title   string `json:"title"`
	StartedAt int64 `json:"startedAt"`
}

// MarvinAPIClient interfaces with the Marvin API.
type MarvinAPIClient interface {
	GetTrackedItem() (*TrackedItemResponse, error)
	Track(taskID string, action string) error
}

type marvinClient struct {
	httpClient *http.Client
	token      string
	baseURL    string
}

func NewMarvinClient(token string) MarvinAPIClient {
	return &marvinClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		token:      token,
		baseURL:    marvinBaseURL,
	}
}

func (mc *marvinClient) GetTrackedItem() (*TrackedItemResponse, error) {
	req, err := http.NewRequest(http.MethodGet, mc.baseURL+"/trackedItem", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Token", mc.token)

	resp, err := mc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("marvin API error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("marvin API returned %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Empty body means no tracked item
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" || trimmed == "null" || trimmed == "{}" {
		return nil, nil
	}

	var item TrackedItemResponse
	if err := json.Unmarshal(body, &item); err != nil {
		return nil, fmt.Errorf("marvin API decode error: %w", err)
	}

	if item.TaskID == "" {
		return nil, nil
	}

	return &item, nil
}

func (mc *marvinClient) Track(taskID string, action string) error {
	payload := fmt.Sprintf(`{"taskId":"%s","action":"%s"}`, taskID, action)
	req, err := http.NewRequest(http.MethodPost, mc.baseURL+"/track", strings.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("X-API-Token", mc.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := mc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("marvin track error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("marvin track returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
