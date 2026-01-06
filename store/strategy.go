package store

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// StrategyStore strategy storage
type StrategyStore struct {
	db *gorm.DB
}

// Strategy strategy configuration
type Strategy struct {
	ID            string    `gorm:"primaryKey" json:"id"`
	UserID        string    `gorm:"column:user_id;not null;default:'';index" json:"user_id"`
	Name          string    `gorm:"not null" json:"name"`
	Description   string    `gorm:"default:''" json:"description"`
	IsActive      bool      `gorm:"column:is_active;default:false;index" json:"is_active"`
	IsDefault     bool      `gorm:"column:is_default;default:false" json:"is_default"`
	IsPublic      bool      `gorm:"column:is_public;default:false;index" json:"is_public"`    // whether visible in strategy market
	ConfigVisible bool      `gorm:"column:config_visible;default:true" json:"config_visible"` // whether config details are visible
	Config        string    `gorm:"not null;default:'{}'" json:"config"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (Strategy) TableName() string { return "strategies" }

// StrategyConfig strategy configuration details (JSON structure)
type StrategyConfig struct {
	// language setting: "zh" for Chinese, "en" for English
	// This determines the language used for data formatting and prompt generation
	Language string `json:"language,omitempty"`
	// coin source configuration
	CoinSource CoinSourceConfig `json:"coin_source"`
	// quantitative data configuration
	Indicators IndicatorConfig `json:"indicators"`
	// vision configuration (optional, v1: per-symbol chart reading)
	Vision VisionConfig `json:"vision,omitempty"`
	// custom prompt (appended at the end)
	CustomPrompt string `json:"custom_prompt,omitempty"`
	// risk control configuration
	RiskControl RiskControlConfig `json:"risk_control"`
	// editable sections of System Prompt
	PromptSections PromptSectionsConfig `json:"prompt_sections,omitempty"`
}

// VisionConfig controls chart rendering + multimodal sending for vision-capable models.
// Default is disabled for backward compatibility.
type VisionIndicatorsConfig struct {
	// ShowEMA controls whether EMA overlays are drawn on the price chart.
	ShowEMA *bool `json:"show_ema,omitempty"`
	// ShowMACD controls whether a MACD panel is rendered.
	ShowMACD *bool `json:"show_macd,omitempty"`
	// ShowCVD controls whether a CVD (cumulative volume delta) panel is rendered.
	ShowCVD *bool `json:"show_cvd,omitempty"`
	// ShowWaveTrend controls whether a WaveTrend panel is rendered.
	ShowWaveTrend *bool `json:"show_wavetrend,omitempty"`
	// ShowSqueeze controls whether squeeze state is annotated.
	ShowSqueeze *bool `json:"show_squeeze,omitempty"`
	// ShowDivergence controls whether divergence summary is annotated.
	ShowDivergence *bool `json:"show_divergence,omitempty"`
}

var (
	boolTrue  = true
	boolFalse = false
)

type VisionConfig struct {
	Enabled bool `json:"enabled"`
	// MaxSymbols limits how many candidate symbols will be analyzed with charts per cycle.
	// Defaults to 5 when Enabled.
	MaxSymbols int `json:"max_symbols,omitempty"`
	// Timeframes are the chart timeframes to render per symbol.
	// Defaults to ["1h","15m"] when Enabled.
	Timeframes []string `json:"timeframes,omitempty"`
	// ImageSize controls chart PNG size. Defaults to 1024x640 when Enabled.
	ImageWidth  int `json:"image_width,omitempty"`
	ImageHeight int `json:"image_height,omitempty"`
	// RenderConcurrency limits concurrent chart rendering jobs. Defaults to 1 when Enabled.
	RenderConcurrency int `json:"render_concurrency,omitempty"`
	// Indicators controls which overlays/panels are included in vision PNGs.
	Indicators VisionIndicatorsConfig `json:"indicators,omitempty"`
}

// PromptSectionsConfig editable sections of System Prompt
type PromptSectionsConfig struct {
	// role definition (title + description)
	RoleDefinition string `json:"role_definition,omitempty"`
	// trading frequency awareness
	TradingFrequency string `json:"trading_frequency,omitempty"`
	// entry standards
	EntryStandards string `json:"entry_standards,omitempty"`
	// decision process
	DecisionProcess string `json:"decision_process,omitempty"`
	// exit strategy plan (LLM output schema)
	ExitStrategyPlan string `json:"exit_strategy_plan,omitempty"`
	// recent closed trades to include in User Prompt context
	RecentTradesLimit int `json:"recent_trades_limit,omitempty"`
}

// CoinSourceConfig coin source configuration
type CoinSourceConfig struct {
	// source type: "static" | "ai500" | "coinpool"(legacy alias) | "oi_top" | "otc_top" | "mixed"
	SourceType string `json:"source_type"`
	// static coin list (used when source_type = "static")
	StaticCoins []string `json:"static_coins,omitempty"`
	// excluded coins list (filtered out from all sources)
	ExcludedCoins []string `json:"excluded_coins,omitempty"`
	// whether to use AI500 coin pool
	UseAI500 bool `json:"use_ai500"`
	// AI500 coin pool maximum count
	AI500Limit int `json:"ai500_limit,omitempty"`
	// whether to use OI Top
	UseOITop bool `json:"use_oi_top"`
	// OI Top maximum count
	OITopLimit int `json:"oi_top_limit,omitempty"`
	// Note: AI500/OI Top API URLs are now built automatically using NofxOSAPIKey from IndicatorConfig.
	// The fields below are kept for backward compatibility with older configs.
	OITopAPIURL  string `json:"oi_top_api_url,omitempty"`
	UseOTCTop    bool   `json:"use_otc_top"`
	OTCTopAPIURL string `json:"otc_top_api_url,omitempty"`
}

// IndicatorConfig indicator configuration
type IndicatorConfig struct {
	// K-line configuration
	Klines KlineConfig `json:"klines"`
	// raw kline data (OHLCV) - always enabled, required for AI analysis
	EnableRawKlines bool `json:"enable_raw_klines"`
	// technical indicator switches
	EnableEMA         bool `json:"enable_ema"`
	EnableMACD        bool `json:"enable_macd"`
	EnableRSI         bool `json:"enable_rsi"`
	EnableATR         bool `json:"enable_atr"`
	EnableBOLL        bool `json:"enable_boll"` // Bollinger Bands
	EnableVolume      bool `json:"enable_volume"`
	EnableOI          bool `json:"enable_oi"`           // open interest
	EnableFundingRate bool `json:"enable_funding_rate"` // funding rate
	// EMA period configuration
	EMAPeriods []int `json:"ema_periods,omitempty"` // default [21, 55, 100, 200]
	// RSI period configuration
	RSIPeriods []int `json:"rsi_periods,omitempty"` // default [7, 14]
	// ATR period configuration
	ATRPeriods []int `json:"atr_periods,omitempty"` // default [14]
	// BOLL period configuration (period, standard deviation multiplier is fixed at 2)
	BOLLPeriods []int `json:"boll_periods,omitempty"` // default [20] - can select multiple timeframes
	// external data sources
	ExternalDataSources []ExternalDataSource `json:"external_data_sources,omitempty"`

	// ========== NofxOS Unified API Configuration ==========
	// Unified API Key for all NofxOS data sources
	NofxOSAPIKey string `json:"nofxos_api_key,omitempty"`

	// quantitative data sources (capital flow, position changes, price changes)
	EnableQuantData    bool `json:"enable_quant_data"`    // whether to enable quantitative data
	EnableQuantOI      bool `json:"enable_quant_oi"`      // whether to show OI data
	EnableQuantNetflow bool `json:"enable_quant_netflow"` // whether to show Netflow data

	// OI ranking data (market-wide open interest increase/decrease rankings)
	EnableOIRanking   bool   `json:"enable_oi_ranking"`             // whether to enable OI ranking data
	OIRankingDuration string `json:"oi_ranking_duration,omitempty"` // duration: 1h, 4h, 24h
	OIRankingLimit    int    `json:"oi_ranking_limit,omitempty"`    // number of entries (default 10)

	// NetFlow ranking data (market-wide fund flow rankings - institution/personal)
	EnableNetFlowRanking   bool   `json:"enable_netflow_ranking"`             // whether to enable NetFlow ranking data
	NetFlowRankingDuration string `json:"netflow_ranking_duration,omitempty"` // duration: 1h, 4h, 24h
	NetFlowRankingLimit    int    `json:"netflow_ranking_limit,omitempty"`    // number of entries (default 10)

	// Price ranking data (market-wide gainers/losers)
	EnablePriceRanking   bool   `json:"enable_price_ranking"`             // whether to enable price ranking data
	PriceRankingDuration string `json:"price_ranking_duration,omitempty"` // durations: "1h" or "1h,4h,24h"
	PriceRankingLimit    int    `json:"price_ranking_limit,omitempty"`    // number of entries per ranking (default 10)
}

// KlineConfig K-line configuration
type KlineConfig struct {
	// primary timeframe: "1m", "3m", "5m", "15m", "1h", "4h"
	PrimaryTimeframe string `json:"primary_timeframe"`
	// primary timeframe K-line count
	PrimaryCount int `json:"primary_count"`
	// longer timeframe
	LongerTimeframe string `json:"longer_timeframe,omitempty"`
	// longer timeframe K-line count
	LongerCount int `json:"longer_count,omitempty"`
	// whether to enable multi-timeframe analysis
	EnableMultiTimeframe bool `json:"enable_multi_timeframe"`
	// selected timeframe list (new: supports multi-timeframe selection)
	SelectedTimeframes []string `json:"selected_timeframes,omitempty"`
	// decision schedule: aligned to min timeframe close
	DecisionIntervalMultiple int  `json:"decision_interval_multiple,omitempty"`
	DecisionOffsetSeconds    int  `json:"decision_offset_seconds,omitempty"`
	DecisionRunImmediately   bool `json:"decision_run_immediately,omitempty"`
}

// ExternalDataSource external data source configuration
type ExternalDataSource struct {
	Name        string            `json:"name"`   // data source name
	Type        string            `json:"type"`   // type: "api" | "webhook"
	URL         string            `json:"url"`    // API URL
	Method      string            `json:"method"` // HTTP method
	Headers     map[string]string `json:"headers,omitempty"`
	DataPath    string            `json:"data_path,omitempty"`    // JSON data path
	RefreshSecs int               `json:"refresh_secs,omitempty"` // refresh interval (seconds)
}

// RiskControlConfig risk control configuration
// All parameters are clearly defined without ambiguity:
//
// Position Limits:
//   - MaxPositions: max number of coins held simultaneously (CODE ENFORCED)
//
// Trading Leverage (exchange leverage for opening positions):
//   - BTCETHMaxLeverage: BTC/ETH max exchange leverage (AI guided)
//   - AltcoinMaxLeverage: Altcoin max exchange leverage (AI guided)
//   - ATREnabled: enable ATR-based leverage + stop-loss sizing
//   - ATRPeriod: ATR period for leverage calc
//   - ATRTimeframe: timeframe used for ATR
//   - StopLossRiskPct: risk per trade (% of equity) for stop-loss sizing
//
// Position Value Limits (single position notional value / account equity):
//   - BTCETHMaxPositionValueRatio: BTC/ETH max = equity × ratio (CODE ENFORCED)
//   - AltcoinMaxPositionValueRatio: Altcoin max = equity × ratio (CODE ENFORCED)
//
// Risk Controls:
//   - MaxMarginUsage: max margin utilization percentage (CODE ENFORCED)
//   - MinPositionSize: minimum position size in USDT (CODE ENFORCED)
//   - MinRiskRewardRatio: min take_profit / stop_loss ratio (AI guided)
//   - MinConfidence: min AI confidence to open position (AI guided)
type RiskControlConfig struct {
	// Max number of coins held simultaneously (CODE ENFORCED)
	MaxPositions int `json:"max_positions"`

	// EnforceAICloseGuard blocks AI-initiated close decisions unless hard exit conditions trigger.
	// Hard conditions are evaluated in execution layer (trader), independent of prompt.
	// nil = default true (opt-out)
	EnforceAICloseGuard *bool `json:"enforce_ai_close_guard,omitempty"`

	// EnforceAIClaimGuard blocks AI-initiated opens when the decision reasoning references
	// data that is not present in the current prompt cycle (anti-hallucination guard).
	// nil = default false (opt-in)
	EnforceAIClaimGuard *bool `json:"enforce_ai_claim_guard,omitempty"`

	// BTC/ETH exchange leverage for opening positions (AI guided)
	BTCETHMaxLeverage int `json:"btc_eth_max_leverage"`
	// Altcoin exchange leverage for opening positions (AI guided)
	AltcoinMaxLeverage int `json:"altcoin_max_leverage"`

	// ATR-based leverage and stop-loss sizing
	ATREnabled      bool    `json:"atr_enabled"`
	ATRPeriod       int     `json:"atr_period"`
	ATRTimeframe    string  `json:"atr_timeframe"`
	StopLossRiskPct float64 `json:"stop_loss_risk_pct"`

	// Stop-loss-based position sizing (CODE ENFORCED).
	// When enabled (and ATR leverage is disabled), the system overrides AI-provided position_size_usd
	// using: position_size_usd = equity * stop_loss_risk_pct% * leverage / stop_distance_pct.
	StopLossSizingEnabled bool `json:"stop_loss_sizing_enabled,omitempty"`

	// BTC/ETH single position max value = equity × this ratio (CODE ENFORCED, default: 5)
	BTCETHMaxPositionValueRatio float64 `json:"btc_eth_max_position_value_ratio"`
	// Altcoin single position max value = equity × this ratio (CODE ENFORCED, default: 1)
	AltcoinMaxPositionValueRatio float64 `json:"altcoin_max_position_value_ratio"`

	// Max margin utilization (e.g. 0.9 = 90%) (CODE ENFORCED)
	MaxMarginUsage float64 `json:"max_margin_usage"`
	// Min position size in USDT (CODE ENFORCED)
	MinPositionSize float64 `json:"min_position_size"`
	// Minimum open interest notional value (in millions of USD) required for a candidate coin to be tradable.
	//  - nil: use default (15M)
	//  - <= 0: disable the OI liquidity filter
	MinOpenInterestValueMillions *float64 `json:"min_open_interest_value_millions,omitempty"`
	// Enforce minimum order notional checks (CODE ENFORCED, default: true)
	// When disabled, the system may attempt very small orders that could be rejected by the exchange.
	EnforceMinPositionSize *bool `json:"enforce_min_position_size,omitempty"`

	// Min take_profit / stop_loss ratio (AI guided)
	MinRiskRewardRatio float64 `json:"min_risk_reward_ratio"`
	// Min AI confidence to open position (AI guided)
	MinConfidence int `json:"min_confidence"`
}

// NewStrategyStore creates a new StrategyStore
func NewStrategyStore(db *gorm.DB) *StrategyStore {
	return &StrategyStore{db: db}
}

func (s *StrategyStore) initTables() error {
	// SQLite: avoid GORM AutoMigrate table-rebuild behavior (can fail on existing data)
	if s.db.Dialector.Name() == "sqlite" {
		if s.db.Migrator().HasTable(&Strategy{}) {
			for _, field := range []string{
				"UserID",
				"Name",
				"Description",
				"IsActive",
				"IsDefault",
				"IsPublic",
				"ConfigVisible",
				"Config",
				"CreatedAt",
				"UpdatedAt",
			} {
				if !s.db.Migrator().HasColumn(&Strategy{}, field) {
					_ = s.db.Migrator().AddColumn(&Strategy{}, field)
				}
			}

			s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_strategies_user_id ON strategies(user_id)`)
			s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_strategies_is_active ON strategies(is_active)`)
			s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_strategies_is_public ON strategies(is_public)`)

			// Legacy migration: older SQLite DBs may have empty/incorrect user_id for strategies.
			// Fix ownership to avoid mistakenly marking user strategies as system defaults (which makes them read-only in UI).
			//
			// 1) Prefer inferring owner from any trader that references this strategy_id.
			s.db.Exec(`
				UPDATE strategies
				SET user_id = (
					SELECT t.user_id
					FROM traders t
					WHERE t.strategy_id = strategies.id
						AND t.user_id IS NOT NULL
						AND t.user_id != ''
						AND t.user_id != 'default'
					LIMIT 1
				),
				is_default = 0
				WHERE (user_id IS NULL OR user_id = '' OR user_id = 'default')
				  AND EXISTS (
					SELECT 1
					FROM traders t
					WHERE t.strategy_id = strategies.id
						AND t.user_id IS NOT NULL
						AND t.user_id != ''
						AND t.user_id != 'default'
				  )
			`)

			// 2) Fallback: assign remaining legacy strategies to the earliest created user.
			s.db.Exec(`
				UPDATE strategies
				SET user_id = (SELECT id FROM users ORDER BY created_at ASC LIMIT 1),
				    is_default = 0
				WHERE (user_id IS NULL OR user_id = '' OR user_id = 'default')
				  AND (SELECT COUNT(*) FROM users) >= 1
			`)

			// Ensure non-default strategies are not flagged as system defaults.
			s.db.Exec(`UPDATE strategies SET is_default = 0 WHERE user_id != 'default'`)

			s.db.Exec(`UPDATE strategies SET is_public = 0 WHERE is_public IS NULL`)
			s.db.Exec(`UPDATE strategies SET config_visible = 1 WHERE config_visible IS NULL`)
			return nil
		}
	}

	// AutoMigrate will add missing columns without dropping existing data
	return s.db.AutoMigrate(&Strategy{})
}

