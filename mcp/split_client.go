package mcp

import (
	"fmt"
	"time"
)

// SplitClient routes requests to different underlying clients:
// - Text/tool requests go to decisionClient
// - Vision (image) requests go to visionClient (if available), otherwise decisionClient
//
// This allows using separate API keys/models for chart reading vs decision making.
type SplitClient struct {
	decisionClient AIClient
	visionClient   AIClient
}

func NewSplitClient(decisionClient, visionClient AIClient) *SplitClient {
	return &SplitClient{
		decisionClient: decisionClient,
		visionClient:   visionClient,
	}
}

func (c *SplitClient) DecisionClient() AIClient { return c.decisionClient }
func (c *SplitClient) VisionClient() AIClient   { return c.visionClient }

func (c *SplitClient) SetAPIKey(apiKey string, customURL string, customModel string) {
	// Convenience: set on both. Callers that need different keys should set on the underlying clients directly.
	if c.decisionClient != nil {
		c.decisionClient.SetAPIKey(apiKey, customURL, customModel)
	}
	if c.visionClient != nil && c.visionClient != c.decisionClient {
		c.visionClient.SetAPIKey(apiKey, customURL, customModel)
	}
}

func (c *SplitClient) SetTimeout(timeout time.Duration) {
	if c.decisionClient != nil {
		c.decisionClient.SetTimeout(timeout)
	}
	if c.visionClient != nil && c.visionClient != c.decisionClient {
		c.visionClient.SetTimeout(timeout)
	}
}

func (c *SplitClient) CallWithMessages(systemPrompt, userPrompt string) (string, error) {
	if c.decisionClient == nil {
		return "", fmt.Errorf("decision client is nil")
	}
	return c.decisionClient.CallWithMessages(systemPrompt, userPrompt)
}

func (c *SplitClient) CallWithRequest(req *Request) (string, error) {
	if c.decisionClient == nil {
		return "", fmt.Errorf("decision client is nil")
	}
	target := c.decisionClient
	if requestHasImageParts(req) && c.visionClient != nil && isVisionCapableClient(c.visionClient) {
		target = c.visionClient
	}
	return target.CallWithRequest(req)
}

func requestHasImageParts(req *Request) bool {
	if req == nil {
		return false
	}
	for _, msg := range req.Messages {
		for _, part := range msg.Parts {
			if part.Type == "image" && part.DataURI != "" {
				return true
			}
		}
	}
	return false
}

func isVisionCapableClient(c AIClient) bool {
	switch v := c.(type) {
	case *OpenAIClient, *GeminiClient, *ClaudeClient:
		return true
	case *FailoverClient:
		for _, cli := range v.clients {
			if isVisionCapableClient(cli) {
				return true
			}
		}
		return false
	case *Client:
		return v.Provider == ProviderCustom || v.Provider == ProviderOpenAI
	case *SplitClient:
		return isVisionCapableClient(v.visionClient) || isVisionCapableClient(v.decisionClient)
	default:
		return false
	}
}

// Ensure SplitClient satisfies the AIClient interface.
var _ AIClient = (*SplitClient)(nil)
