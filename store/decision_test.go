package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestDecisionLogDecision_VisionImagesNilStoredAsEmptyArray(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("New store: %v", err)
	}
	defer st.db.Close()

	rec := &DecisionRecord{
		TraderID:       "t_test",
		CycleNumber:    1,
		Timestamp:      time.Now().UTC(),
		CandidateCoins: []string{},
		ExecutionLog:   []string{},
		Decisions:      []DecisionAction{},
		VisionImages:   nil,
		Success:        true,
	}

	if err := st.Decision().LogDecision(rec); err != nil {
		t.Fatalf("LogDecision: %v", err)
	}

	var dbRec DecisionRecordDB
	if err := st.gdb.Where("id = ?", rec.ID).First(&dbRec).Error; err != nil {
		t.Fatalf("load db record: %v", err)
	}
	if dbRec.VisionImages != "[]" {
		t.Fatalf("expected vision_images to be '[]', got %q", dbRec.VisionImages)
	}
}
