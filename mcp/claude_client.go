package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const (
	ProviderClaude       = "claude"
	DefaultClaudeBaseURL = "https://api.anthropic.com" // SDK adds /v1/messages automatically
	DefaultClaudeModel   = "claude-opus-4-5-20251101"
)

type ClaudeClient struct {
	*Client
	sdkClient    anthropic.Client // Anthropic official SDK client (value type, not pointer)
	sdkInitialized bool           // Flag to check if SDK client is initialized
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

	// 4. Create Claude client with SDK
	claudeClient := &ClaudeClient{
		Client: baseClient,
	}

	// 5. Initialize SDK client if API key is available
	if baseClient.APIKey != "" {
		claudeClient.sdkClient = anthropic.NewClient(
			option.WithAPIKey(baseClient.APIKey),
			option.WithBaseURL(baseClient.BaseURL),
		)
		claudeClient.sdkInitialized = true
	}

	// 6. Set hooks to point to ClaudeClient (implement dynamic dispatch)
	baseClient.hooks = claudeClient

	return claudeClient
}

func (c *ClaudeClient) SetAPIKey(apiKey string, customURL string, customModel string) {
	// Use base class helper for common logic
	c.Client.setAPIKeyInternal(apiKey, customURL, customModel)

	// Recreate SDK client with new configuration
	sdkOpts := []option.RequestOption{
		option.WithAPIKey(apiKey),
	}
	if c.BaseURL != "" {
		sdkOpts = append(sdkOpts, option.WithBaseURL(c.BaseURL))
	}
	c.sdkClient = anthropic.NewClient(sdkOpts...)
	c.sdkInitialized = true
}

// setAuthHeader Claude uses x-api-key header instead of Authorization Bearer
func (c *ClaudeClient) setAuthHeader(reqHeaders http.Header) {
	reqHeaders.Set("x-api-key", c.APIKey)
	reqHeaders.Set("anthropic-version", "2023-06-01")
}

// buildUrl Claude uses /v1/messages endpoint
func (c *ClaudeClient) buildUrl() string {
	return fmt.Sprintf("%s/v1/messages", c.BaseURL)
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
// Uses anthropic-sdk-go for type-safe API calls
func (c *ClaudeClient) CallWithRequest(req *Request) (string, error) {
	if !c.sdkInitialized {
		return "", fmt.Errorf("AI API key not set, please call SetAPIKey first")
	}
	if req == nil {
		return "", fmt.Errorf("request is nil")
	}

	// If Model is not set in Request, use Client's Model
	model := req.Model
	if model == "" {
		model = c.Model
	}

	// Build SDK message params
	params, err := c.buildSDKMessageParams(req, model)
	if err != nil {
		return "", err
	}

	// Retry flow (same semantics as base client)
	var lastErr error
	maxRetries := c.config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 1
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			c.logger.Warnf("⚠️  Claude API call failed, retrying (%d/%d)...", attempt, maxRetries)
		}

		// Call SDK
		ctx := context.Background()
		message, err := c.sdkClient.Messages.New(ctx, params)
		if err == nil {
			if attempt > 1 {
				c.logger.Infof("✓ Claude API retry succeeded")
			}
			return c.extractSDKResponse(message)
		}

		lastErr = err
		// Check retryability
		if !c.isRetryableSDKError(err) {
			return "", err
		}
		if attempt < maxRetries {
			waitTime := c.config.RetryWaitBase * time.Duration(attempt)
			time.Sleep(waitTime)
		}
	}

	return "", fmt.Errorf("still failed after %d retries: %w", maxRetries, lastErr)
}

// buildSDKMessageParams converts mcp.Request to anthropic.MessageNewParams
func (c *ClaudeClient) buildSDKMessageParams(req *Request, model string) (anthropic.MessageNewParams, error) {
	// Extract system prompt and build messages
	var systemBlocks []anthropic.TextBlockParam
	sdkMessages := make([]anthropic.MessageParam, 0, len(req.Messages))

	for _, msg := range req.Messages {
		switch msg.Role {
		case "system":
			if msg.Content != "" {
				systemBlocks = append(systemBlocks, anthropic.TextBlockParam{Text: msg.Content})
			}
		case "user":
			if len(msg.Parts) > 0 {
				blocks := c.convertPartsToSDK(msg.Parts)
				sdkMessages = append(sdkMessages, anthropic.NewUserMessage(blocks...))
			} else if msg.Content != "" {
				sdkMessages = append(sdkMessages, anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content)))
			}
		case "assistant":
			if msg.Content != "" {
				sdkMessages = append(sdkMessages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(msg.Content)))
			}
		}
	}

	// Determine max tokens
	maxTokens := int64(c.MaxTokens)
	if req.MaxTokens != nil && *req.MaxTokens > 0 {
		maxTokens = int64(*req.MaxTokens)
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: maxTokens,
		Messages:  sdkMessages,
	}

	// Add system prompt if present
	if len(systemBlocks) > 0 {
		params.System = systemBlocks
	}

	// Add temperature if set
	if req.Temperature != nil {
		params.Temperature = anthropic.Float(*req.Temperature)
	} else {
		params.Temperature = anthropic.Float(c.config.Temperature)
	}

	// Convert tools if present
	if len(req.Tools) > 0 {
		sdkTools := make([]anthropic.ToolUnionParam, 0, len(req.Tools))
		for _, tool := range req.Tools {
			// Build InputSchema from JSON Schema parameters
			// Parameters is a full JSON Schema with type, properties, required fields
			inputSchema := anthropic.ToolInputSchemaParam{}

			if props, ok := tool.Function.Parameters["properties"]; ok {
				inputSchema.Properties = props
			}
			if required, ok := tool.Function.Parameters["required"].([]any); ok {
				reqStrings := make([]string, 0, len(required))
				for _, r := range required {
					if s, ok := r.(string); ok {
						reqStrings = append(reqStrings, s)
					}
				}
				inputSchema.Required = reqStrings
			} else if required, ok := tool.Function.Parameters["required"].([]string); ok {
				inputSchema.Required = required
			}

			sdkTools = append(sdkTools, anthropic.ToolUnionParam{
				OfTool: &anthropic.ToolParam{
					Name:        tool.Function.Name,
					Description: anthropic.String(tool.Function.Description),
					InputSchema: inputSchema,
				},
			})
		}
		params.Tools = sdkTools

		// Set tool choice if required
		if req.ToolChoice == "required" && len(sdkTools) > 0 {
			params.ToolChoice = anthropic.ToolChoiceUnionParam{
				OfTool: &anthropic.ToolChoiceToolParam{
					Name: req.Tools[0].Function.Name,
				},
			}
		}
	}

	return params, nil
}

