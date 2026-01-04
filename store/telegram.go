package store

import (
	"time"

	"gorm.io/gorm"
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

type telegramConfigDB struct {
	UserID        string    `gorm:"primaryKey;column:user_id" json:"user_id"`
	Enabled       bool      `gorm:"column:enabled;default:false" json:"enabled"`
	BotToken      string    `gorm:"column:bot_token;default:''" json:"bot_token"`
	ChatID        string    `gorm:"column:chat_id;default:''" json:"chat_id"`
	NotifyOnOpen  bool      `gorm:"column:notify_on_open;default:true" json:"notify_on_open"`
	NotifyOnClose bool      `gorm:"column:notify_on_close;default:true" json:"notify_on_close"`
	NotifyOnError bool      `gorm:"column:notify_on_error;default:true" json:"notify_on_error"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (telegramConfigDB) TableName() string { return "telegram_configs" }

type TelegramStore struct {
	db *gorm.DB
}

func NewTelegramStore(db *gorm.DB) *TelegramStore {
	return &TelegramStore{db: db}
}

func (s *TelegramStore) initTables() error {
	return s.db.AutoMigrate(&telegramConfigDB{})
}

func (s *TelegramStore) Get(userID string) (*TelegramConfig, error) {
	var cfg telegramConfigDB
	err := s.db.Where("user_id = ?", userID).First(&cfg).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &TelegramConfig{
		UserID:        cfg.UserID,
		Enabled:       cfg.Enabled,
		BotToken:      cfg.BotToken,
		ChatID:        cfg.ChatID,
		NotifyOnOpen:  cfg.NotifyOnOpen,
		NotifyOnClose: cfg.NotifyOnClose,
		NotifyOnError: cfg.NotifyOnError,
		CreatedAt:     cfg.CreatedAt,
		UpdatedAt:     cfg.UpdatedAt,
	}, nil
}

func (s *TelegramStore) Upsert(userID string, enabled bool, botToken string, chatID string, notifyOnOpen, notifyOnClose, notifyOnError bool) error {
	var existing telegramConfigDB
	err := s.db.Where("user_id = ?", userID).First(&existing).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	tokenToStore := botToken
	chatIDToStore := chatID
	if err == nil {
		if tokenToStore == "" {
			tokenToStore = existing.BotToken
		}
		if chatIDToStore == "" {
			chatIDToStore = existing.ChatID
		}
	}

	if err == nil {
		return s.db.Model(&telegramConfigDB{}).Where("user_id = ?", userID).Updates(map[string]interface{}{
			"enabled":         enabled,
			"bot_token":       tokenToStore,
			"chat_id":         chatIDToStore,
			"notify_on_open":  notifyOnOpen,
			"notify_on_close": notifyOnClose,
			"notify_on_error": notifyOnError,
		}).Error
	}

	next := telegramConfigDB{
		UserID:        userID,
		Enabled:       enabled,
		BotToken:      tokenToStore,
		ChatID:        chatIDToStore,
		NotifyOnOpen:  notifyOnOpen,
		NotifyOnClose: notifyOnClose,
		NotifyOnError: notifyOnError,
	}

	// Use map to ensure zero-values (false) are persisted even when DB defaults are set.
	return s.db.Model(&telegramConfigDB{}).Create(map[string]interface{}{
		"user_id":         next.UserID,
		"enabled":         next.Enabled,
		"bot_token":       next.BotToken,
		"chat_id":         next.ChatID,
		"notify_on_open":  next.NotifyOnOpen,
		"notify_on_close": next.NotifyOnClose,
		"notify_on_error": next.NotifyOnError,
	}).Error
}
