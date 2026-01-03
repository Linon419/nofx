package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

type ModelListItem struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// ListModels fetches the provider's available model list using the configured BaseURL and auth headers.
//
// Notes:
// - For OpenAI-compatible providers, BaseURL should typically end with "/v1".
// - If UseFullURL is enabled (BaseURL ends with "#"), model listing is not supported since BaseURL is the full chat endpoint.
func (client *Client) ListModels(ctx context.Context) ([]ModelListItem, error) {
	if client == nil {
		return nil, fmt.Errorf("mcp client is nil")
	}
	if strings.TrimSpace(client.APIKey) == "" {
		return nil, fmt.Errorf("AI API key not set, please call SetAPIKey first")
	}
	if client.UseFullURL {
		return nil, fmt.Errorf("cannot list models when base_url ends with '#'; set base_url to the provider root (e.g. https://api.openai.com/v1) without trailing '#'")
	}

	baseURL := strings.TrimRight(strings.TrimSpace(client.BaseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("base url is empty")
	}

	url := baseURL
	if !strings.HasSuffix(strings.ToLower(baseURL), "/models") {
		url = baseURL + "/models"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	client.hooks.setAuthHeader(req.Header)

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned error (status %d): %s", resp.StatusCode, string(body))
	}

	type listResponse struct {
		Data []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			DisplayName string `json:"display_name"`
		} `json:"data"`
		Models []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			DisplayName string `json:"display_name"`
		} `json:"models"`
	}

	var parsed listResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse models response: %w", err)
	}

	items := make([]ModelListItem, 0, len(parsed.Data)+len(parsed.Models))
	add := func(id, name string) {
		id = strings.TrimSpace(id)
		if id == "" {
			return
		}
		items = append(items, ModelListItem{ID: id, Name: strings.TrimSpace(name)})
	}

	for _, m := range parsed.Data {
		name := m.Name
		if name == "" {
			name = m.DisplayName
		}
		add(m.ID, name)
	}
	for _, m := range parsed.Models {
		name := m.Name
		if name == "" {
			name = m.DisplayName
		}
		add(m.ID, name)
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("no models returned")
	}

	// Stable order + de-dup by ID
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	out := make([]ModelListItem, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, it := range items {
		if _, ok := seen[it.ID]; ok {
			continue
		}
		seen[it.ID] = struct{}{}
		out = append(out, it)
	}

	return out, nil
}
