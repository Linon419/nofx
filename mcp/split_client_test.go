package mcp

import (
	"testing"
)

func TestSplitClient_CallWithMessages_UsesDecisionClient(t *testing.T) {
	decisionHTTP := NewMockHTTPClient()
	decisionHTTP.SetSuccessResponse("decision-ok")
	decision := NewDeepSeekClientWithOptions(WithHTTPClient(decisionHTTP.ToHTTPClient()))
	decision.SetAPIKey("sk-decision", "", "")

	visionHTTP := NewMockHTTPClient()
	visionHTTP.SetSuccessResponse("vision-ok")
	vision := NewOpenAIClientWithOptions(WithHTTPClient(visionHTTP.ToHTTPClient()))
	vision.SetAPIKey("sk-vision", "", "")

	client := NewSplitClient(decision, vision)
	_, err := client.CallWithMessages("sys", "user")
	if err != nil {
		t.Fatalf("CallWithMessages error: %v", err)
	}

	if got := len(decisionHTTP.GetRequests()); got != 1 {
		t.Fatalf("decision client requests = %d, want 1", got)
	}
	if got := len(visionHTTP.GetRequests()); got != 0 {
		t.Fatalf("vision client requests = %d, want 0", got)
	}
}

func TestSplitClient_CallWithRequest_WithImages_UsesVisionClient(t *testing.T) {
	decisionHTTP := NewMockHTTPClient()
	decisionHTTP.SetSuccessResponse("decision-ok")
	decision := NewDeepSeekClientWithOptions(WithHTTPClient(decisionHTTP.ToHTTPClient()))
	decision.SetAPIKey("sk-decision", "", "")

	visionHTTP := NewMockHTTPClient()
	visionHTTP.SetSuccessResponse("vision-ok")
	vision := NewOpenAIClientWithOptions(WithHTTPClient(visionHTTP.ToHTTPClient()))
	vision.SetAPIKey("sk-vision", "", "")

	client := NewSplitClient(decision, vision)

	req := NewRequestBuilder().
		WithSystemPrompt("sys").
		WithUserParts([]ContentPart{
			NewTextPart("hello"),
			NewImagePartDataURI("data:image/png;base64,AAAA"),
		}).
		WithTemperature(0.1).
		WithMaxTokens(10).
		MustBuild()

	_, err := client.CallWithRequest(req)
	if err != nil {
		t.Fatalf("CallWithRequest error: %v", err)
	}

	if got := len(visionHTTP.GetRequests()); got != 1 {
		t.Fatalf("vision client requests = %d, want 1", got)
	}
	if got := len(decisionHTTP.GetRequests()); got != 0 {
		t.Fatalf("decision client requests = %d, want 0", got)
	}
}

func TestSplitClient_CallWithRequest_WithImages_NonVisionClient_FallsBackToDecision(t *testing.T) {
	decisionHTTP := NewMockHTTPClient()
	decisionHTTP.SetSuccessResponse("decision-ok")
	decision := NewDeepSeekClientWithOptions(WithHTTPClient(decisionHTTP.ToHTTPClient()))
	decision.SetAPIKey("sk-decision", "", "")

	visionHTTP := NewMockHTTPClient()
	visionHTTP.SetSuccessResponse("vision-ok")
	// DeepSeek client is not treated as vision-capable by SplitClient; should fall back to decision.
	vision := NewDeepSeekClientWithOptions(WithHTTPClient(visionHTTP.ToHTTPClient()))
	vision.SetAPIKey("sk-vision", "", "")

	client := NewSplitClient(decision, vision)

	req := NewRequestBuilder().
		WithSystemPrompt("sys").
		WithUserParts([]ContentPart{
			NewTextPart("hello"),
			NewImagePartDataURI("data:image/png;base64,AAAA"),
		}).
		MustBuild()

	_, err := client.CallWithRequest(req)
	if err != nil {
		t.Fatalf("CallWithRequest error: %v", err)
	}

	if got := len(decisionHTTP.GetRequests()); got != 1 {
		t.Fatalf("decision client requests = %d, want 1", got)
	}
	if got := len(visionHTTP.GetRequests()); got != 0 {
		t.Fatalf("vision client requests = %d, want 0", got)
	}
}
