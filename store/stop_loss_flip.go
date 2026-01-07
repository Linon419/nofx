package store

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// StopLossFlipStatus defines lifecycle for reverse-on-stop-loss tasks.
type StopLossFlipStatus string

const (
	StopLossFlipArmed     StopLossFlipStatus = "ARMED"
	StopLossFlipTriggered StopLossFlipStatus = "TRIGGERED" // stop-loss closed original position
	StopLossFlipReversed  StopLossFlipStatus = "REVERSED"  // reverse position opened
	StopLossFlipRunner    StopLossFlipStatus = "RUNNER"    // recovery TP executed; trailing runner active
	StopLossFlipDone      StopLossFlipStatus = "DONE"
	StopLossFlipFailed    StopLossFlipStatus = "FAILED"
	StopLossFlipCanceled  StopLossFlipStatus = "CANCELED"
)

// StopLossFlipTask persists state for automatic reverse-on-stop-loss execution.
// All time fields use Unix milliseconds UTC.
type StopLossFlipTask struct {
	ID           int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	TraderID     string `gorm:"column:trader_id;not null;index:idx_slf_trader" json:"trader_id"`
	ExchangeID   string `gorm:"column:exchange_id;not null;default:'';index:idx_slf_exchange" json:"exchange_id"`
	ExchangeType string `gorm:"column:exchange_type;not null;default:''" json:"exchange_type"`

	Symbol string `gorm:"column:symbol;not null;index:idx_slf_symbol" json:"symbol"`
	Side   string `gorm:"column:side;not null" json:"side"` // LONG/SHORT (original)

	EntryPrice    float64 `gorm:"column:entry_price;not null" json:"entry_price"`
	StopLossPrice float64 `gorm:"column:stop_loss_price;not null" json:"stop_loss_price"`
	Quantity      float64 `gorm:"column:quantity;not null" json:"quantity"`
	Leverage      int     `gorm:"column:leverage;default:1" json:"leverage"`

	// Config snapshot
	RunnerRatio float64 `gorm:"column:runner_ratio;default:0.3" json:"runner_ratio"`
	TrailPct    float64 `gorm:"column:trail_pct;default:0" json:"trail_pct"`

	// Trigger (stop-loss close) details
	CloseType      string  `gorm:"column:close_type;default:''" json:"close_type"`
	CloseOrderID   string  `gorm:"column:close_order_id;default:''" json:"close_order_id"`
	CloseTimeMs    int64   `gorm:"column:close_time_ms;default:0;index:idx_slf_close_time" json:"close_time_ms"`
	CloseExitPrice float64 `gorm:"column:close_exit_price;default:0" json:"close_exit_price"`

	// Reverse details
	ReverseSide       string  `gorm:"column:reverse_side;default:''" json:"reverse_side"` // LONG/SHORT
	ReverseOrderID    string  `gorm:"column:reverse_order_id;default:''" json:"reverse_order_id"`
	ReverseEntryPrice float64 `gorm:"column:reverse_entry_price;default:0" json:"reverse_entry_price"`
	RecoveryTarget    float64 `gorm:"column:recovery_target;default:0" json:"recovery_target"`

	// Runner tracking
	LastExtremePrice float64 `gorm:"column:last_extreme_price;default:0" json:"last_extreme_price"`
	CurrentStopPrice float64 `gorm:"column:current_stop_price;default:0" json:"current_stop_price"`

	Status       StopLossFlipStatus `gorm:"column:status;not null;default:ARMED;index:idx_slf_status" json:"status"`
	ErrorMessage string             `gorm:"column:error_message;type:text;default:''" json:"error_message"`

	CreatedAt UnixMilli `gorm:"column:created_at;index:idx_slf_created" json:"created_at"`
	UpdatedAt UnixMilli `gorm:"column:updated_at" json:"updated_at"`
}

func (StopLossFlipTask) TableName() string {
	return "stop_loss_flip_tasks"
}

type StopLossFlipStore struct {
	db *gorm.DB
}

func NewStopLossFlipStore(db *gorm.DB) *StopLossFlipStore {
	return &StopLossFlipStore{db: db}
}

func (s *StopLossFlipStore) initTables() error {
	if s.db == nil {
		return fmt.Errorf("db is nil")
	}
	if err := s.db.AutoMigrate(&StopLossFlipTask{}); err != nil {
		return fmt.Errorf("failed to migrate stop_loss_flip_tasks: %w", err)
	}
	return nil
}

func (s *StopLossFlipStore) Create(task *StopLossFlipTask) error {
	if task == nil {
		return nil
	}
	nowMs := time.Now().UTC().UnixMilli()
	if task.CreatedAt == 0 {
		task.CreatedAt = UnixMilli(nowMs)
	}
	task.UpdatedAt = UnixMilli(nowMs)
	if task.Status == "" {
		task.Status = StopLossFlipArmed
	}
	return s.db.Create(task).Error
}

func (s *StopLossFlipStore) GetLatestArmed(traderID, exchangeID, symbol, side string) (*StopLossFlipTask, error) {
	var t StopLossFlipTask
	err := s.db.Where("trader_id = ? AND exchange_id = ? AND symbol = ? AND side = ? AND status = ?",
		traderID, exchangeID, symbol, side, StopLossFlipArmed).
		Order("created_at DESC").
		First(&t).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (s *StopLossFlipStore) ListActive(traderID, exchangeID string) ([]StopLossFlipTask, error) {
	var tasks []StopLossFlipTask
	err := s.db.Where("trader_id = ? AND exchange_id = ? AND status IN ?",
		traderID, exchangeID, []StopLossFlipStatus{StopLossFlipTriggered, StopLossFlipReversed, StopLossFlipRunner}).
		Order("updated_at DESC").
		Find(&tasks).Error
	return tasks, err
}

func (s *StopLossFlipStore) Update(id int64, updates map[string]interface{}) error {
	if id == 0 || len(updates) == 0 {
		return nil
	}
	updates["updated_at"] = time.Now().UTC().UnixMilli()
	return s.db.Model(&StopLossFlipTask{}).Where("id = ?", id).Updates(updates).Error
}
