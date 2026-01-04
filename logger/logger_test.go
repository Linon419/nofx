package logger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPruneNofxLogFiles_KeepFour(t *testing.T) {
	dir := t.TempDir()

	// Create 6 daily log files, plus an unrelated log file.
	names := []string{
		"nofx_2026-01-01.log",
		"nofx_2026-01-02.log",
		"nofx_2026-01-03.log",
		"nofx_2026-01-04.log",
		"nofx_2026-01-05.log",
		"nofx_2026-01-06.log",
		"other.log",
	}
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	if err := pruneNofxLogFiles(dir, 4, "nofx_2026-01-06.log"); err != nil {
		t.Fatalf("prune: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	got := make(map[string]bool, len(entries))
	for _, e := range entries {
		got[e.Name()] = true
	}

	// Should keep the latest 4 nofx logs (01-03..01-06), plus unrelated files.
	wantPresent := []string{
		"nofx_2026-01-03.log",
		"nofx_2026-01-04.log",
		"nofx_2026-01-05.log",
		"nofx_2026-01-06.log",
		"other.log",
	}
	for _, name := range wantPresent {
		if !got[name] {
			t.Fatalf("expected %s to remain", name)
		}
	}
	wantGone := []string{
		"nofx_2026-01-01.log",
		"nofx_2026-01-02.log",
	}
	for _, name := range wantGone {
		if got[name] {
			t.Fatalf("expected %s to be removed", name)
		}
	}
}

