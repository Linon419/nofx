package store

import (
	"database/sql"
	"fmt"
	"time"
)

// TelegramConfig per-user telegram notification configuration.
type TelegramConfig struct {
	UserID        string    `json:"user_id"`
	Enabled       bool      `json:"enabled"`
	BotToken      string    `json:"bot_token"`
	ChatID        string    `json:"chat_id"`
	NotifyOnOpen  bool      `json:"notify_on_open"`
	NotifyOnClose bool      `json:"notify_on_close"`
	NotifyOnError bool      `json:"notify_on_error"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type TelegramStore struct {
	db          *sql.DB
	encryptFunc func(string) string
	decryptFunc func(string) string
}

func (s *TelegramStore) encrypt(plaintext string) string {
	if s.encryptFunc != nil {
		return s.encryptFunc(plaintext)
	}
	return plaintext
}

func (s *TelegramStore) decrypt(encrypted string) string {
	if s.decryptFunc != nil {
		return s.decryptFunc(encrypted)
	}
	return encrypted
}

func (s *TelegramStore) initTables() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS telegram_configs (
			user_id TEXT PRIMARY KEY,
			enabled BOOLEAN DEFAULT 0,
			bot_token TEXT DEFAULT '',
			chat_id TEXT DEFAULT '',
			notify_on_open BOOLEAN DEFAULT 1,
			notify_on_close BOOLEAN DEFAULT 1,
			notify_on_error BOOLEAN DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create telegram_configs table: %w", err)
	}

	_, err = s.db.Exec(`
		CREATE TRIGGER IF NOT EXISTS update_telegram_configs_updated_at
		AFTER UPDATE ON telegram_configs
		BEGIN
			UPDATE telegram_configs SET updated_at = CURRENT_TIMESTAMP WHERE user_id = NEW.user_id;
		END
	`)
	if err != nil {
		return fmt.Errorf("failed to create telegram_configs trigger: %w", err)
	}

	return nil
}

func (s *TelegramStore) Get(userID string) (*TelegramConfig, error) {
	var cfg TelegramConfig
	var botToken string
	var createdAt, updatedAt string

	err := s.db.QueryRow(`
		SELECT user_id, enabled, bot_token, chat_id, notify_on_open, notify_on_close, notify_on_error, created_at, updated_at
		FROM telegram_configs WHERE user_id = ? LIMIT 1
	`, userID).Scan(
		&cfg.UserID,
		&cfg.Enabled,
		&botToken,
		&cfg.ChatID,
		&cfg.NotifyOnOpen,
		&cfg.NotifyOnClose,
		&cfg.NotifyOnError,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	cfg.BotToken = s.decrypt(botToken)
	cfg.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	cfg.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
	return &cfg, nil
}

func (s *TelegramStore) Upsert(userID string, enabled bool, botToken string, chatID string, notifyOnOpen, notifyOnClose, notifyOnError bool) error {
	existing, err := s.Get(userID)
	if err != nil {
		return err
	}

	tokenToStore := ""
	if botToken != "" {
		tokenToStore = s.encrypt(botToken)
	} else if existing != nil && existing.BotToken != "" {
		tokenToStore = s.encrypt(existing.BotToken)
	}

	if existing == nil {
		_, err := s.db.Exec(`
			INSERT INTO telegram_configs (user_id, enabled, bot_token, chat_id, notify_on_open, notify_on_close, notify_on_error, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))
		`, userID, enabled, tokenToStore, chatID, notifyOnOpen, notifyOnClose, notifyOnError)
		return err
	}

	_, err = s.db.Exec(`
		UPDATE telegram_configs
		SET enabled = ?, bot_token = ?, chat_id = ?, notify_on_open = ?, notify_on_close = ?, notify_on_error = ?, updated_at = datetime('now')
		WHERE user_id = ?
	`, enabled, tokenToStore, chatID, notifyOnOpen, notifyOnClose, notifyOnError, userID)
	return err
}
