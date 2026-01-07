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
	if requestHasImageParts(req) && c.visionClient != nil {
		// Always try the configured vision client first when image parts are present.
		// If it fails (unsupported payload/model, auth, etc.), fall back to the decision client.
		out, err := c.visionClient.CallWithRequest(req)
		if err == nil {
			return out, nil
		}

		// If the decision client can't handle image parts, fall back to a text-only request.
		if !isVisionCapableClient(c.decisionClient) {
			return c.decisionClient.CallWithRequest(stripImageParts(req))
		}
		return c.decisionClient.CallWithRequest(req)
	}
	return c.decisionClient.CallWithRequest(req)
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
	case *QwenClient, *KimiClient, *GrokClient:
		// These providers use OpenAI-compatible request payloads (image_url parts).
		return true
	case *FailoverClient:
		for _, cli := range v.clients {
			if isVisionCapableClient(cli) {
				return true
			}
		}
		return false
	case *Client:
		return v.Provider == ProviderCustom ||
			v.Provider == ProviderOpenAI ||
			v.Provider == ProviderQwen ||
			v.Provider == ProviderKimi ||
			v.Provider == ProviderGrok
	case *SplitClient:
		return isVisionCapableClient(v.visionClient) || isVisionCapableClient(v.decisionClient)
	default:
		return false
	}
}

func stripImageParts(req *Request) *Request {
	if req == nil {
		return nil
	}

	out := *req
	if len(req.Messages) == 0 {
		return &out
	}

	msgs := make([]Message, 0, len(req.Messages))
	for _, m := range req.Messages {
		if len(m.Parts) == 0 {
			msgs = append(msgs, m)
			continue
		}

		parts := make([]ContentPart, 0, len(m.Parts))
		for _, p := range m.Parts {
			if p.Type == "text" && p.Text != "" {
				parts = append(parts, p)
			}
		}

		clone := m
		clone.Parts = parts
		msgs = append(msgs, clone)
	}

	out.Messages = msgs
	return &out
}

// Ensure SplitClient satisfies the AIClient interface.
var _ AIClient = (*SplitClient)(nil)
