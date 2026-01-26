package mcp

import (
	"net/http"
)

const (
	ProviderQwen       = "qwen"
	DefaultQwenBaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	DefaultQwenModel   = "qwen3-max"
)

type QwenClient struct {
	*Client
}

// NewQwenClient creates Qwen client (backward compatible)
//
// Deprecated: Recommend using NewQwenClientWithOptions for better flexibility
func NewQwenClient() AIClient {
	return NewQwenClientWithOptions()
}

// NewQwenClientWithOptions creates Qwen client (supports options pattern)
//
// Usage examples:
//   // Basic usage
//   client := mcp.NewQwenClientWithOptions()
//
//   // Custom configuration
//   client := mcp.NewQwenClientWithOptions(
//       mcp.WithAPIKey("sk-xxx"),
//       mcp.WithLogger(customLogger),
//       mcp.WithTimeout(60*time.Second),
//   )
func NewQwenClientWithOptions(opts ...ClientOption) AIClient {
	// 1. Create Qwen preset options
	qwenOpts := []ClientOption{
		WithProvider(ProviderQwen),
		WithModel(DefaultQwenModel),
		WithBaseURL(DefaultQwenBaseURL),
	}

	// 2. Merge user options (user options have higher priority)
	allOpts := append(qwenOpts, opts...)

	// 3. Create base client
	baseClient := NewClient(allOpts...).(*Client)

	// 4. Create Qwen client
	qwenClient := &QwenClient{
		Client: baseClient,
	}

	// 5. Set hooks to point to QwenClient (implement dynamic dispatch)
	baseClient.hooks = qwenClient

	return qwenClient
}

func (qwenClient *QwenClient) SetAPIKey(apiKey string, customURL string, customModel string) {
	qwenClient.Client.setAPIKeyInternal(apiKey, customURL, customModel)
}

func (qwenClient *QwenClient) setAuthHeader(reqHeaders http.Header) {
	qwenClient.Client.setAuthHeader(reqHeaders)
}