func (s *StrategyStore) initDefaultData() error {
	// No longer pre-populate strategies - create on demand when user configures
	return nil
}

// GetDefaultStrategyConfig returns the default strategy configuration for the given language
func GetDefaultStrategyConfig(lang string) StrategyConfig {
	// Normalize language to "zh" or "en"
	normalizedLang := "en"
	if lang == "zh" {
		normalizedLang = "zh"
	}
	enforceMinPositionSize := true
	enforceAICloseGuard := true
	enforceAIClaimGuard := false
	minOIMillions := 15.0

	config := StrategyConfig{
		Language: normalizedLang,
		CoinSource: CoinSourceConfig{
			SourceType:   "ai500",
			UseAI500:     true,
			AI500Limit:   10,
			UseOITop:     false,
			OITopLimit:   20,
			UseOTCTop:    false,
			OTCTopAPIURL: "",
		},
		Indicators: IndicatorConfig{
			Klines: KlineConfig{
				PrimaryTimeframe:         "5m",
				PrimaryCount:             30,
				LongerTimeframe:          "4h",
				LongerCount:              10,
				EnableMultiTimeframe:     true,
				SelectedTimeframes:       []string{"5m", "15m", "1h", "4h"},
				DecisionIntervalMultiple: 1,
				DecisionOffsetSeconds:    10,
				DecisionRunImmediately:   false,
			},
			EnableRawKlines:   true, // Required - raw OHLCV data for AI analysis
			EnableEMA:         false,
			EnableMACD:        false,
			EnableRSI:         false,
			EnableATR:         false,
			EnableBOLL:        false,
			EnableVolume:      true,
			EnableOI:          true,
			EnableFundingRate: true,
			EMAPeriods:        []int{21, 55, 100, 200},
			RSIPeriods:        []int{7, 14},
			ATRPeriods:        []int{14},
			BOLLPeriods:       []int{20},
			// NofxOS unified API key
			NofxOSAPIKey: "cm_568c67eae410d912c54c",
			// Quant data
			EnableQuantData:    true,
			EnableQuantOI:      true,
			EnableQuantNetflow: true,
			// OI ranking data
			EnableOIRanking:   true,
			OIRankingDuration: "1h",
			OIRankingLimit:    10,
			// NetFlow ranking data
			EnableNetFlowRanking:   true,
			NetFlowRankingDuration: "1h",
			NetFlowRankingLimit:    10,
			// Price ranking data
			EnablePriceRanking:   true,
			PriceRankingDuration: "1h,4h,24h",
			PriceRankingLimit:    10,
		},
		Vision: VisionConfig{
			Enabled:           false,
			MaxSymbols:        5,
			Timeframes:        []string{"1h", "15m"},
			ImageWidth:        1600,
			ImageHeight:       1396,
			RenderConcurrency: 1,
			Indicators: VisionIndicatorsConfig{
				ShowEMA:        &boolTrue,
				ShowMACD:       &boolTrue,
				ShowCVD:        &boolTrue,
				ShowWaveTrend:  &boolTrue,
				ShowSqueeze:    &boolFalse,
				ShowDivergence: &boolTrue,
			},
		},
		RiskControl: RiskControlConfig{
			MaxPositions:                 3, // Max 3 coins simultaneously (CODE ENFORCED)
			EnforceAICloseGuard:          &enforceAICloseGuard,
			EnforceAIClaimGuard:          &enforceAIClaimGuard,
			BTCETHMaxLeverage:            5, // BTC/ETH exchange leverage (AI guided)
			AltcoinMaxLeverage:           5, // Altcoin exchange leverage (AI guided)
			ATREnabled:                   true,
			ATRPeriod:                    14,
			ATRTimeframe:                 "1d",
			StopLossRiskPct:              5.0,
			StopLossSizingEnabled:        false,
			BTCETHMaxPositionValueRatio:  5.0, // BTC/ETH: max position = 5x equity (CODE ENFORCED)
			AltcoinMaxPositionValueRatio: 1.0, // Altcoin: max position = 1x equity (CODE ENFORCED)
			MaxMarginUsage:               0.9, // Max 90% margin usage (CODE ENFORCED)
			MinPositionSize:              12,  // Min 12 USDT per position (CODE ENFORCED)
			MinOpenInterestValueMillions: &minOIMillions,
			EnforceMinPositionSize:       &enforceMinPositionSize,
			MinRiskRewardRatio:           3.0, // Min 3:1 profit/loss ratio (AI guided)
			MinConfidence:                75,  // Min 75% confidence (AI guided)
		},
	}

	if lang == "zh" {
		config.PromptSections = PromptSectionsConfig{
			RoleDefinition: `# 你是一个专业的加密货币交易AI

你的任务是根据提供的市场数据做出交易决策。你是一个经验丰富的量化交易员，擅长技术分析和风险管理。`,
			TradingFrequency: `# ⏱️ 交易频率意识

- 优秀交易员：每天2-4笔 ≈ 每小时0.1-0.2笔
- 每小时超过2笔 = 过度交易
- 单笔持仓时间 ≥ 30-60分钟
如果你发现自己每个周期都在交易 → 标准太低；如果持仓不到30分钟就平仓 → 太冲动。`,
			EntryStandards: `# 🎯 入场标准（严格）

只在多个信号共振时入场。自由使用任何有效的分析方法，避免单一指标、信号矛盾、横盘震荡、或平仓后立即重新开仓等低质量行为。`,
			DecisionProcess: `# 📋 决策流程

1. 检查持仓 → 是否止盈/止损
2. 扫描候选币种 + 多时间框架 → 是否存在强信号
3. 先写思维链，再输出结构化JSON`,
		}
	} else {
		config.PromptSections = PromptSectionsConfig{
			RoleDefinition: `# You are a professional cryptocurrency trading AI

Your task is to make trading decisions based on the provided market data. You are an experienced quantitative trader skilled in technical analysis and risk management.`,
			TradingFrequency: `# ⏱️ Trading Frequency Awareness

- Excellent trader: 2-4 trades per day ≈ 0.1-0.2 trades per hour
- >2 trades per hour = overtrading
- Single position holding time ≥ 30-60 minutes
If you find yourself trading every cycle → standards are too low; if closing positions in <30 minutes → too impulsive.`,
			EntryStandards: `# 🎯 Entry Standards (Strict)

Only enter positions when multiple signals resonate. Freely use any effective analysis methods, avoid low-quality behaviors such as single indicators, contradictory signals, sideways oscillation, or immediately restarting after closing positions.`,
			DecisionProcess: `# 📋 Decision Process

1. Check positions → whether to take profit/stop loss
2. Scan candidate coins + multi-timeframe → whether strong signals exist
3. Write chain of thought first, then output structured JSON`,
		}
	}

	config.PromptSections.RecentTradesLimit = 3
	if config.PromptSections.ExitStrategyPlan == "" {
		config.PromptSections.ExitStrategyPlan = "plan_tp_tiers_sl_single"
	}

	return config
}

