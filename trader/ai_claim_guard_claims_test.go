package trader

import "testing"

func TestHasAssertedKeyword_NegationAndHedge(t *testing.T) {
	r := "4h no double_top; 15m double_bottom breakout"
	if hasAssertedKeyword(r, []string{"double_top"}) {
		t.Fatalf("expected double_top to be treated as negated")
	}
	if !hasAssertedKeyword(r, []string{"double_bottom"}) {
		t.Fatalf("expected double_bottom to be treated as asserted")
	}
	tf := extractTimeframeNearAssertedKeyword(r, []string{"double_bottom"})
	if tf != "15m" {
		t.Fatalf("expected timeframe 15m, got %q", tf)
	}
}

func TestHasAssertedKeyword_ChineseNegation(t *testing.T) {
	r := "4h 没有双顶；15m 双底形成"
	if hasAssertedKeyword(r, []string{"双顶"}) {
		t.Fatalf("expected 双顶 to be treated as negated")
	}
	if !hasAssertedKeyword(r, []string{"双底"}) {
		t.Fatalf("expected 双底 to be treated as asserted")
	}
	tf := extractTimeframeNearAssertedKeyword(r, []string{"双底"})
	if tf != "15m" {
		t.Fatalf("expected timeframe 15m, got %q", tf)
	}
}

func TestHasAssertedKeyword_QuestionMarkIsHedge(t *testing.T) {
	r := "15m double_top?"
	if hasAssertedKeyword(r, []string{"double_top"}) {
		t.Fatalf("expected question-mark mention to be treated as hedge")
	}
}
