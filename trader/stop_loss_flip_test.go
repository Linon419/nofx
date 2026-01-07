package trader

import "testing"

func TestCalcRecoveryTarget(t *testing.T) {
	t.Run("long stop then reverse short", func(t *testing.T) {
		entry := 100.0
		stop := 90.0
		got := calcRecoveryTarget(entry, stop)
		want := 80.0
		if got != want {
			t.Fatalf("got %v want %v", got, want)
		}
	})

	t.Run("short stop then reverse long", func(t *testing.T) {
		entry := 100.0
		stop := 110.0
		got := calcRecoveryTarget(entry, stop)
		want := 120.0
		if got != want {
			t.Fatalf("got %v want %v", got, want)
		}
	})
}