// Create create a strategy
func (s *StrategyStore) Create(strategy *Strategy) error {
	return s.db.Create(strategy).Error
}

// Update update a strategy
func (s *StrategyStore) Update(strategy *Strategy) error {
	return s.db.Model(&Strategy{}).
		Where("id = ? AND user_id = ?", strategy.ID, strategy.UserID).
		Updates(map[string]interface{}{
			"name":           strategy.Name,
			"description":    strategy.Description,
			"config":         strategy.Config,
			"is_public":      strategy.IsPublic,
			"config_visible": strategy.ConfigVisible,
			"updated_at":     time.Now().UTC(),
		}).Error
}

// Delete delete a strategy
func (s *StrategyStore) Delete(userID, id string) error {
	// do not allow deleting system default strategy
	var st Strategy
	if err := s.db.Where("id = ?", id).First(&st).Error; err == nil && st.IsDefault {
		return fmt.Errorf("cannot delete system default strategy")
	}

	return s.db.Where("id = ? AND user_id = ?", id, userID).Delete(&Strategy{}).Error
}

// List get user's strategy list
func (s *StrategyStore) List(userID string) ([]*Strategy, error) {
	var strategies []*Strategy
	err := s.db.Where("user_id = ? OR is_default = ?", userID, true).
		Order("is_default DESC, created_at DESC").
		Find(&strategies).Error
	if err != nil {
		return nil, err
	}
	return strategies, nil
}

