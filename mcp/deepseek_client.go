package mcp

import (
	"net/http"
)

const (
	ProviderDeepSeek       = "deepseek"
	DefaultDeepSeekBaseURL = "https://api.deepseek.com"
	DefaultDeepSeekModel   = "deepseek-chat"
)

type DeepSeekClient struct {
	*Client
}

// NewDeepSeekClient creates DeepSeek client (backward compatible)
//
// Deprecated: Recommend using NewDeepSeekClientWithOptions for better flexibility
func NewDeepSeekClient() AIClient {
	return NewDeepSeekClientWithOptions()
}

// NewDeepSeekClientWithOptions creates DeepSeek client (supports options pattern)
//
// Usage examples:
//   // Basic usage
//   client := mcp.NewDeepSeekClientWithOptions()
//
//   // Custom configuration
//   client := mcp.NewDeepSeekClientWithOptions(
//       mcp.WithAPIKey("sk-xxx"),
//       mcp.WithLogger(customLogger),
//       mcp.WithTimeout(60*time.Second),
//   )
func NewDeepSeekClientWithOptions(opts ...ClientOption) AIClient {
	// 1. Create DeepSeek preset options
	deepseekOpts := []ClientOption{
		WithProvider(ProviderDeepSeek),
		WithModel(DefaultDeepSeekModel),
		WithBaseURL(DefaultDeepSeekBaseURL),
	}

	// 2. Merge user options (user options have higher priority)
	allOpts := append(deepseekOpts, opts...)

	// 3. Create base client
	baseClient := NewClient(allOpts...).(*Client)

	// 4. Create DeepSeek client
	dsClient := &DeepSeekClient{
		Client: baseClient,
	}

	// 5. Set hooks to point to DeepSeekClient (implement dynamic dispatch)
	baseClient.hooks = dsClient

	return dsClient
}

func (dsClient *DeepSeekClient) SetAPIKey(apiKey string, customURL string, customModel string) {
	dsClient.Client.setAPIKeyInternal(apiKey, customURL, customModel)
}

func (dsClient *DeepSeekClient) setAuthHeader(reqHeaders http.Header) {
	dsClient.Client.setAuthHeader(reqHeaders)
}
