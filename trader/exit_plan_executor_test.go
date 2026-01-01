package trader

import "testing"

func TestTierTriggered(t *testing.T) {
	if !tierTriggered(true, 105, 100) {
		t.Fatalf("expected long tier to trigger when price >= target")
	}
	if tierTriggered(true, 95, 100) {
		t.Fatalf("expected long tier not to trigger when price < target")
	}
	if !tierTriggered(false, 95, 100) {
		t.Fatalf("expected short tier to trigger when price <= target")
	}
	if tierTriggered(false, 105, 100) {
		t.Fatalf("expected short tier not to trigger when price > target")
	}
}

func TestResolveATRTargetPrice(t *testing.T) {
	params := &exitPlanAtrParams{
		ATRValue:          10,
		TriggerMultiplier: 2,
		TrailMultiplier:   1,
	}
	longTarget, err := resolveATRTargetPrice("long", 100, "BTCUSDT", params)
	if err != nil {
		t.Fatalf("resolveATRTargetPrice long error: %v", err)
	}
	if longTarget != 120 {
		t.Fatalf("unexpected long target: %.2f", longTarget)
	}

	shortTarget, err := resolveATRTargetPrice("short", 100, "BTCUSDT", params)
	if err != nil {
		t.Fatalf("resolveATRTargetPrice short error: %v", err)
	}
	if shortTarget != 80 {
		t.Fatalf("unexpected short target: %.2f", shortTarget)
	}
}
