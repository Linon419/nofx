package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClaudeClient_CallWithRequest_ToolUseResponse_ReturnsInputJSON(t *testing.T) {
	// Create a mock server that returns tool_use response
	mockResponse := `{
  "content": [
    {
      "type": "tool_use",
      "id": "toolu_test",
      "name": "submit_decisions",
      "input": {
        "reasoning": "r",
        "decisions": [
          {"symbol":"ALL","action":"wait","reasoning":"x"}
        ]
      }
    }
  ],
  "usage": { "input_tokens": 1, "output_tokens": 1 },
  "id": "msg_test",
  "type": "message",
  "role": "assistant",
  "model": "claude-opus-4-5-20251101",
  "stop_reason": "tool_use"
}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	mockLogger := NewMockLogger()

	// Create client with SDK pointing to mock server
	client := NewClaudeClientWithOptions(
		WithLogger(mockLogger),
		WithAPIKey("sk-test-key"),
		WithBaseURL(server.URL), // Override to mock server
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
}
