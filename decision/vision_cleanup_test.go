package decision

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPruneOldVisionCyclesInDir_KeepLatestFive(t *testing.T) {
	root := t.TempDir()
	traderDir := filepath.Join(root, "vision_images", "traderA")
	if err := os.MkdirAll(traderDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Create cycles 1..8 plus a non-numeric folder.
	for i := 1; i <= 8; i++ {
		if err := os.MkdirAll(filepath.Join(traderDir, itoa(i)), 0o755); err != nil {
			t.Fatalf("mkdir cycle %d: %v", i, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(traderDir, "misc"), 0o755); err != nil {
		t.Fatalf("mkdir misc: %v", err)
	}

	deleted, err := pruneOldVisionCyclesInDir(traderDir, 5)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if deleted != 3 {
		t.Fatalf("expected deleted=3, got %d", deleted)
	}

	// Expect cycles 4..8 to remain (5 dirs), 1..3 removed; misc untouched.
	for i := 1; i <= 3; i++ {
		if _, err := os.Stat(filepath.Join(traderDir, itoa(i))); !os.IsNotExist(err) {
			t.Fatalf("expected cycle %d to be removed, stat err=%v", i, err)
		}
	}
	for i := 4; i <= 8; i++ {
		if _, err := os.Stat(filepath.Join(traderDir, itoa(i))); err != nil {
			t.Fatalf("expected cycle %d to exist, stat err=%v", i, err)
		}
	}
	if _, err := os.Stat(filepath.Join(traderDir, "misc")); err != nil {
		t.Fatalf("expected misc to exist, stat err=%v", err)
	}
}

func itoa(n int) string {
	// Minimal, avoids strconv import in test file.
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

