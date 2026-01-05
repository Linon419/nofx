package trader

import "testing"

func TestCalcRiskRewardRatio_Long(t *testing.T) {
	rr := calcRiskRewardRatio("open_long", 17.704, 16, 18.5)
	if rr <= 0 {
		t.Fatalf("expected rr > 0, got %v", rr)
	}
	// (18.5-17.704)/(17.704-16) ~= 0.467
	if rr < 0.45 || rr > 0.49 {
		t.Fatalf("expected rr ~0.47, got %v", rr)
	}
}

func TestCalcRiskRewardRatio_Short(t *testing.T) {
	rr := calcRiskRewardRatio("open_short", 100, 105, 94)
	// risk=5, reward=6 => 1.2
	if rr < 1.19 || rr > 1.21 {
		t.Fatalf("expected rr ~1.2, got %v", rr)
	}
}

func TestCalcRiskRewardRatio_Invalid(t *testing.T) {
	if rr := calcRiskRewardRatio("open_long", 100, 0, 110); rr != 0 {
		t.Fatalf("expected rr 0 for invalid inputs, got %v", rr)
	}
	if rr := calcRiskRewardRatio("hold", 100, 90, 110); rr != 0 {
		t.Fatalf("expected rr 0 for non-open action, got %v", rr)
	}
}