// convertPartsToSDK converts mcp.ContentPart to SDK content blocks
func (c *ClaudeClient) convertPartsToSDK(parts []ContentPart) []anthropic.ContentBlockParamUnion {
	blocks := make([]anthropic.ContentBlockParamUnion, 0, len(parts))
	for _, p := range parts {
		switch p.Type {
		case "text":
			if strings.TrimSpace(p.Text) != "" {
				blocks = append(blocks, anthropic.NewTextBlock(p.Text))
			}
		case "image":
			mediaType, data, ok := parseDataURI(p.DataURI)
			if !ok {
				c.logger.Warnf("⚠️  Claude: invalid image data URI, skipping")
				continue
			}
			blocks = append(blocks, anthropic.NewImageBlock(anthropic.Base64ImageSourceParam{
				MediaType: anthropic.Base64ImageSourceMediaType(mediaType),
				Data:      data,
			}))
		}
	}
	return blocks
}

// extractSDKResponse extracts text or tool_use from SDK response
func (c *ClaudeClient) extractSDKResponse(message *anthropic.Message) (string, error) {
	// Report token usage if callback is set
	totalTokens := int(message.Usage.InputTokens + message.Usage.OutputTokens)
	if TokenUsageCallback != nil && totalTokens > 0 {
		TokenUsageCallback(TokenUsage{
			Provider:         c.Provider,
			Model:            c.Model,
			PromptTokens:     int(message.Usage.InputTokens),
			CompletionTokens: int(message.Usage.OutputTokens),
			TotalTokens:      totalTokens,
		})
	}

	if len(message.Content) == 0 {
		return "", fmt.Errorf("Claude returned empty content")
	}

	// Find text or tool_use content
	for _, block := range message.Content {
		switch b := block.AsAny().(type) {
		case anthropic.ToolUseBlock:
			// Return tool input as JSON string
			data, err := json.Marshal(b.Input)
			if err != nil {
				return "", fmt.Errorf("failed to marshal tool_use input: %w", err)
			}
			return string(data), nil
		case anthropic.TextBlock:
			return b.Text, nil
		}
	}

	return "", fmt.Errorf("no text content in Claude response")
}

// isRetryableSDKError checks if SDK error is retryable
func (c *ClaudeClient) isRetryableSDKError(err error) bool {
	// Check for SDK API error with status code
	var apiErr *anthropic.Error
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case 429: // Rate limit
			return true
		case 500, 502, 503, 504: // Server errors
			return true
		case 529: // Overloaded
			return true
		}
		return false
	}

	// Fallback to string matching for network errors
	errStr := strings.ToLower(err.Error())
	networkPatterns := []string{
		"timeout",
		"connection reset",
		"connection refused",
		"eof",
		"temporary failure",
	}

	for _, pattern := range networkPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	return false
}

func parseDataURI(raw string) (mediaType, data string, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || !strings.HasPrefix(raw, "data:") {
		return "", "", false
	}
	comma := strings.Index(raw, ",")
	if comma < 0 {
		return "", "", false
	}
	meta := strings.TrimSpace(raw[len("data:"):comma])
	data = strings.TrimSpace(raw[comma+1:])
	if data == "" {
		return "", "", false
	}
	parts := strings.Split(meta, ";")
	if len(parts) == 0 {
		return "", "", false
	}
	mediaType = strings.TrimSpace(parts[0])
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}
	hasBase64 := false
	for _, part := range parts[1:] {
		if strings.EqualFold(strings.TrimSpace(part), "base64") {
			hasBase64 = true
			break
		}
	}
	if !hasBase64 {
		return "", "", false
	}
	return mediaType, data, true
}
