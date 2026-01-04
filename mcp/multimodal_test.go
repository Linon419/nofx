package mcp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRequestBuilder_WithUserParts(t *testing.T) {
	req := NewRequestBuilder().
		WithModel("test-model").
		WithSystemPrompt("sys").
		WithUserParts([]ContentPart{
			NewTextPart("hello"),
			NewImagePartDataURI("data:image/png;base64,AAAA"),
		}).
		MustBuild()

	if len(req.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(req.Messages))
	}
	if req.Messages[1].Role != "user" {
		t.Fatalf("expected user message, got %s", req.Messages[1].Role)
	}
	if len(req.Messages[1].Parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(req.Messages[1].Parts))
	}
}

func TestClient_buildRequestBodyFromRequest_MultimodalOpenAI(t *testing.T) {
	client := NewClient(WithProvider(ProviderOpenAI), WithModel("gpt-test")).(*Client)

	req := NewRequestBuilder().
		WithModel("gpt-test").
		WithUserParts([]ContentPart{
			NewTextPart("hello"),
			NewImagePartDataURI("data:image/png;base64,AAAA"),
		}).
		WithMaxTokens(123).
		MustBuild()

	body := client.buildRequestBodyFromRequest(req)
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, `"type":"image_url"`) || !strings.Contains(s, `"data:image/png;base64,AAAA"`) {
		t.Fatalf("expected image_url content in request body, got: %s", s)
	}
	if !strings.Contains(s, `"max_completion_tokens":123`) {
		t.Fatalf("expected max_completion_tokens for openai provider, got: %s", s)
	}
}