// ListPublic get all public strategies for the strategy market
func (s *StrategyStore) ListPublic() ([]*Strategy, error) {
	var strategies []*Strategy
	err := s.db.Where("is_public = ?", true).
		Order("created_at DESC").
		Find(&strategies).Error
	if err != nil {
		return nil, err
	}
	return strategies, nil
}

// Get get a single strategy
func (s *StrategyStore) Get(userID, id string) (*Strategy, error) {
	var st Strategy
	err := s.db.Where("id = ? AND (user_id = ? OR is_default = ?)", id, userID, true).
		First(&st).Error
	if err != nil {
		return nil, err
	}
	return &st, nil
}

// GetActive get user's currently active strategy
func (s *StrategyStore) GetActive(userID string) (*Strategy, error) {
	var st Strategy
	err := s.db.Where("user_id = ? AND is_active = ?", userID, true).First(&st).Error
	if err == gorm.ErrRecordNotFound {
		// no active strategy, return system default strategy
		return s.GetDefault()
	}
	if err != nil {
		return nil, err
	}
	return &st, nil
}

// GetDefault get system default strategy
func (s *StrategyStore) GetDefault() (*Strategy, error) {
	var st Strategy
	err := s.db.Where("is_default = ?", true).First(&st).Error
	if err != nil {
		return nil, err
	}
	return &st, nil
}

