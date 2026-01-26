package mcp

import (
	"net/http"
)

const (
	ProviderKimi       = "kimi"
	DefaultKimiBaseURL = "https://api.moonshot.ai/v1" // Global endpoint (use api.moonshot.cn for China)
	DefaultKimiModel   = "moonshot-v1-auto"
)

type KimiClient struct {
	*Client
}

// NewKimiClient creates Kimi (Moonshot) client (backward compatible)
func NewKimiClient() AIClient {
	return NewKimiClientWithOptions()
}

// NewKimiClientWithOptions creates Kimi client (supports options pattern)
func NewKimiClientWithOptions(opts ...ClientOption) AIClient {
	// 1. Create Kimi preset options
	kimiOpts := []ClientOption{
		WithProvider(ProviderKimi),
		WithModel(DefaultKimiModel),
		WithBaseURL(DefaultKimiBaseURL),
	}

	// 2. Merge user options (user options have higher priority)
	allOpts := append(kimiOpts, opts...)

	// 3. Create base client
	baseClient := NewClient(allOpts...).(*Client)

	// 4. Create Kimi client
	kimiClient := &KimiClient{
		Client: baseClient,
	}

	// 5. Set hooks to point to KimiClient (implement dynamic dispatch)
	baseClient.hooks = kimiClient

	return kimiClient
}

func (c *KimiClient) SetAPIKey(apiKey string, customURL string, customModel string) {
	c.Client.setAPIKeyInternal(apiKey, customURL, customModel)
}

// Kimi uses standard OpenAI-compatible API, so we just use the base client methods
func (c *KimiClient) setAuthHeader(reqHeaders http.Header) {
	c.Client.setAuthHeader(reqHeaders)
}
