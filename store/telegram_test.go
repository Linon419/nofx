package store

import (
	"path/filepath"
	"testing"
)

func TestTelegramConfigUpsertPersistsEnabledAndChatID(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("New store: %v", err)
	}
	defer st.db.Close()

	userID := "u_test"
	if err := st.Telegram().Upsert(userID, true, "bot_token_123", "chat_456", true, false, true); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	cfg, err := st.Telegram().Get(userID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if cfg == nil {
		t.Fatalf("expected config, got nil")
	}
	if !cfg.Enabled {
		t.Fatalf("expected enabled=true, got false")
	}
	if cfg.ChatID != "chat_456" {
		t.Fatalf("expected chat_id=chat_456, got %q", cfg.ChatID)
	}
	if cfg.BotToken != "bot_token_123" {
		t.Fatalf("expected bot_token to roundtrip, got %q", cfg.BotToken)
	}
	if !cfg.NotifyOnOpen || cfg.NotifyOnClose || !cfg.NotifyOnError {
		t.Fatalf("unexpected notify flags: open=%v close=%v error=%v", cfg.NotifyOnOpen, cfg.NotifyOnClose, cfg.NotifyOnError)
	}
}