// SetActive set active strategy (will first deactivate other strategies)
func (s *StrategyStore) SetActive(userID, strategyID string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// first deactivate all strategies for the user
		if err := tx.Model(&Strategy{}).Where("user_id = ?", userID).
			Update("is_active", false).Error; err != nil {
			return err
		}

		// activate specified strategy
		return tx.Model(&Strategy{}).
			Where("id = ? AND (user_id = ? OR is_default = ?)", strategyID, userID, true).
			Update("is_active", true).Error
	})
}

// Duplicate duplicate a strategy (used to create custom strategy based on default strategy)
func (s *StrategyStore) Duplicate(userID, sourceID, newID, newName string) error {
	// get source strategy
	source, err := s.Get(userID, sourceID)
	if err != nil {
		return fmt.Errorf("failed to get source strategy: %w", err)
	}

	// create new strategy
	newStrategy := &Strategy{
		ID:          newID,
		UserID:      userID,
		Name:        newName,
		Description: "Created based on [" + source.Name + "]",
		IsActive:    false,
		IsDefault:   false,
		Config:      source.Config,
	}

	return s.Create(newStrategy)
}

// ParseConfig parse strategy configuration JSON
func (s *Strategy) ParseConfig() (*StrategyConfig, error) {
	var config StrategyConfig
	if err := json.Unmarshal([]byte(s.Config), &config); err != nil {
		return nil, fmt.Errorf("failed to parse strategy configuration: %w", err)
	}
	return &config, nil
}

// SetConfig set strategy configuration
func (s *Strategy) SetConfig(config *StrategyConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to serialize strategy configuration: %w", err)
	}
	s.Config = string(data)
	return nil
}
