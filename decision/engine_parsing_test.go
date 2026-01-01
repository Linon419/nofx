package decision

import "testing"

func TestExtractDecisions_AllowsTildeAndReasonAlias(t *testing.T) {
	aiResponse := "<reasoning>\nTest\n</reasoning>\n\n<decision>\n```json\n[\n" +
		"  {\"symbol\":\"BTCUSDT\",\"action\":\"wait\",\"confidence\":45," +
		"\"reason\":\"Wait for pullback to ~88336 before entry.\"}\n" +
		"]\n```\n</decision>\n"

	decisions, err := extractDecisions(aiResponse)
	if err != nil {
		t.Fatalf("extractDecisions returned error: %v", err)
	}
	if len(decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(decisions))
	}
	if decisions[0].Symbol != "BTCUSDT" {
		t.Fatalf("expected symbol BTCUSDT, got %q", decisions[0].Symbol)
	}
	if decisions[0].Action != "wait" {
		t.Fatalf("expected action wait, got %q", decisions[0].Action)
	}
	if decisions[0].Confidence != 45 {
		t.Fatalf("expected confidence 45, got %d", decisions[0].Confidence)
	}
	if decisions[0].Reasoning == "" {
		t.Fatalf("expected reasoning to be populated from reason/reasoning")
	}
}

func TestValidateJSONFormat_DoesNotRejectTildeInStrings(t *testing.T) {
	jsonContent := `[
  {"symbol":"BTCUSDT","action":"wait","confidence":45,"reasoning":"~88336 and 98,000 mentioned in text"}
]`

	if err := validateJSONFormat(jsonContent); err != nil {
		t.Fatalf("validateJSONFormat returned error: %v", err)
	}
}
