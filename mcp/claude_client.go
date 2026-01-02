package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	ProviderClaude       = "claude"
	DefaultClaudeBaseURL = "https://api.anthropic.com/v1"
	DefaultClaudeModel   = "claude-opus-4-5-20251101"
)

type ClaudeClient struct {
	*Client
}

// NewClaudeClient creates Claude client (backward compatible)
func NewClaudeClient() AIClient {
	return NewClaudeClientWithOptions()
}

// NewClaudeClientWithOptions creates Claude client (supports options pattern)
func NewClaudeClientWithOptions(opts ...ClientOption) AIClient {
	// 1. Create Claude preset options
	claudeOpts := []ClientOption{
		WithProvider(ProviderClaude),
		WithModel(DefaultClaudeModel),
		WithBaseURL(DefaultClaudeBaseURL),
	}

	// 2. Merge user options (user options have higher priority)
	allOpts := append(claudeOpts, opts...)

	// 3. Create base client
	baseClient := NewClient(allOpts...).(*Client)

	// 4. Create Claude client
	claudeClient := &ClaudeClient{
		Client: baseClient,
	}

	// 5. Set hooks to point to ClaudeClient (implement dynamic dispatch)
	baseClient.hooks = claudeClient

	return claudeClient
}

func (c *ClaudeClient) SetAPIKey(apiKey string, customURL string, customModel string) {
	c.APIKey = apiKey

	if len(apiKey) > 8 {
		c.logger.Infof("🔧 [MCP] Claude API Key: %s...%s", apiKey[:4], apiKey[len(apiKey)-4:])
	}
	if customURL != "" {
		c.BaseURL = customURL
		c.logger.Infof("🔧 [MCP] Claude using custom BaseURL: %s", customURL)
	} else {
		c.logger.Infof("🔧 [MCP] Claude using default BaseURL: %s", c.BaseURL)
	}
	if customModel != "" {
		c.Model = customModel
		c.logger.Infof("🔧 [MCP] Claude using custom Model: %s", customModel)
	} else {
		c.logger.Infof("🔧 [MCP] Claude using default Model: %s", c.Model)
	}
}

// setAuthHeader Claude uses x-api-key header instead of Authorization Bearer
func (c *ClaudeClient) setAuthHeader(reqHeaders http.Header) {
	reqHeaders.Set("x-api-key", c.APIKey)
	reqHeaders.Set("anthropic-version", "2023-06-01")
}

// buildUrl Claude uses /messages endpoint
func (c *ClaudeClient) buildUrl() string {
	return fmt.Sprintf("%s/messages", c.BaseURL)
}

// buildMCPRequestBody Claude has different request format
func (c *ClaudeClient) buildMCPRequestBody(systemPrompt, userPrompt string) map[string]any {
	requestBody := map[string]any{
		"model":      c.Model,
		"max_tokens": c.MaxTokens,
		"system":     systemPrompt,
		"messages": []map[string]string{
			{"role": "user", "content": userPrompt},
		},
	}

	return requestBody
}

