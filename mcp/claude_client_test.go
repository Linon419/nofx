package mcp

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestClaudeClient_CallWithRequest_ToolUseResponse_ReturnsInputJSON(t *testing.T) {
	mockHTTP := NewMockHTTPClient()
	mockLogger := NewMockLogger()

	mockHTTP.StatusCode = http.StatusOK
	mockHTTP.Response = `{
  "content": [
    {
      "type": "tool_use",
      "name": "submit_decisions",
      "input": {
        "reasoning": "r",
        "decisions": [
          {"symbol":"ALL","action":"wait","reasoning":"x"}
        ]
      }
    }
  ],
  "usage": { "input_tokens": 1, "output_tokens": 1 }
}`

	client := NewClaudeClientWithOptions(
		WithHTTPClient(mockHTTP.ToHTTPClient()),
		WithLogger(mockLogger),
		WithAPIKey("sk-test-key"),
	).(*ClaudeClient)

	req := NewRequestBuilder().
		WithSystemPrompt("sys").
		WithUserPrompt("user").
		AddFunction("submit_decisions", "Submit decisions", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"decisions": map[string]any{"type": "array"},
			},
			"required": []string{"decisions"},
		}).
		WithToolChoice("required").
		MustBuild()

	out, err := client.CallWithRequest(req)
	if err != nil {
		t.Fatalf("CallWithRequest() error = %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("expected JSON output, got %q: %v", out, err)
	}
	if payload["reasoning"] != "r" {
		t.Fatalf("unexpected reasoning: %+v", payload["reasoning"])
	}
	decisions, ok := payload["decisions"].([]any)
	if !ok || len(decisions) != 1 {
		t.Fatalf("unexpected decisions: %+v", payload["decisions"])
	}

	// Verify request contains tools + forced tool choice
	requests := mockHTTP.GetRequests()
	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}
	var body map[string]any
	_ = json.NewDecoder(requests[0].Body).Decode(&body)
	if _, ok := body["tools"]; !ok {
		t.Fatalf("expected tools in request body: %+v", body)
	}
	if tc, ok := body["tool_choice"].(map[string]any); !ok || tc["type"] != "tool" {
		t.Fatalf("expected Claude tool_choice in request body, got: %+v", body["tool_choice"])
	}
}
