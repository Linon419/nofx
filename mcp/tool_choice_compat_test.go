package mcp

import (
	"encoding/json"
	"testing"
)

func TestDeepSeek_CallWithRequest_ToolChoiceRequiredConverted(t *testing.T) {
	mockHTTP := NewMockHTTPClient()
	mockHTTP.SetSuccessResponse("ok")
	mockLogger := NewMockLogger()

	client := NewDeepSeekClientWithOptions(
		WithHTTPClient(mockHTTP.ToHTTPClient()),
		WithLogger(mockLogger),
		WithAPIKey("sk-test-key"),
	)

	req := NewRequestBuilder().
		WithUserPrompt("Decision").
		AddFunction("submit_decisions", "Submit decisions", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"decisions": map[string]any{"type": "array"},
			},
			"required": []string{"decisions"},
		}).
		WithToolChoice("required").
		MustBuild()

	_, err := client.CallWithRequest(req)
	if err != nil {
		t.Fatalf("CallWithRequest() error = %v", err)
	}

	requests := mockHTTP.GetRequests()
	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}

	var body map[string]any
	if err := json.NewDecoder(requests[0].Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}

	toolChoice, ok := body["tool_choice"].(map[string]any)
	if !ok {
		t.Fatalf("expected tool_choice to be object, got %T", body["tool_choice"])
	}

	if toolChoice["type"] != "function" {
		t.Fatalf("expected tool_choice.type=function, got %v", toolChoice["type"])
	}

	fn, ok := toolChoice["function"].(map[string]any)
	if !ok {
		t.Fatalf("expected tool_choice.function to be object, got %T", toolChoice["function"])
	}

	if fn["name"] != "submit_decisions" {
		t.Fatalf("expected tool_choice.function.name=submit_decisions, got %v", fn["name"])
	}
}

func TestGemini_CallWithRequest_ToolChoiceRequiredConverted(t *testing.T) {
	mockHTTP := NewMockHTTPClient()
	mockHTTP.SetSuccessResponse("ok")
	mockLogger := NewMockLogger()

	client := NewGeminiClientWithOptions(
		WithHTTPClient(mockHTTP.ToHTTPClient()),
		WithLogger(mockLogger),
		WithAPIKey("sk-test-key"),
	)

	req := NewRequestBuilder().
		WithUserPrompt("Decision").
		AddFunction("submit_decisions", "Submit decisions", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"decisions": map[string]any{"type": "array"},
			},
			"required": []string{"decisions"},
		}).
		WithToolChoice("required").
		MustBuild()

	_, err := client.CallWithRequest(req)
	if err != nil {
		t.Fatalf("CallWithRequest() error = %v", err)
	}

	requests := mockHTTP.GetRequests()
	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}

	var body map[string]any
	if err := json.NewDecoder(requests[0].Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}

	toolChoice, ok := body["tool_choice"].(map[string]any)
	if !ok {
		t.Fatalf("expected tool_choice to be object, got %T", body["tool_choice"])
	}

	if toolChoice["type"] != "function" {
		t.Fatalf("expected tool_choice.type=function, got %v", toolChoice["type"])
	}

	fn, ok := toolChoice["function"].(map[string]any)
	if !ok {
		t.Fatalf("expected tool_choice.function to be object, got %T", toolChoice["function"])
	}

	if fn["name"] != "submit_decisions" {
		t.Fatalf("expected tool_choice.function.name=submit_decisions, got %v", fn["name"])
	}
}