// parseMCPResponse Claude has different response format
func (c *ClaudeClient) parseMCPResponse(body []byte) (string, error) {
	var response struct {
		Content []struct {
			Type  string `json:"type"`
			Text  string `json:"text"`
			Name  string `json:"name,omitempty"`
			Input any    `json:"input,omitempty"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
		Error *struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse Claude response: %w, body: %s", err, string(body))
	}

	if response.Error != nil {
		return "", fmt.Errorf("Claude API error: %s - %s", response.Error.Type, response.Error.Message)
	}

	if len(response.Content) == 0 {
		return "", fmt.Errorf("Claude returned empty content, body: %s", string(body))
	}

	// Report token usage if callback is set
	totalTokens := response.Usage.InputTokens + response.Usage.OutputTokens
	if TokenUsageCallback != nil && totalTokens > 0 {
		TokenUsageCallback(TokenUsage{
			Provider:         c.Provider,
			Model:            c.Model,
			PromptTokens:     response.Usage.InputTokens,
			CompletionTokens: response.Usage.OutputTokens,
			TotalTokens:      totalTokens,
		})
	}

	// Find text content
	for _, content := range response.Content {
		if content.Type == "tool_use" && content.Input != nil {
			data, err := json.Marshal(content.Input)
			if err != nil {
				return "", fmt.Errorf("failed to marshal tool_use input: %w", err)
			}
			return string(data), nil
		}
		if content.Type == "text" {
			return content.Text, nil
		}
	}

	return "", fmt.Errorf("no text content in Claude response")
}

// CallWithRequest calls Claude API using Request object (supports tools)
//
// Note: Claude's /messages API is not OpenAI-compatible, so we must build and parse
// the request/response using Claude-specific formats here.
func (c *ClaudeClient) CallWithRequest(req *Request) (string, error) {
	if c.APIKey == "" {
		return "", fmt.Errorf("AI API key not set, please call SetAPIKey first")
	}
	if req == nil {
		return "", fmt.Errorf("request is nil")
	}

	// If Model is not set in Request, use Client's Model
	if req.Model == "" {
		req.Model = c.Model
	}

	// Build Claude system prompt (Claude uses top-level "system", not a system message)
	var systemParts []string
	messages := make([]map[string]string, 0, len(req.Messages))
	for _, msg := range req.Messages {
		switch msg.Role {
		case "system":
			if msg.Content != "" {
				systemParts = append(systemParts, msg.Content)
			}
		case "user", "assistant":
			messages = append(messages, map[string]string{
				"role":    msg.Role,
				"content": msg.Content,
			})
		}
	}

	maxTokens := c.MaxTokens
	if req.MaxTokens != nil && *req.MaxTokens > 0 {
		maxTokens = *req.MaxTokens
	}

	temperature := c.config.Temperature
	if req.Temperature != nil {
		temperature = *req.Temperature
	}

	requestBody := map[string]any{
		"model":       req.Model,
		"max_tokens":  maxTokens,
		"messages":    messages,
		"temperature": temperature,
	}
	if len(systemParts) > 0 {
		requestBody["system"] = strings.Join(systemParts, "\n\n")
	}

	// Convert OpenAI-style tools to Claude tools format
	if len(req.Tools) > 0 {
		tools := make([]map[string]any, 0, len(req.Tools))
		for _, tool := range req.Tools {
			tools = append(tools, map[string]any{
				"name":         tool.Function.Name,
				"description":  tool.Function.Description,
				"input_schema": tool.Function.Parameters,
			})
		}
		requestBody["tools"] = tools

		// Force tool usage if requested
		if req.ToolChoice == "required" && len(tools) > 0 {
			requestBody["tool_choice"] = map[string]any{
				"type": "tool",
				"name": tools[0]["name"],
			}
		}
	}

	// Retry flow (same semantics as base client)
	var lastErr error
	maxRetries := c.config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 1
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			c.logger.Warnf("鈿狅笍  Claude API call failed, retrying (%d/%d)...", attempt, maxRetries)
		}

		// Serialize request
		jsonData, err := json.Marshal(requestBody)
		if err != nil {
			return "", fmt.Errorf("failed to serialize request: %w", err)
		}

		// Build request
		url := c.buildUrl()
		httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			return "", fmt.Errorf("failed to create request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")
		c.setAuthHeader(httpReq.Header)

		// Send request
		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("failed to send request: %w", err)
		} else {
			bodyBytes, readErr := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if readErr != nil {
				lastErr = fmt.Errorf("failed to read response: %w", readErr)
			} else if resp.StatusCode != http.StatusOK {
				lastErr = fmt.Errorf("API returned error (status %d): %s", resp.StatusCode, string(bodyBytes))
			} else {
				out, parseErr := c.parseMCPResponse(bodyBytes)
				if parseErr == nil {
					if attempt > 1 {
						c.logger.Infof("鉁?Claude API retry succeeded")
					}
					return out, nil
				}
				lastErr = parseErr
			}
		}

		// Check retryability
		if lastErr != nil && !c.hooks.isRetryableError(lastErr) {
			return "", lastErr
		}
		if attempt < maxRetries {
			waitTime := c.config.RetryWaitBase * time.Duration(attempt)
			time.Sleep(waitTime)
		}
	}

	return "", fmt.Errorf("still failed after %d retries: %w", maxRetries, lastErr)
}
