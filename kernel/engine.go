package kernel

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"nofx/analysis"
	"nofx/logger"
	"nofx/market"
	"nofx/mcp"
	"nofx/provider"
	"nofx/provider/nofxos"
	"nofx/security"
	"nofx/store"
	"regexp"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

// ============================================================================
// Pre-compiled regular expressions (performance optimization)
// ============================================================================

var (
	// Safe regex: precisely match ```json code blocks
	reJSONFence      = regexp.MustCompile(`(?is)` + "```json\\s*(\\[\\s*\\{.*?\\}\\s*\\])\\s*```")
	reJSONArray      = regexp.MustCompile(`(?is)\[\s*\{.*?\}\s*\]`)
	reArrayHead      = regexp.MustCompile(`^\[\s*\{`)
	reArrayOpenSpace = regexp.MustCompile(`^\[\s+\{`)
	reInvisibleRunes = regexp.MustCompile("[\u200B\u200C\u200D\uFEFF]")

	// XML tag extraction (supports any characters in reasoning chain)
	reReasoningTag = regexp.MustCompile(`(?s)<reasoning>(.*?)</reasoning>`)
	reDecisionTag  = regexp.MustCompile(`(?s)<decision>(.*?)</decision>`)
)

// ============================================================================
// Type Definitions
// ============================================================================

// PositionInfo position information
type PositionInfo struct {
	Symbol           string  `json:"symbol"`
	Side             string  `json:"side"` // "long" or "short"
	EntryPrice       float64 `json:"entry_price"`
	MarkPrice        float64 `json:"mark_price"`
	Quantity         float64 `json:"quantity"`
	Leverage         int     `json:"leverage"`
	UnrealizedPnL    float64 `json:"unrealized_pnl"`
	UnrealizedPnLPct float64 `json:"unrealized_pnl_pct"`
	PeakPnLPct       float64 `json:"peak_pnl_pct"` // Historical peak profit percentage
	LiquidationPrice float64 `json:"liquidation_price"`
	MarginUsed       float64 `json:"margin_used"`
	UpdateTime       int64   `json:"update_time"` // Position update timestamp (milliseconds)
	OpenOrders       []OpenOrderInfo `json:"open_orders,omitempty"`
}

// OpenOrderInfo represents an active (open) order that is relevant to an existing position,
// such as stop-loss / take-profit / conditional reduce-only orders.
type OpenOrderInfo struct {
	Symbol       string  `json:"symbol"`
	PositionSide string  `json:"position_side,omitempty"` // "long" or "short" (if available)
	OrderID      string  `json:"order_id,omitempty"`
	Source       string  `json:"source,omitempty"` // exchange-specific source (e.g., "open_order", "algo", "plan")
	Type         string  `json:"type,omitempty"`
	Side         string  `json:"side,omitempty"` // BUY/SELL (if available)
	ReduceOnly   bool    `json:"reduce_only,omitempty"`
	CloseOnly    bool    `json:"close_only,omitempty"`
	Quantity     float64 `json:"quantity,omitempty"`
	Price        float64 `json:"price,omitempty"`
	StopPrice    float64 `json:"stop_price,omitempty"`
	TimeInForce  string  `json:"time_in_force,omitempty"`
	Status       string  `json:"status,omitempty"`
	UpdateTime   int64   `json:"update_time,omitempty"`
}

// AccountInfo account information
type AccountInfo struct {
	TotalEquity      float64 `json:"total_equity"`      // Account equity
	AvailableBalance float64 `json:"available_balance"` // Available balance
	UnrealizedPnL    float64 `json:"unrealized_pnl"`    // Unrealized profit/loss
	TotalPnL         float64 `json:"total_pnl"`         // Total profit/loss
	TotalPnLPct      float64 `json:"total_pnl_pct"`     // Total profit/loss percentage
	MarginUsed       float64 `json:"margin_used"`       // Used margin
	MarginUsedPct    float64 `json:"margin_used_pct"`   // Margin usage rate
	PositionCount    int     `json:"position_count"`    // Number of positions
}

// CandidateCoin candidate coin (from coin pool)
type CandidateCoin struct {
	Symbol               string   `json:"symbol"`
	Sources              []string `json:"sources"` // Sources: "ai500", "oi_top", "otc_top", "static"
	PeriodQuality        string   `json:"period_quality,omitempty"`
	PeriodQualityTime    string   `json:"period_quality_time,omitempty"`
	PeriodQualityExpired bool     `json:"period_quality_expired,omitempty"`
}

// OITopData open interest growth top data (for AI decision reference)
type OITopData struct {
	Rank              int     // OI Top ranking
	OIDeltaPercent    float64 // Open interest change percentage (1 hour)
	OIDeltaValue      float64 // Open interest change value
	PriceDeltaPercent float64 // Price change percentage
}

// TradingStats trading statistics (for AI input)
type TradingStats struct {
	TotalTrades    int     `json:"total_trades"`     // Total number of trades (closed)
	WinRate        float64 `json:"win_rate"`         // Win rate (%)
	ProfitFactor   float64 `json:"profit_factor"`    // Profit factor
	SharpeRatio    float64 `json:"sharpe_ratio"`     // Sharpe ratio
	TotalPnL       float64 `json:"total_pnl"`        // Total profit/loss
	AvgWin         float64 `json:"avg_win"`          // Average win
	AvgLoss        float64 `json:"avg_loss"`         // Average loss
	MaxDrawdownPct float64 `json:"max_drawdown_pct"` // Maximum drawdown (%)
}

// RecentOrder recently completed order (for AI input)
type RecentOrder struct {
	Symbol       string  `json:"symbol"`        // Trading pair
	Side         string  `json:"side"`          // long/short
	EntryPrice   float64 `json:"entry_price"`   // Entry price
	ExitPrice    float64 `json:"exit_price"`    // Exit price
	RealizedPnL  float64 `json:"realized_pnl"`  // Realized profit/loss
	PnLPct       float64 `json:"pnl_pct"`       // Profit/loss percentage
	EntryTime    string  `json:"entry_time"`    // Entry time
	ExitTime     string  `json:"exit_time"`     // Exit time
	HoldDuration string  `json:"hold_duration"` // Hold duration, e.g. "2h30m"
}

// Context trading context (complete information passed to AI)
type Context struct {
	CurrentTime        string                             `json:"current_time"`
	RuntimeMinutes     int                                `json:"runtime_minutes"`
	CallCount          int                                `json:"call_count"`
	Account            AccountInfo                        `json:"account"`
	Positions          []PositionInfo                     `json:"positions"`
	CandidateCoins     []CandidateCoin                    `json:"candidate_coins"`
	PromptVariant      string                             `json:"prompt_variant,omitempty"`
	TradingStats       *TradingStats                      `json:"trading_stats,omitempty"`
	RecentOrders       []RecentOrder                      `json:"recent_orders,omitempty"`
	MarketDataMap      map[string]*market.Data            `json:"-"`
	MultiTFMarket      map[string]map[string]*market.Data `json:"-"`
	OITopDataMap       map[string]*OITopData              `json:"-"`
	QuantDataMap       map[string]*QuantData              `json:"-"`
	OIRankingData      *nofxos.OIRankingData              `json:"-"` // Market-wide OI ranking data
	NetFlowRankingData *nofxos.NetFlowRankingData         `json:"-"` // Market-wide fund flow ranking data
	PriceRankingData   *nofxos.PriceRankingData           `json:"-"` // Market-wide price gainers/losers
	BTCETHLeverage     int                                `json:"-"`
	AltcoinLeverage    int                                `json:"-"`
	Timeframes         []string                           `json:"-"`
}

// Decision AI trading decision
type Decision struct {
	Symbol string `json:"symbol"`
	Action string `json:"action"` // "open_long", "open_short", "close_long", "close_short", "hold", "wait"

	// Opening position parameters
	Leverage        int       `json:"leverage,omitempty"`
	PositionSizeUSD float64   `json:"position_size_usd,omitempty"`
	StopLoss        float64   `json:"stop_loss,omitempty"`
	TakeProfit      float64   `json:"take_profit,omitempty"`
	ExitPlan        *ExitPlan `json:"exit_plan,omitempty"`

	// Common parameters
	Confidence int     `json:"confidence,omitempty"` // Confidence level (0-100)
	RiskUSD    float64 `json:"risk_usd,omitempty"`   // Maximum USD risk
	Reasoning  string  `json:"reasoning"`
}

type priceLookupFunc func(symbol string) (float64, bool)

// UnmarshalJSON accepts both "reasoning" (preferred) and legacy/incorrect "reason".
func (d *Decision) UnmarshalJSON(data []byte) error {
	type wireDecision struct {
		Symbol          string    `json:"symbol"`
		Action          string    `json:"action"`
		Leverage        int       `json:"leverage,omitempty"`
		PositionSizeUSD float64   `json:"position_size_usd,omitempty"`
		StopLoss        float64   `json:"stop_loss,omitempty"`
		TakeProfit      float64   `json:"take_profit,omitempty"`
		ExitPlan        *ExitPlan `json:"exit_plan,omitempty"`
		Confidence      int       `json:"confidence,omitempty"`
		RiskUSD         float64   `json:"risk_usd,omitempty"`
		Reasoning       string    `json:"reasoning,omitempty"`
		Reason          string    `json:"reason,omitempty"`
	}

	var wd wireDecision
	if err := json.Unmarshal(data, &wd); err != nil {
		return err
	}

	d.Symbol = wd.Symbol
	d.Action = wd.Action
	d.Leverage = wd.Leverage
	d.PositionSizeUSD = wd.PositionSizeUSD
	d.StopLoss = wd.StopLoss
	d.TakeProfit = wd.TakeProfit
	d.ExitPlan = wd.ExitPlan
	d.Confidence = wd.Confidence
	d.RiskUSD = wd.RiskUSD
	if strings.TrimSpace(wd.Reasoning) != "" {
		d.Reasoning = wd.Reasoning
	} else {
		d.Reasoning = wd.Reason
	}
	return nil
}

// FullDecision AI's complete decision (including chain of thought)
type FullDecision struct {
	SystemPrompt        string                  `json:"system_prompt"`
	UserPrompt          string                  `json:"user_prompt"`
	CoTTrace            string                  `json:"cot_trace"`
	Decisions           []Decision              `json:"decisions"`
	RawResponse         string                  `json:"raw_response"`
	Timestamp           time.Time               `json:"timestamp"`
	AIRequestDurationMs int64                   `json:"ai_request_duration_ms,omitempty"`
	VisionImages        []store.VisionImageMeta `json:"vision_images,omitempty"`
}

// QuantData quantitative data structure (fund flow, position changes, price changes)
type QuantData struct {
	Symbol      string             `json:"symbol"`
	Price       float64            `json:"price"`
	Netflow     *NetflowData       `json:"netflow,omitempty"`
	OI          map[string]*OIData `json:"oi,omitempty"`
	PriceChange map[string]float64 `json:"price_change,omitempty"`
}

type NetflowData struct {
	Institution *FlowTypeData `json:"institution,omitempty"`
	Personal    *FlowTypeData `json:"personal,omitempty"`
}

type FlowTypeData struct {
	Future map[string]float64 `json:"future,omitempty"`
	Spot   map[string]float64 `json:"spot,omitempty"`
}

type OIData struct {
	CurrentOI float64                 `json:"current_oi"`
	Delta     map[string]*OIDeltaData `json:"delta,omitempty"`
}

type OIDeltaData struct {
	OIDelta        float64 `json:"oi_delta"`
	OIDeltaValue   float64 `json:"oi_delta_value"`
	OIDeltaPercent float64 `json:"oi_delta_percent"`
}

// ============================================================================
// StrategyEngine - Core Strategy Execution Engine
// ============================================================================

// StrategyEngine strategy execution engine
type StrategyEngine struct {
	config       *store.StrategyConfig
	nofxosClient *nofxos.Client
}

// NewStrategyEngine creates strategy execution engine
func NewStrategyEngine(config *store.StrategyConfig) *StrategyEngine {
	apiKey := ""
	if config != nil {
		apiKey = strings.TrimSpace(config.Indicators.NofxOSAPIKey)
	}
	client := nofxos.NewClient("", apiKey)
	return &StrategyEngine{config: config, nofxosClient: client}
}

// GetRiskControlConfig gets risk control configuration
func (e *StrategyEngine) GetRiskControlConfig() store.RiskControlConfig {
	return e.config.RiskControl
}

// GetConfig gets complete strategy configuration
func (e *StrategyEngine) GetConfig() *store.StrategyConfig {
	return e.config
}

// ============================================================================
// Entry Functions - Main API
// ============================================================================

// GetFullDecision gets AI's complete trading decision (batch analysis of all coins and positions)
// Uses default strategy configuration - for production use GetFullDecisionWithStrategy with explicit config
func GetFullDecision(ctx *Context, mcpClient mcp.AIClient) (*FullDecision, error) {
	defaultConfig := store.GetDefaultStrategyConfig("en")
	engine := NewStrategyEngine(&defaultConfig)
	return GetFullDecisionWithStrategy(ctx, mcpClient, engine, "", "", 0)
}

// GetFullDecisionWithStrategy uses StrategyEngine to get AI decision (unified prompt generation)
func GetFullDecisionWithStrategy(ctx *Context, mcpClient mcp.AIClient, engine *StrategyEngine, variant string, traderID string, cycleNumber int) (*FullDecision, error) {
	return getFullDecisionWithStrategyInternal(ctx, mcpClient, nil, engine, variant, traderID, cycleNumber)
}

// GetFullDecisionWithStrategyWithAnalysis runs a pre-analysis stage (text notes) before producing the final JSON decisions.
// analysisClient can be nil to disable the pre-analysis stage.
func GetFullDecisionWithStrategyWithAnalysis(ctx *Context, decisionClient mcp.AIClient, analysisClient mcp.AIClient, engine *StrategyEngine, variant string, traderID string, cycleNumber int) (*FullDecision, error) {
	return getFullDecisionWithStrategyInternal(ctx, decisionClient, analysisClient, engine, variant, traderID, cycleNumber)
}

func getFullDecisionWithStrategyInternal(ctx *Context, mcpClient mcp.AIClient, analysisClient mcp.AIClient, engine *StrategyEngine, variant string, traderID string, cycleNumber int) (*FullDecision, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is nil")
	}
	if engine == nil {
		defaultConfig := store.GetDefaultStrategyConfig("en")
		engine = NewStrategyEngine(&defaultConfig)
	}

	// 1. Fetch market data using strategy config
	if len(ctx.MarketDataMap) == 0 {
		if err := fetchMarketDataWithStrategy(ctx, engine); err != nil {
			return nil, fmt.Errorf("failed to fetch market data: %w", err)
		}
	}

	// Ensure OITopDataMap is initialized
	if ctx.OITopDataMap == nil {
		ctx.OITopDataMap = make(map[string]*OITopData)
		oiPositions, err := engine.nofxosClient.GetOITopPositions()
		if err == nil {
			for _, pos := range oiPositions {
				ctx.OITopDataMap[pos.Symbol] = &OITopData{
					Rank:              pos.Rank,
					OIDeltaPercent:    pos.OIDeltaPercent,
					OIDeltaValue:      pos.OIDeltaValue,
					PriceDeltaPercent: pos.PriceDeltaPercent,
				}
			}
		}
	}

	// 2. Build System Prompt using strategy engine
	riskConfig := engine.GetRiskControlConfig()
	exitPlanID := normalizeExitPlanID(engine.config.PromptSections.ExitStrategyPlan)
	systemPrompt := engine.BuildSystemPrompt(ctx.Account.TotalEquity, variant)

	// 3. Build User Prompt using strategy engine
	userPromptOpts := UserPromptOptions{}
	var visionImages []store.VisionImageMeta
	if engine.config.Vision.Enabled {
		notes, selected, images := engine.collectVisionNotes(ctx, mcpClient, traderID, cycleNumber)
		if len(selected) > 0 {
			userPromptOpts.CandidateSymbols = selected
		}
		if len(notes) > 0 {
			userPromptOpts.VisionNotes = notes
		}
		if len(images) > 0 {
			visionImages = images
		}
	}
	userPrompt := engine.BuildUserPromptWithOptions(ctx, userPromptOpts)

	decisionPrompt := userPrompt
	if analysisClient != nil {
		analysisSystem := engine.buildDecisionAnalysisSystemPrompt()
		analysisStart := time.Now()
		analysisNotes, err := analysisClient.CallWithMessages(analysisSystem, userPrompt)
		analysisDur := time.Since(analysisStart)
		if err != nil {
			logger.Infof("⚠️ Pre-analysis stage failed; continuing without analysis notes: %v", err)
		} else {
			analysisNotes = strings.TrimSpace(analysisNotes)
			if analysisNotes != "" {
				decisionPrompt = appendDecisionAnalysisNotes(userPrompt, analysisNotes)
				logger.Infof("🧠 Pre-analysis stage completed in %d ms", analysisDur.Milliseconds())
			}
		}
	}

	callAIAndParse := func(prompt string) (*FullDecision, error) {
		aiCallStart := time.Now()
		aiResponse, err := mcpClient.CallWithMessages(systemPrompt, prompt)
		aiCallDuration := time.Since(aiCallStart)
		if err != nil {
			return &FullDecision{
				SystemPrompt:        systemPrompt,
				UserPrompt:          prompt,
				Timestamp:           time.Now(),
				AIRequestDurationMs: aiCallDuration.Milliseconds(),
				VisionImages:        visionImages,
			}, fmt.Errorf("AI API call failed: %w", err)
		}

		enforceMinPosSize := true
		if riskConfig.EnforceMinPositionSize != nil {
			enforceMinPosSize = *riskConfig.EnforceMinPositionSize
		}
		priceLookup := makePriceLookup(ctx)
		decision, parseErr := parseFullDecisionResponse(
			aiResponse,
			priceLookup,
			ctx.Account.TotalEquity,
			riskConfig.BTCETHMaxLeverage,
			riskConfig.AltcoinMaxLeverage,
			riskConfig.BTCETHMaxPositionValueRatio,
			riskConfig.AltcoinMaxPositionValueRatio,
			riskConfig.MinPositionSize,
			enforceMinPosSize,
			riskConfig.MinRiskRewardRatio,
			exitPlanID,
		)

		if decision != nil {
			decision.Timestamp = time.Now()
			decision.SystemPrompt = systemPrompt
			decision.UserPrompt = prompt
			decision.AIRequestDurationMs = aiCallDuration.Milliseconds()
			decision.RawResponse = aiResponse
			decision.VisionImages = visionImages
		}

		if parseErr != nil {
			if decision == nil {
				return &FullDecision{
					SystemPrompt:        systemPrompt,
					UserPrompt:          prompt,
					Timestamp:           time.Now(),
					AIRequestDurationMs: aiCallDuration.Milliseconds(),
					RawResponse:         aiResponse,
					VisionImages:        visionImages,
				}, fmt.Errorf("failed to parse AI response: %w", parseErr)
			}
			return decision, fmt.Errorf("failed to parse AI response: %w", parseErr)
		}
		return decision, nil
	}

	var toolDecision *FullDecision
	var toolErr error
	if isDecisionToolEnabled() {
		toolDecision, toolErr = callAIWithDecisionTool(
			ctx,
			mcpClient,
			systemPrompt,
			decisionPrompt,
			riskConfig,
			exitPlanID,
		)
		if toolDecision != nil {
			toolDecision.VisionImages = visionImages
		}
		if toolErr == nil && toolDecision != nil && len(toolDecision.Decisions) > 0 {
			return toolDecision, nil
		}
		if toolErr != nil {
			if shouldTemporarilyDisableDecisionTool(toolErr) {
				disableDecisionToolFor(10*time.Minute, toolErr)
			}
			logger.Infof("AI tool-call decision failed, falling back to text decision parsing: %v", toolErr)
		}
	} else {
		logger.Infof("AI tool-call decision temporarily disabled; using text decision parsing")
	}

	decision, err := callAIAndParse(decisionPrompt)
	const maxDecisionFormatRetries = 2
	lastDecision := decision
	lastAIResponse := ""
	if decision != nil {
		lastAIResponse = decision.RawResponse
	}

	isFormatRelatedError := func(err error) bool {
		if err == nil {
			return false
		}
		s := err.Error()
		if strings.Contains(s, "failed to extract decisions") {
			return true
		}
		if strings.Contains(s, "JSON parsing failed") {
			return true
		}
		if strings.Contains(s, "JSON format validation failed") {
			return true
		}
		if strings.Contains(s, "JSON must start with") {
			return true
		}
		if strings.Contains(s, "not a valid decision array") {
			return true
		}
		return false
	}

	if err != nil {
		// If the error is format-related (e.g., truncated JSON), try to repair it instead of failing the whole cycle.
		if !isFormatRelatedError(err) {
			return decision, err
		}
		logger.Infof("AI output JSON but parsing failed; retrying up to %d time(s) for structured output: %v", maxDecisionFormatRetries, err)
	} else {
		if !isMissingDecisionJSONFallback(decision) {
			return decision, nil
		}
		logger.Infof("AI did not output JSON decision array; retrying up to %d time(s) for structured output", maxDecisionFormatRetries)
	}

	for attempt := 1; attempt <= maxDecisionFormatRetries; attempt++ {
		time.Sleep(time.Duration(attempt) * decisionFormatRetryBaseDelay)
		repairPrompt := buildDecisionRepairPrompt(decisionPrompt, lastAIResponse)

		retryDecision, retryErr := callAIAndParse(repairPrompt)
		if retryDecision != nil {
			lastDecision = retryDecision
			lastAIResponse = retryDecision.RawResponse
		}
		if retryErr != nil {
			logger.Infof("AI decision format retry %d/%d failed: %v", attempt, maxDecisionFormatRetries, retryErr)
			continue
		}
		if isMissingDecisionJSONFallback(retryDecision) {
			logger.Infof("AI decision format retry %d/%d still missing JSON decision array", attempt, maxDecisionFormatRetries)
			continue
		}
		return retryDecision, nil
	}

	// Safe fallback: avoid propagating a hard error from format issues.
	if lastDecision == nil || len(lastDecision.Decisions) == 0 {
		lastDecision = &FullDecision{
			SystemPrompt: systemPrompt,
			UserPrompt:   decisionPrompt,
			RawResponse:  lastAIResponse,
			Timestamp:    time.Now(),
			VisionImages: visionImages,
			Decisions: []Decision{{
				Symbol:    "ALL",
				Action:    "wait",
				Reasoning: "JSON decision parsing failed; entering safe wait",
			}},
		}
	}
	return lastDecision, nil
}

var decisionFormatRetryBaseDelay = 300 * time.Millisecond

const decisionToolName = "submit_decisions"

var decisionToolDisabledUntilUnixNano atomic.Int64

func isDecisionToolEnabled() bool {
	return decisionToolDisabledUntilUnixNano.Load() <= time.Now().UnixNano()
}

func disableDecisionToolFor(d time.Duration, err error) {
	until := time.Now().Add(d).UnixNano()
	decisionToolDisabledUntilUnixNano.Store(until)
	logger.Infof("AI decision tool disabled for %v due to: %v", d, err)
}

func shouldTemporarilyDisableDecisionTool(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	if strings.Contains(s, "API returned error (status 400)") && strings.Contains(s, "invalid_request_error") {
		return true
	}
	if strings.Contains(s, "请求参数不合法") {
		return true
	}
	return false
}

func callAIWithDecisionTool(
	ctx *Context,
	mcpClient mcp.AIClient,
	systemPrompt, userPrompt string,
	riskConfig store.RiskControlConfig,
	exitPlanID string,
) (*FullDecision, error) {
	if mcpClient == nil {
		return nil, fmt.Errorf("mcp client is nil")
	}
	if ctx == nil {
		return nil, fmt.Errorf("context is nil")
	}

	parameters := buildDecisionToolParameters(exitPlanID)
	req := mcp.NewRequestBuilder().
		WithSystemPrompt(systemPrompt).
		WithUserPrompt(userPrompt).
		AddFunction(decisionToolName, "Submit trading decisions as structured JSON (no free-form text).", parameters).
		WithToolChoice("required").
		MustBuild()

	aiCallStart := time.Now()
	toolArgsJSON, err := mcpClient.CallWithRequest(req)
	aiCallDuration := time.Since(aiCallStart)
	if err != nil {
		return nil, err
	}

	cotTrace, decisions, err := parseDecisionToolArguments(toolArgsJSON)
	if err != nil {
		return nil, err
	}

	enforceMinPosSize := true
	if riskConfig.EnforceMinPositionSize != nil {
		enforceMinPosSize = *riskConfig.EnforceMinPositionSize
	}
	priceLookup := makePriceLookup(ctx)
	if err := validateDecisions(
		decisions,
		priceLookup,
		ctx.Account.TotalEquity,
		riskConfig.BTCETHMaxLeverage,
		riskConfig.AltcoinMaxLeverage,
		riskConfig.BTCETHMaxPositionValueRatio,
		riskConfig.AltcoinMaxPositionValueRatio,
		riskConfig.MinPositionSize,
		enforceMinPosSize,
		riskConfig.MinRiskRewardRatio,
		exitPlanID,
	); err != nil {
		return &FullDecision{
			CoTTrace:  cotTrace,
			Decisions: decisions,
		}, fmt.Errorf("decision validation failed: %w", err)
	}

	return &FullDecision{
		SystemPrompt:        systemPrompt,
		UserPrompt:          userPrompt,
		CoTTrace:            cotTrace,
		Decisions:           decisions,
		RawResponse:         toolArgsJSON,
		Timestamp:           time.Now(),
		AIRequestDurationMs: aiCallDuration.Milliseconds(),
	}, nil
}

func parseDecisionToolArguments(toolArgsJSON string) (string, []Decision, error) {
	s := strings.TrimSpace(removeInvisibleRunes(toolArgsJSON))
	if s == "" {
		return "", nil, fmt.Errorf("empty tool arguments")
	}

	type toolPayload struct {
		Reasoning string     `json:"reasoning,omitempty"`
		Decisions []Decision `json:"decisions"`
	}

	var payload toolPayload
	if err := json.Unmarshal([]byte(s), &payload); err != nil {
		s2 := fixMissingQuotes(s)
		if err2 := json.Unmarshal([]byte(s2), &payload); err2 != nil {
			return "", nil, fmt.Errorf("failed to parse tool arguments: %w", err)
		}
	}
	if len(payload.Decisions) == 0 {
		return strings.TrimSpace(payload.Reasoning), nil, fmt.Errorf("tool arguments missing decisions")
	}
	return strings.TrimSpace(payload.Reasoning), payload.Decisions, nil
}

func buildDecisionToolParameters(exitPlanID string) map[string]any {
	planID := normalizeExitPlanID(exitPlanID)
	exitPlanHint := "Optional. If missing, the backend will auto-generate it from stop_loss/take_profit."
	if planID != "" {
		exitPlanHint = fmt.Sprintf("Strongly recommended. If provided, exit_plan.plan_id must be %s.", planID)
	}

	exitPlanSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"plan_id": map[string]any{
				"type":        "string",
				"description": "Exit plan template ID.",
			},
			"children": map[string]any{
				"type":        "array",
				"description": "Exit plan components (tp/sl).",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"component": map[string]any{"type": "string"},
						"handler":   map[string]any{"type": "string"},
						"params":    map[string]any{"type": "object"},
					},
					"required": []string{"component", "handler"},
				},
			},
		},
	}

	decisionBaseProps := map[string]any{
		"symbol": map[string]any{
			"type":        "string",
			"description": "Trading pair like BTCUSDT",
		},
		"action": map[string]any{
			"type": "string",
			"enum": []string{"open_long", "open_short", "close_long", "close_short", "hold", "wait"},
		},
		"leverage": map[string]any{
			"type":        "integer",
			"minimum":     1,
			"description": "Leverage multiplier",
		},
		"position_size_usd": map[string]any{
			"type":        "number",
			"minimum":     0,
			"description": "Position value in USDT",
		},
		"stop_loss": map[string]any{
			"type":        "number",
			"minimum":     0,
			"description": "Stop loss price (absolute price)",
		},
		"take_profit": map[string]any{
			"type":        "number",
			"minimum":     0,
			"description": "Take profit price (absolute price)",
		},
		"exit_plan": map[string]any{
			"description": exitPlanHint,
			"anyOf": []any{
				exitPlanSchema,
				map[string]any{"type": "null"},
			},
		},
		"confidence": map[string]any{
			"type":        "integer",
			"minimum":     0,
			"maximum":     100,
			"description": "Confidence 0-100",
		},
		"risk_usd": map[string]any{
			"type":        "number",
			"minimum":     0,
			"description": "Max risk in USDT",
		},
		"reasoning": map[string]any{
			"type":        "string",
			"description": "One sentence summary",
		},
	}

	openRequired := []string{"symbol", "action", "leverage", "position_size_usd", "stop_loss", "take_profit", "confidence", "risk_usd", "reasoning"}
	closeRequired := []string{"symbol", "action"}
	waitRequired := []string{"symbol", "action", "reasoning"}

	decisionItemSchema := map[string]any{
		"type":       "object",
		"properties": decisionBaseProps,
		"oneOf": []any{
			map[string]any{
				"properties": map[string]any{"action": map[string]any{"const": "open_long"}},
				"required":   openRequired,
			},
			map[string]any{
				"properties": map[string]any{"action": map[string]any{"const": "open_short"}},
				"required":   openRequired,
			},
			map[string]any{
				"properties": map[string]any{"action": map[string]any{"const": "close_long"}},
				"required":   closeRequired,
			},
			map[string]any{
				"properties": map[string]any{"action": map[string]any{"const": "close_short"}},
				"required":   closeRequired,
			},
			map[string]any{
				"properties": map[string]any{"action": map[string]any{"const": "hold"}},
				"required":   waitRequired,
			},
			map[string]any{
				"properties": map[string]any{"action": map[string]any{"const": "wait"}},
				"required":   waitRequired,
			},
		},
	}

	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"reasoning": map[string]any{
				"type":        "string",
				"description": "Short reasoning. Do not include JSON here.",
			},
			"decisions": map[string]any{
				"type":        "array",
				"minItems":    1,
				"description": "Decision array to execute",
				"items":       decisionItemSchema,
			},
		},
		"required": []string{"decisions"},
	}
}

func buildDecisionRepairPrompt(userPrompt, priorAIOutput string) string {
	const priorOutputMaxRunes = 6000
	priorAIOutput = truncateTailRunes(strings.TrimSpace(priorAIOutput), priorOutputMaxRunes)

	var sb strings.Builder
	sb.WriteString(userPrompt)
	sb.WriteString("\n\n")
	sb.WriteString("# OUTPUT REPAIR (CRITICAL)\n")
	sb.WriteString("Your previous reply did not include a JSON decision array and cannot be executed.\n")
	sb.WriteString("Convert your previous reply into the required format.\n")
	sb.WriteString("\n")
	sb.WriteString("## Previous reply (for extraction)\n")
	sb.WriteString(priorAIOutput)
	sb.WriteString("\n\n")
	sb.WriteString("## Requirements (MUST FOLLOW)\n")
	sb.WriteString("- Output ONLY the required XML tags <reasoning> and <decision>.\n")
	sb.WriteString("- Inside <decision>, output a single pure JSON array of objects (no code fences).\n")
	sb.WriteString("- Do NOT add any extra text outside these tags.\n")
	sb.WriteString("- Do NOT change any numbers or decisions from your previous reply.\n")
	sb.WriteString("- If you cannot extract a valid decision array, output:\n<reasoning>- missing structured JSON decision</reasoning>\n<decision>[{\"symbol\":\"ALL\",\"action\":\"wait\",\"reasoning\":\"missing structured JSON decision\"}]</decision>\n")
	return sb.String()
}

func truncateTailRunes(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return "[TRUNCATED]\n" + string(runes[len(runes)-maxRunes:])
}

func isMissingDecisionJSONFallback(fd *FullDecision) bool {
	if fd == nil || len(fd.Decisions) != 1 {
		return false
	}
	d := fd.Decisions[0]
	if strings.TrimSpace(d.Symbol) != "ALL" || strings.TrimSpace(d.Action) != "wait" {
		return false
	}
	return strings.Contains(d.Reasoning, "Model didn't output structured JSON decision")
}

// ============================================================================
// Market Data Fetching
// ============================================================================

// fetchMarketDataWithStrategy fetches market data using strategy config (multiple timeframes)
func fetchMarketDataWithStrategy(ctx *Context, engine *StrategyEngine) error {
	config := engine.GetConfig()
	ctx.MarketDataMap = make(map[string]*market.Data)

	timeframes := config.Indicators.Klines.SelectedTimeframes
	primaryTimeframe := config.Indicators.Klines.PrimaryTimeframe
	klineCount := config.Indicators.Klines.PrimaryCount

	// Compatible with old configuration
	if len(timeframes) == 0 {
		if primaryTimeframe != "" {
			timeframes = append(timeframes, primaryTimeframe)
		} else {
			timeframes = append(timeframes, "3m")
		}
		if config.Indicators.Klines.LongerTimeframe != "" {
			timeframes = append(timeframes, config.Indicators.Klines.LongerTimeframe)
		}
	}

	// Vision mode may require additional timeframes for chart rendering
	if config.Vision.Enabled {
		visionTFs := config.Vision.Timeframes
		if len(visionTFs) == 0 {
			visionTFs = []string{"1h", "15m"}
		}
		seen := make(map[string]bool, len(timeframes)+len(visionTFs))
		for _, tf := range timeframes {
			seen[tf] = true
		}
		for _, tf := range visionTFs {
			tf = strings.TrimSpace(tf)
			if tf == "" || seen[tf] {
				continue
			}
			seen[tf] = true
			timeframes = append(timeframes, tf)
		}
	}
	if primaryTimeframe == "" {
		primaryTimeframe = timeframes[0]
	}
	if klineCount <= 0 {
		klineCount = 30
	}

	logger.Infof("[Info] Strategy timeframes: %v, Primary: %s, Kline count: %d", timeframes, primaryTimeframe, klineCount)

	// 1. First fetch data for position coins (must fetch)
	for _, pos := range ctx.Positions {
		data, err := market.GetWithTimeframes(pos.Symbol, timeframes, primaryTimeframe, klineCount)
		if err != nil {
			logger.Infof("[Warn] Failed to fetch market data for position %s: %v", pos.Symbol, err)
			continue
		}
		ctx.MarketDataMap[pos.Symbol] = data
	}

	// 2. Fetch data for all candidate coins
	positionSymbols := make(map[string]bool)
	for _, pos := range ctx.Positions {
		positionSymbols[pos.Symbol] = true
	}

	minOIThresholdMillions := 15.0 // 15M USD minimum open interest value
	if cfg := engine.GetConfig(); cfg != nil {
		if cfg.RiskControl.MinOpenInterestValueMillions != nil {
			minOIThresholdMillions = *cfg.RiskControl.MinOpenInterestValueMillions
			if minOIThresholdMillions < 0 {
				minOIThresholdMillions = 0
			}
		}
	}

	for _, coin := range ctx.CandidateCoins {
		if _, exists := ctx.MarketDataMap[coin.Symbol]; exists {
			continue
		}

		data, err := market.GetWithTimeframes(coin.Symbol, timeframes, primaryTimeframe, klineCount)
		if err != nil {
			logger.Infof("[Warn] Failed to fetch market data for %s: %v", coin.Symbol, err)
			continue
		}

		// Liquidity filter (skip for xyz dex assets - they don't have OI data from Binance)
		isExistingPosition := positionSymbols[coin.Symbol]
		isXyzAsset := market.IsXyzDexAsset(coin.Symbol)
		isManualCandidate := hasSource(coin.Sources, "static")
		if minOIThresholdMillions > 0 && !isExistingPosition && !isXyzAsset && !isManualCandidate && data.OpenInterest != nil && data.OpenInterest.Latest > 0 && data.CurrentPrice > 0 {
			oiValue := data.OpenInterest.Latest * data.CurrentPrice
			oiValueInMillions := oiValue / 1_000_000
			if oiValueInMillions < minOIThresholdMillions {
				logger.Infof("[Filter] %s OI value too low (%.2fM USD < %.1fM), skipping coin",
					coin.Symbol, oiValueInMillions, minOIThresholdMillions)
				continue
			}
		}

		ctx.MarketDataMap[coin.Symbol] = data
	}

	logger.Infof("[OK] Successfully fetched multi-timeframe market data for %d coins", len(ctx.MarketDataMap))
	return nil
}

// ============================================================================
// Candidate Coins
// ============================================================================

// GetCandidateCoins gets candidate coins based on strategy configuration
func (e *StrategyEngine) GetCandidateCoins() ([]CandidateCoin, error) {
	var candidates []CandidateCoin
	symbolSources := make(map[string][]string)

	coinSource := e.config.CoinSource

	if coinSource.OTCTopAPIURL != "" {
		provider.SetOTCTopAPI(coinSource.OTCTopAPIURL)
	}

	switch coinSource.SourceType {
	case "static":
		for _, symbol := range coinSource.StaticCoins {
			symbol = market.Normalize(symbol)
			candidates = append(candidates, CandidateCoin{
				Symbol:  symbol,
				Sources: []string{"static"},
			})
		}
		return candidates, nil

	case "ai500", "coinpool":
		// 检查 use_coin_pool 标志，如为 false 则回退到静态币种
		if !coinSource.UseAI500 {
			logger.Infof("source_type is 'coinpool' but use_coin_pool is false, falling back to static coins")
			for _, symbol := range coinSource.StaticCoins {
				symbol = market.Normalize(symbol)
				candidates = append(candidates, CandidateCoin{
					Symbol:  symbol,
					Sources: []string{"static"},
				})
			}
			return candidates, nil
		}
		return e.getAI500Coins(coinSource.AI500Limit)

	case "oi_top":
		// 检查 use_oi_top 标志，如为 false 则回退到静态币种
		if !coinSource.UseOITop {
			logger.Infof("source_type is 'oi_top' but use_oi_top is false, falling back to static coins")
			for _, symbol := range coinSource.StaticCoins {
				symbol = market.Normalize(symbol)
				candidates = append(candidates, CandidateCoin{
					Symbol:  symbol,
					Sources: []string{"static"},
				})
			}
			return candidates, nil
		}
		return e.getOITopCoins(coinSource.OITopLimit)

	case "otc_top":
		// Fallback to static coins when use_otc_top is false.
		if !coinSource.UseOTCTop {
			logger.Infof("source_type is 'otc_top' but use_otc_top is false, falling back to static coins")
			for _, symbol := range coinSource.StaticCoins {
				symbol = market.Normalize(symbol)
				candidates = append(candidates, CandidateCoin{
					Symbol:  symbol,
					Sources: []string{"static"},
				})
			}
			return candidates, nil
		}
		return e.getOTCTopCoins()

	case "mixed":
		otcMetaBySymbol := make(map[string]CandidateCoin)
		if coinSource.UseAI500 {
			poolCoins, err := e.getAI500Coins(coinSource.AI500Limit)
			if err != nil {
				logger.Infof("[Warn] Failed to get AI500 coin pool: %v", err)
			} else {
				for _, coin := range poolCoins {
					symbolSources[coin.Symbol] = append(symbolSources[coin.Symbol], "ai500")
				}
			}
		}

		if coinSource.UseOITop {
			oiCoins, err := e.getOITopCoins(coinSource.OITopLimit)
			if err != nil {
				logger.Infof("[Warn] Failed to get OI Top: %v", err)
			} else {
				for _, coin := range oiCoins {
					symbolSources[coin.Symbol] = append(symbolSources[coin.Symbol], "oi_top")
				}
			}
		}

		if coinSource.UseOTCTop {
			otcCoins, err := e.getOTCTopCoins()
			if err != nil {
				logger.Infof("Failed to get OTC Top: %v", err)
			} else {
				for _, coin := range otcCoins {
					symbolSources[coin.Symbol] = append(symbolSources[coin.Symbol], "otc_top")
					otcMetaBySymbol[coin.Symbol] = coin
				}
			}
		}

		for _, symbol := range coinSource.StaticCoins {
			symbol = market.Normalize(symbol)
			if _, exists := symbolSources[symbol]; !exists {
				symbolSources[symbol] = []string{"static"}
			} else {
				symbolSources[symbol] = append(symbolSources[symbol], "static")
			}
		}

		for symbol, sources := range symbolSources {
			candidate := CandidateCoin{
				Symbol:  symbol,
				Sources: sources,
			}
			if hasSource(sources, "otc_top") {
				if meta, ok := otcMetaBySymbol[symbol]; ok {
					candidate.PeriodQuality = meta.PeriodQuality
					candidate.PeriodQualityTime = meta.PeriodQualityTime
					candidate.PeriodQualityExpired = meta.PeriodQualityExpired
				}
			}
			candidates = append(candidates, candidate)
		}
		return candidates, nil

	default:
		return nil, fmt.Errorf("unknown coin source type: %s", coinSource.SourceType)
	}
}

// GetCandidateCoinsWithWarnings returns candidate coins plus user-facing warnings for common misconfigurations.
// It does not attempt to expose underlying provider errors; check logs for detailed fetch failures.
func (e *StrategyEngine) GetCandidateCoinsWithWarnings() ([]CandidateCoin, []string, error) {
	candidates, err := e.GetCandidateCoins()
	if err != nil {
		return nil, nil, err
	}
	if e == nil || e.config == nil {
		return candidates, nil, nil
	}

	var warnings []string
	cs := e.config.CoinSource

	hasAnySource := func(source string) bool {
		for _, c := range candidates {
			if hasSource(c.Sources, source) {
				return true
			}
		}
		return false
	}

	switch cs.SourceType {
	case "mixed":
		if cs.UseAI500 && !hasAnySource("ai500") {
			warnings = append(warnings, "coin_source: mixed has use_ai500=true but AI500 returned 0 symbols (check NofxOS connectivity/auth)")
		}
		if cs.UseOITop && !hasAnySource("oi_top") {
			warnings = append(warnings, "coin_source: mixed has use_oi_top=true but OI Top returned 0 symbols (check NofxOS connectivity/auth)")
		}
		if cs.UseOTCTop && !hasAnySource("otc_top") {
			warnings = append(warnings, "coin_source: mixed has use_otc_top=true but OTC Top returned 0 symbols (check OTC API URL/reachability)")
		}
		if len(cs.StaticCoins) > 0 && !hasAnySource("static") {
			warnings = append(warnings, "coin_source: mixed has static_coins but none were included after normalization")
		}
		if len(candidates) == 0 {
			warnings = append(warnings, "coin_source: mixed produced 0 candidate coins")
		}
	case "static":
		var ignored []string
		if cs.UseAI500 {
			ignored = append(ignored, "use_ai500")
		}
		if cs.UseOITop {
			ignored = append(ignored, "use_oi_top")
		}
		if cs.UseOTCTop {
			ignored = append(ignored, "use_otc_top")
		}
		if len(ignored) > 0 {
			warnings = append(warnings, fmt.Sprintf(
				"coin_source: source_type=static ignores %s; set source_type=mixed to combine sources",
				strings.Join(ignored, ","),
			))
		}
	case "ai500", "coinpool":
		if !cs.UseAI500 {
			warnings = append(warnings, "coin_source: source_type=ai500 but use_ai500=false; falling back to static_coins")
		}
		var ignored []string
		if cs.UseOITop {
			ignored = append(ignored, "use_oi_top")
		}
		if cs.UseOTCTop {
			ignored = append(ignored, "use_otc_top")
		}
		if len(ignored) > 0 {
			warnings = append(warnings, fmt.Sprintf(
				"coin_source: source_type=ai500 ignores %s; set source_type=mixed to combine sources",
				strings.Join(ignored, ","),
			))
		}
	case "oi_top":
		if !cs.UseOITop {
			warnings = append(warnings, "coin_source: source_type=oi_top but use_oi_top=false; falling back to static_coins")
		}
		var ignored []string
		if cs.UseAI500 {
			ignored = append(ignored, "use_ai500")
		}
		if cs.UseOTCTop {
			ignored = append(ignored, "use_otc_top")
		}
		if len(ignored) > 0 {
			warnings = append(warnings, fmt.Sprintf(
				"coin_source: source_type=oi_top ignores %s; set source_type=mixed to combine sources",
				strings.Join(ignored, ","),
			))
		}
	case "otc_top":
		if !cs.UseOTCTop {
			warnings = append(warnings, "coin_source: source_type=otc_top but use_otc_top=false; falling back to static_coins")
		}
		var ignored []string
		if cs.UseAI500 {
			ignored = append(ignored, "use_ai500")
		}
		if cs.UseOITop {
			ignored = append(ignored, "use_oi_top")
		}
		if len(ignored) > 0 {
			warnings = append(warnings, fmt.Sprintf(
				"coin_source: source_type=otc_top ignores %s; set source_type=mixed to combine sources",
				strings.Join(ignored, ","),
			))
		}
	}

	return candidates, warnings, nil
}

func (e *StrategyEngine) getAI500Coins(limit int) ([]CandidateCoin, error) {
	if limit <= 0 {
		limit = 30
	}

	symbols, err := e.nofxosClient.GetTopRatedCoins(limit)
	if err != nil {
		return nil, err
	}

	var candidates []CandidateCoin
	for _, symbol := range symbols {
		candidates = append(candidates, CandidateCoin{
			Symbol:  symbol,
			Sources: []string{"ai500"},
		})
	}
	return candidates, nil
}

func (e *StrategyEngine) getOITopCoins(limit int) ([]CandidateCoin, error) {
	if limit <= 0 {
		limit = 20
	}

	positions, err := e.nofxosClient.GetOITopPositions()
	if err != nil {
		return nil, err
	}

	var candidates []CandidateCoin
	for i, pos := range positions {
		if i >= limit {
			break
		}
		symbol := market.FromBinanceFuturesSymbol(pos.Symbol)
		candidates = append(candidates, CandidateCoin{
			Symbol:  symbol,
			Sources: []string{"oi_top"},
		})
	}
	return candidates, nil
}

func (e *StrategyEngine) getOTCTopCoins() ([]CandidateCoin, error) {
	items, err := provider.GetOTCTopCoins()
	if err != nil {
		return nil, err
	}

	var candidates []CandidateCoin
	now := time.Now().UTC()
	for _, item := range items {
		symbol := market.FromBinanceFuturesSymbol(item.Symbol)
		meta := buildOTCPeriodQualityMeta(item, now)
		candidates = append(candidates, CandidateCoin{
			Symbol:               symbol,
			Sources:              []string{"otc_top"},
			PeriodQuality:        meta.Quality,
			PeriodQualityTime:    meta.Time,
			PeriodQualityExpired: meta.Expired,
		})
	}
	return candidates, nil
}

// ============================================================================
// External & Quant Data
// ============================================================================

// FetchMarketData fetches market data based on strategy configuration
func (e *StrategyEngine) FetchMarketData(symbol string) (*market.Data, error) {
	return market.Get(symbol)
}

// FetchExternalData fetches external data sources
func (e *StrategyEngine) FetchExternalData() (map[string]interface{}, error) {
	externalData := make(map[string]interface{})

	for _, source := range e.config.Indicators.ExternalDataSources {
		data, err := e.fetchSingleExternalSource(source)
		if err != nil {
			logger.Infof("[Warn] Failed to fetch external data source [%s]: %v", source.Name, err)
			continue
		}
		externalData[source.Name] = data
	}

	return externalData, nil
}

func (e *StrategyEngine) fetchSingleExternalSource(source store.ExternalDataSource) (interface{}, error) {
	// SSRF Protection: Validate URL before making request
	if err := security.ValidateURL(source.URL); err != nil {
		return nil, fmt.Errorf("external source URL validation failed: %w", err)
	}

	timeout := time.Duration(source.RefreshSecs) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Use SSRF-safe HTTP client
	client := security.SafeHTTPClient(timeout)

	req, err := http.NewRequest(source.Method, source.URL, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range source.Headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if source.DataPath != "" {
		result = extractJSONPath(result, source.DataPath)
	}

	return result, nil
}

func extractJSONPath(data interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[part]
		} else {
			return nil
		}
	}

	return current
}

// FetchQuantData fetches quantitative data for a single coin
func (e *StrategyEngine) FetchQuantData(symbol string) (*QuantData, error) {
	indicators := e.config.Indicators
	if !indicators.EnableQuantData {
		return nil, nil
	}
	if !indicators.EnableQuantOI && !indicators.EnableQuantNetflow {
		return nil, nil
	}

	includes := []string{"price"}
	if indicators.EnableQuantNetflow {
		includes = append(includes, "netflow")
	}
	if indicators.EnableQuantOI {
		includes = append(includes, "oi")
	}
	include := strings.Join(includes, ",")

	data, err := e.nofxosClient.GetCoinData(symbol, include)
	if err != nil {
		return nil, err
	}
	return convertNofxosQuantData(data), nil
}

func convertNofxosQuantData(data *nofxos.QuantData) *QuantData {
	if data == nil {
		return nil
	}
	b, err := json.Marshal(data)
	if err != nil {
		return &QuantData{
			Symbol: nofxos.NormalizeSymbol(data.Symbol),
			Price:  data.Price,
		}
	}
	var out QuantData
	if err := json.Unmarshal(b, &out); err != nil {
		return &QuantData{
			Symbol: nofxos.NormalizeSymbol(data.Symbol),
			Price:  data.Price,
		}
	}
	if strings.TrimSpace(out.Symbol) == "" {
		out.Symbol = nofxos.NormalizeSymbol(data.Symbol)
	}
	return &out
}

// FetchQuantDataBatch batch fetches quantitative data
func (e *StrategyEngine) FetchQuantDataBatch(symbols []string) map[string]*QuantData {
	result := make(map[string]*QuantData)

	if !e.config.Indicators.EnableQuantData {
		return result
	}

	for _, symbol := range symbols {
		data, err := e.FetchQuantData(symbol)
		if err != nil {
			logger.Infof("[Warn] Failed to fetch quantitative data for %s: %v", symbol, err)
			continue
		}
		if data != nil {
			result[symbol] = data
		}
	}

	return result
}

// FetchOIRankingData fetches market-wide OI ranking data
func (e *StrategyEngine) FetchOIRankingData() *nofxos.OIRankingData {
	indicators := e.config.Indicators
	if !indicators.EnableOIRanking {
		return nil
	}

	duration := indicators.OIRankingDuration
	if duration == "" {
		duration = "1h"
	}

	limit := indicators.OIRankingLimit
	if limit <= 0 {
		limit = 10
	}

	logger.Infof("Fetching OI ranking data (duration: %s, limit: %d)", duration, limit)

	data, err := e.nofxosClient.GetOIRanking(duration, limit)
	if err != nil {
		logger.Warnf("Failed to fetch OI ranking data: %v", err)
		return nil
	}

	logger.Infof("[OK] OI ranking data ready: %d top, %d low positions",
		len(data.TopPositions), len(data.LowPositions))

	return data
}

// FetchNetFlowRankingData fetches market-wide NetFlow ranking data
func (e *StrategyEngine) FetchNetFlowRankingData() *nofxos.NetFlowRankingData {
	indicators := e.config.Indicators
	if !indicators.EnableNetFlowRanking {
		return nil
	}

	duration := indicators.NetFlowRankingDuration
	if duration == "" {
		duration = "1h"
	}

	limit := indicators.NetFlowRankingLimit
	if limit <= 0 {
		limit = 10
	}

	logger.Infof("📊 Fetching NetFlow ranking data (duration: %s, limit: %d)", duration, limit)

	data, err := e.nofxosClient.GetNetFlowRanking(duration, limit)
	if err != nil {
		logger.Warnf("⚠️  Failed to fetch NetFlow ranking data: %v", err)
		return nil
	}

	logger.Infof("✅ NetFlow ranking data ready: inst_in=%d, inst_out=%d, retail_in=%d, retail_out=%d",
		len(data.InstitutionFutureTop), len(data.InstitutionFutureLow),
		len(data.PersonalFutureTop), len(data.PersonalFutureLow))

	return data
}

// FetchPriceRankingData fetches market-wide price ranking data
func (e *StrategyEngine) FetchPriceRankingData() *nofxos.PriceRankingData {
	indicators := e.config.Indicators
	if !indicators.EnablePriceRanking {
		return nil
	}

	duration := indicators.PriceRankingDuration
	if duration == "" {
		duration = "1h"
	}

	limit := indicators.PriceRankingLimit
	if limit <= 0 {
		limit = 10
	}

	logger.Infof("📊 Fetching price ranking data (duration: %s, limit: %d)", duration, limit)

	data, err := e.nofxosClient.GetPriceRanking(duration, limit)
	if err != nil {
		logger.Warnf("⚠️  Failed to fetch price ranking data: %v", err)
		return nil
	}

	logger.Infof("✅ Price ranking data ready: %d durations", len(data.Durations))
	return data
}

// ============================================================================
// Prompt Building - System Prompt
// ============================================================================

// BuildSystemPrompt builds System Prompt according to strategy configuration
func (e *StrategyEngine) BuildSystemPrompt(accountEquity float64, variant string) string {
	var sb strings.Builder
	riskControl := e.config.RiskControl
	promptSections := e.config.PromptSections
	exitPlanID := normalizeExitPlanID(promptSections.ExitStrategyPlan)

	// 0. Data Dictionary & Schema (ensure AI understands all fields)
	lang := detectLanguage(promptSections.RoleDefinition)
	schemaPrompt := GetSchemaPrompt(lang)
	sb.WriteString(schemaPrompt)
	sb.WriteString("\n\n")
	sb.WriteString("---\n\n")

	// 1. Role definition (editable)
	if promptSections.RoleDefinition != "" {
		sb.WriteString(promptSections.RoleDefinition)
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("# You are a professional cryptocurrency trading AI\n\n")
		sb.WriteString("Your task is to make trading decisions based on provided market data.\n\n")
	}

	// 2. Trading mode variant
	switch strings.ToLower(strings.TrimSpace(variant)) {
	case "aggressive":
		sb.WriteString("## Mode: Aggressive\n- Prioritize capturing trend breakouts, can build positions in batches when confidence >=70\n- Allow higher positions, but must strictly set stop-loss and explain risk-reward ratio\n\n")
	case "conservative":
		sb.WriteString("## Mode: Conservative\n- Only open positions when multiple signals resonate\n- Prioritize cash preservation, must pause for multiple periods after consecutive losses\n\n")
	case "scalping":
		sb.WriteString("## Mode: Scalping\n- Focus on short-term momentum, smaller profit targets but require quick action\n- If price doesn't move as expected within two bars, immediately reduce position or stop-loss\n\n")
	}

	// 3. Hard constraints (risk control)
	btcEthPosValueRatio := riskControl.BTCETHMaxPositionValueRatio
	if btcEthPosValueRatio <= 0 {
		btcEthPosValueRatio = 5.0
	}
	altcoinPosValueRatio := riskControl.AltcoinMaxPositionValueRatio
	if altcoinPosValueRatio <= 0 {
		altcoinPosValueRatio = 1.0
	}

	sb.WriteString("# Hard Constraints (Risk Control)\n\n")
	sb.WriteString("## CODE ENFORCED (Backend validation, cannot be bypassed):\n")
	sb.WriteString(fmt.Sprintf("- Max Positions: %d coins simultaneously\n", riskControl.MaxPositions))
	sb.WriteString(fmt.Sprintf("- Position Value Limit (Altcoins): max %.0f USDT (= equity %.0f * %.1fx)\n",
		accountEquity*altcoinPosValueRatio, accountEquity, altcoinPosValueRatio))
	sb.WriteString(fmt.Sprintf("- Position Value Limit (BTC/ETH): max %.0f USDT (= equity %.0f * %.1fx)\n",
		accountEquity*btcEthPosValueRatio, accountEquity, btcEthPosValueRatio))
	sb.WriteString(fmt.Sprintf("- Max Margin Usage: <=%.0f%%\n", riskControl.MaxMarginUsage*100))
	minPositionSize := riskControl.MinPositionSize
	if minPositionSize <= 0 {
		minPositionSize = 12
	}
	enforceMinPositionSize := true
	if riskControl.EnforceMinPositionSize != nil {
		enforceMinPositionSize = *riskControl.EnforceMinPositionSize
	}
	if enforceMinPositionSize {
		sb.WriteString(fmt.Sprintf("- Min Position Size (Altcoins): >=%.0f USDT\n", minPositionSize))
		btcEthMinPositionSize := minPositionSize
		if btcEthMinPositionSize < 60 {
			btcEthMinPositionSize = 60
		}
		sb.WriteString(fmt.Sprintf("- Min Position Size (BTC/ETH): >=%.0f USDT\n\n", btcEthMinPositionSize))
	} else {
		sb.WriteString("- Min Position Size: disabled (exchange may reject small orders)\n\n")
	}

	sb.WriteString("## AI GUIDED (Recommended, you should follow):\n")
	sb.WriteString(fmt.Sprintf("- Trading Leverage: Altcoins max %dx | BTC/ETH max %dx\n",
		riskControl.AltcoinMaxLeverage, riskControl.BTCETHMaxLeverage))
	sb.WriteString(fmt.Sprintf("- Risk-Reward Ratio: >=1:%.1f (take_profit / stop_loss)\n", riskControl.MinRiskRewardRatio))
	if lang == LangChinese {
		sb.WriteString("- RR 不足时：直接输出 wait；不要为了满足 RR 去移动止损/止盈（SL/TP）。优先保持结构失效位的止损与合理目标位。\n")
	} else {
		sb.WriteString("- If RR is insufficient: output wait; do NOT move SL/TP just to satisfy the RR constraint. Keep SL at the structural invalidation level and TP at a realistic target.\n")
	}
	sb.WriteString(fmt.Sprintf("- Min Confidence: >=%d to open position\n\n", riskControl.MinConfidence))

	// Data integrity / anti-hallucination rules
	sb.WriteString("# Data Integrity (Anti-Hallucination)\n\n")
	if lang == LangChinese {
		sb.WriteString("## 必须遵守\n")
		sb.WriteString("- 如果某项数据在本周期输入中不存在/未展示，视为 unknown，禁止编造或推断。\n")
		sb.WriteString("- 只有在出现 `=== <SYMBOL> Quantitative Data ===` 段落时，才允许讨论资金流(Netflow)或 OI 的 5m/15m/1h/4h/12h/24h 变化；否则一律当作 unknown。\n")
		sb.WriteString("\n")
		sb.WriteString("## 均线用法（趋势内偏好）\n")
		sb.WriteString("- EMA21：强势趋势参考（多头趋势中更偏多，空头趋势中更偏空）。\n")
		sb.WriteString("- EMA55：多空都可以做，但在多头趋势中仍偏多，在空头趋势中仍偏空。\n")
		sb.WriteString("- EMA100：多空都可以做，但在多头趋势中更偏空（更像回撤/博弈区），在空头趋势中更偏空。\n")
		sb.WriteString("- EMA200：趋势多空分界线；多头最后防守位/空头最后压制位。有效跌破/突破并确认后，视为趋势拐头信号。\n\n")
	} else {
		sb.WriteString("## MUST FOLLOW\n")
		sb.WriteString("- If a data point is not present in this cycle's input, treat it as unknown. Do NOT fabricate or infer it.\n")
		sb.WriteString("- Only discuss netflow or multi-timeframe OI deltas (5m/15m/1h/4h/12h/24h) when a `=== <SYMBOL> Quantitative Data ===` block is present for that symbol; otherwise treat them as unknown.\n")
		sb.WriteString("\n")
		sb.WriteString("## EMA Heuristics (Trend Bias)\n")
		sb.WriteString("- EMA21: strong trend reference (bias with trend).\n")
		sb.WriteString("- EMA55: tradable both ways, but keep bias with the higher-timeframe trend.\n")
		sb.WriteString("- EMA100: tradable both ways, but in an uptrend it behaves more like a pullback/decision zone (more cautious for longs).\n")
		sb.WriteString("- EMA200: regime boundary; a confirmed break implies trend reversal.\n\n")
	}

	// Position sizing guidance
	sb.WriteString("## Position Sizing Guidance\n")
	sb.WriteString("Calculate `position_size_usd` based on your confidence and the Position Value Limits above:\n")
	sb.WriteString("- High confidence (>=85): Use 80-100%% of max position value limit\n")
	sb.WriteString("- Medium confidence (70-84): Use 50-80%% of max position value limit\n")
	sb.WriteString("- Low confidence (60-69): Use 30-50%% of max position value limit\n")
	sb.WriteString(fmt.Sprintf("- Example: With equity %.0f and BTC/ETH ratio %.1fx, max is %.0f USDT\n",
		accountEquity, btcEthPosValueRatio, accountEquity*btcEthPosValueRatio))
	sb.WriteString("- **DO NOT** just use available_balance as position_size_usd. Use the Position Value Limits!\n\n")
	if riskControl.ATREnabled {
		atrPeriod := riskControl.ATRPeriod
		if atrPeriod <= 0 {
			atrPeriod = 14
		}
		atrTF := strings.TrimSpace(riskControl.ATRTimeframe)
		if atrTF == "" {
			atrTF = "1d"
		}
		riskPct := riskControl.StopLossRiskPct
		if riskPct <= 0 {
			riskPct = 5.0
		}

		sb.WriteString("## ATR Leverage & Stop-Loss Position Sizing (Code Enforced)\n")
		sb.WriteString(fmt.Sprintf("- Leverage = round(close / max_atr_24h) using %s ATR (period %d), clamped to [1, max]\n", atrTF, atrPeriod))
		sb.WriteString(fmt.Sprintf("- position_size_usd = equity * %.1f%% * leverage / stop_distance_pct\n", riskPct))
		sb.WriteString("- Still output stop_loss/take_profit; the system may override leverage and position_size_usd when enabled.\n\n")
	} else if riskControl.StopLossSizingEnabled {
		riskPct := riskControl.StopLossRiskPct
		if riskPct <= 0 {
			riskPct = 5.0
		}
		sb.WriteString("## Stop-Loss Position Sizing (Code Enforced)\n")
		sb.WriteString(fmt.Sprintf("- position_size_usd = equity * %.1f%% * leverage / stop_distance_pct\n", riskPct))
		sb.WriteString("- stop_distance_pct = abs(entry_price - stop_loss) / entry_price * 100\n")
		sb.WriteString("- Still output stop_loss/take_profit; the system overrides position_size_usd before placing orders.\n\n")
	}
	// 4. Trading frequency (editable)
	if promptSections.TradingFrequency != "" {
		sb.WriteString(promptSections.TradingFrequency)
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("# Trading Frequency Awareness\n\n")
		sb.WriteString("- Excellent traders: 2-4 trades/day (about 0.1-0.2 trades/hour)\n")
		sb.WriteString("- >2 trades/hour = Overtrading\n")
		sb.WriteString("- Single position hold time around 30-60 minutes\n")
		sb.WriteString("If you find yourself trading every period -> standards too low; if closing positions < 30 minutes -> too impatient.\n\n")
	}

	// 5. Entry standards (editable)
	if promptSections.EntryStandards != "" {
		sb.WriteString(promptSections.EntryStandards)
		sb.WriteString("\n\nYou have the following indicator data:\n")
		e.writeAvailableIndicators(&sb)
		sb.WriteString(fmt.Sprintf("\n**Confidence >=%d** required to open positions.\n\n", riskControl.MinConfidence))
	} else {
		sb.WriteString("# Entry Standards (Strict)\n\n")
		sb.WriteString("Only open positions when multiple signals resonate. You have:\n")
		e.writeAvailableIndicators(&sb)
		sb.WriteString(fmt.Sprintf("\n**Confidence >=%d** required to open positions.\n\n", riskControl.MinConfidence))
	}

	// 6. Decision process (editable)
	if promptSections.DecisionProcess != "" {
		sb.WriteString(promptSections.DecisionProcess)
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("# Decision Process\n\n")
		sb.WriteString("1. Check positions -> Should we take profit/stop-loss\n")
		sb.WriteString("2. Scan candidate coins + multi-timeframe -> Are there strong signals\n")
		sb.WriteString("3. Write brief public rationale points, then output structured JSON\n\n")
	}

	exitPlanPrompt := buildExitPlanPrompt(exitPlanID)
	if exitPlanPrompt != "" {
		sb.WriteString(exitPlanPrompt)
	}

	// 7. Output format
	sb.WriteString("# Output Format (Strictly Follow)\n\n")
	sb.WriteString("**Must use XML tags <reasoning> and <decision> to separate brief public rationale and decision JSON, avoiding parsing errors**\n\n")
	sb.WriteString("## Format Requirements\n\n")
	sb.WriteString("<reasoning>\n")
	sb.WriteString("Requirements:\n")
	sb.WriteString("- Write brief public rationale notes grouped by symbol (recommended: 2-6 bullets per symbol, total <= 40 lines).\n")
	sb.WriteString("- Do NOT output chain-of-thought.\n")
	sb.WriteString("</reasoning>\n\n")
	sb.WriteString("<decision>\n")
	sb.WriteString("Requirements:\n")
	sb.WriteString("- Only output a pure JSON array inside <decision> ... </decision> (no extra text).\n")
	sb.WriteString("- Do NOT use code fences (no ```).\n\n")
	sb.WriteString("[\n")
	// Use the actual configured position value ratio for BTC/ETH in the example
	examplePositionSize := accountEquity * btcEthPosValueRatio
	exitPlanExample := buildExitPlanExample(exitPlanID)
	sb.WriteString("  {\n")
	sb.WriteString("    \"symbol\": \"BTCUSDT\",\n")
	sb.WriteString("    \"action\": \"open_short\",\n")
	sb.WriteString(fmt.Sprintf("    \"leverage\": %d,\n", riskControl.BTCETHMaxLeverage))
	sb.WriteString(fmt.Sprintf("    \"position_size_usd\": %.0f,\n", examplePositionSize))
	sb.WriteString("    \"stop_loss\": 97000,\n")
	sb.WriteString("    \"take_profit\": 91000,\n")
	sb.WriteString("    \"confidence\": 85,\n")
	sb.WriteString("    \"risk_usd\": 300,\n")
	sb.WriteString("    \"reasoning\": \"One sentence summary of why this action is taken\"")
	if exitPlanExample != "" {
		sb.WriteString(",\n")
		sb.WriteString(exitPlanExample)
	} else {
		sb.WriteString("\n")
	}
	sb.WriteString("  },\n")
	sb.WriteString("  {\"symbol\": \"ETHUSDT\", \"action\": \"close_long\"}\n")
	sb.WriteString("]\n")
	sb.WriteString("</decision>\n\n")
	sb.WriteString("## Field Description\n\n")
	sb.WriteString("- `action`: open_long | open_short | close_long | close_short | hold | wait\n")
	sb.WriteString(fmt.Sprintf("- `confidence`: 0-100 (opening recommended >=%d)\n", riskControl.MinConfidence))
	sb.WriteString("- Required when opening: leverage, position_size_usd, stop_loss, take_profit, confidence, risk_usd\n")
	sb.WriteString("- `exit_plan`: required when opening if exit plan is configured; must match plan_id and component rules\n")
	sb.WriteString("- **IMPORTANT**: All numeric values must be calculated numbers, NOT formulas/expressions (e.g., use `27.76` not `3000 * 0.01`)\n\n")
	sb.WriteString("## JSON Strictness (Must Follow)\n\n")
	sb.WriteString("- No range/approx symbols in JSON: do not use `~` or `～` anywhere (e.g., use `88336`, not `~88336` or `88000~89000`)\n")
	sb.WriteString("- No thousand separators in JSON numbers (e.g., `98000`, not `98,000`)\n")
	sb.WriteString("- No comments in JSON (no `//` or `/* */`)\n\n")
	sb.WriteString("- No trailing commas in JSON\n\n")

	// 8. Custom Prompt
	if e.config.CustomPrompt != "" {
		sb.WriteString("# Personalized Trading Strategy\n\n")
		sb.WriteString(e.config.CustomPrompt)
		sb.WriteString("\n\n")
		sb.WriteString("Note: The above personalized strategy is a supplement to the basic rules and cannot violate the basic risk control principles.\n")
	}

	return sb.String()
}

func (e *StrategyEngine) writeAvailableIndicators(sb *strings.Builder) {
	indicators := e.config.Indicators
	kline := indicators.Klines

	timeframes := make([]string, 0, len(kline.SelectedTimeframes)+2)
	if len(kline.SelectedTimeframes) > 0 {
		timeframes = append(timeframes, kline.SelectedTimeframes...)
	} else if kline.PrimaryTimeframe != "" {
		timeframes = append(timeframes, kline.PrimaryTimeframe)
	}
	if kline.EnableMultiTimeframe && kline.LongerTimeframe != "" {
		timeframes = append(timeframes, kline.LongerTimeframe)
	}
	timeframes = normalizeAndDedupTimeframes(timeframes)

	if len(timeframes) > 0 {
		sb.WriteString(fmt.Sprintf("- K-line OHLCV series (timeframes: %s; primary: %s)\n", strings.Join(timeframes, ", "), kline.PrimaryTimeframe))
	} else {
		sb.WriteString("- K-line OHLCV series\n")
	}
	sb.WriteString("- Technical Analysis (auto-computed per timeframe JSON): pattern, trend structure & key levels (support/resistance), WaveTrend (WT+MFI), divergence, volatility squeeze\n")

	if indicators.EnableEMA {
		sb.WriteString("- EMA indicators")
		if len(indicators.EMAPeriods) > 0 {
			sb.WriteString(fmt.Sprintf(" (periods: %v)", indicators.EMAPeriods))
		}
		sb.WriteString("\n")
	}

	if indicators.EnableMACD {
		sb.WriteString("- MACD indicators\n")
	}

	if indicators.EnableRSI {
		sb.WriteString("- RSI indicators")
		if len(indicators.RSIPeriods) > 0 {
			sb.WriteString(fmt.Sprintf(" (periods: %v)", indicators.RSIPeriods))
		}
		sb.WriteString("\n")
	}

	if indicators.EnableATR {
		sb.WriteString("- ATR indicators")
		if len(indicators.ATRPeriods) > 0 {
			sb.WriteString(fmt.Sprintf(" (periods: %v)", indicators.ATRPeriods))
		}
		sb.WriteString("\n")
	}

	if indicators.EnableBOLL {
		sb.WriteString("- Bollinger Bands (BOLL) - Upper/Middle/Lower bands")
		if len(indicators.BOLLPeriods) > 0 {
			sb.WriteString(fmt.Sprintf(" (periods: %v)", indicators.BOLLPeriods))
		}
		sb.WriteString("\n")
	}

	if indicators.EnableVolume {
		sb.WriteString("- Volume data\n")
	}

	if indicators.EnableOI {
		sb.WriteString("- Open Interest (OI) data\n")
	}

	if indicators.EnableFundingRate {
		sb.WriteString("- Funding rate\n")
	}

	if len(e.config.CoinSource.StaticCoins) > 0 || e.config.CoinSource.UseAI500 || e.config.CoinSource.UseOITop {
		sb.WriteString("- AI500 / OI_Top filter tags (if available)\n")
	}

	if indicators.EnableQuantData {
		sb.WriteString("- Quantitative data (institutional/retail fund flow, position changes, multi-period price changes)\n")
	}
}

// ============================================================================
// Prompt Building - User Prompt
// ============================================================================

type UserPromptOptions struct {
	// CandidateSymbols optionally limits which candidate coins are shown in the prompt.
	// Positions are always shown in full.
	CandidateSymbols []string
	// VisionNotes are optional per-symbol notes produced by vision mode (image reading).
	VisionNotes []VisionNote
}

// BuildUserPrompt builds User Prompt based on strategy configuration
func (e *StrategyEngine) BuildUserPrompt(ctx *Context) string {
	return e.BuildUserPromptWithOptions(ctx, UserPromptOptions{})
}

// BuildUserPromptWithOptions builds User Prompt with optional candidate filtering and vision notes.
func (e *StrategyEngine) BuildUserPromptWithOptions(ctx *Context, opts UserPromptOptions) string {
	var sb strings.Builder

	allowedCandidates := map[string]bool{}
	if len(opts.CandidateSymbols) > 0 {
		for _, s := range opts.CandidateSymbols {
			ns := market.Normalize(s)
			if ns != "" {
				allowedCandidates[ns] = true
			}
		}
	}

	// System status
	sb.WriteString(fmt.Sprintf("Time: %s | Period: #%d | Runtime: %d minutes\n\n",
		ctx.CurrentTime, ctx.CallCount, ctx.RuntimeMinutes))

	// BTC market
	if btcData, hasBTC := ctx.MarketDataMap["BTCUSDT"]; hasBTC {
		sb.WriteString(fmt.Sprintf("BTC: %.2f (1h: %+.2f%%, 4h: %+.2f%%) | MACD: %.4f | RSI: %.2f\n\n",
			btcData.CurrentPrice, btcData.PriceChange1h, btcData.PriceChange4h,
			btcData.CurrentMACD, btcData.CurrentRSI7))
	}

	// Account information
	availableBalancePct := 0.0
	if ctx.Account.TotalEquity > 0 {
		availableBalancePct = (ctx.Account.AvailableBalance / ctx.Account.TotalEquity) * 100
	}
	sb.WriteString(fmt.Sprintf("Account: Equity %.2f | Balance %.2f (%.1f%%) | PnL %+.2f%% | Margin %.1f%% | Positions %d\n\n",
		ctx.Account.TotalEquity,
		ctx.Account.AvailableBalance,
		availableBalancePct,
		ctx.Account.TotalPnLPct,
		ctx.Account.MarginUsedPct,
		ctx.Account.PositionCount))

	// Recently completed orders (placed before positions to ensure visibility)
	if len(ctx.RecentOrders) > 0 {
		sb.WriteString("## Recent Completed Trades\n")
		for i, order := range ctx.RecentOrders {
			resultStr := "Profit"
			if order.RealizedPnL < 0 {
				resultStr = "Loss"
			}
			sb.WriteString(fmt.Sprintf("%d. %s %s | Entry %.4f Exit %.4f | %s: %+.2f USDT (%+.2f%%) | %s -> %s (%s)\n",
				i+1, order.Symbol, order.Side,
				order.EntryPrice, order.ExitPrice,
				resultStr, order.RealizedPnL, order.PnLPct,
				order.EntryTime, order.ExitTime, order.HoldDuration))
		}
		sb.WriteString("\n")
	}

	// Historical trading statistics (helps AI understand past performance)
	if ctx.TradingStats != nil && ctx.TradingStats.TotalTrades > 0 {
		// Detect language from strategy config
		lang := detectLanguage(e.config.PromptSections.RoleDefinition)

		// Win/Loss ratio
		var winLossRatio float64
		if ctx.TradingStats.AvgLoss > 0 {
			winLossRatio = ctx.TradingStats.AvgWin / ctx.TradingStats.AvgLoss
		}

		if lang == LangChinese {
			sb.WriteString("## 历史交易统计\n")
			sb.WriteString(fmt.Sprintf("总交易次数: %d 笔 | 盈利因子: %.2f | 夏普比率: %.2f | 盈亏比: %.2f\n",
				ctx.TradingStats.TotalTrades,
				ctx.TradingStats.ProfitFactor,
				ctx.TradingStats.SharpeRatio,
				winLossRatio))
			sb.WriteString(fmt.Sprintf("总盈亏: %+.2f USDT | 平均盈利: +%.2f | 平均亏损: -%.2f | 最大回撤: %.1f%%\n",
				ctx.TradingStats.TotalPnL,
				ctx.TradingStats.AvgWin,
				ctx.TradingStats.AvgLoss,
				ctx.TradingStats.MaxDrawdownPct))

			// Performance hints based on profit factor, sharpe, and drawdown
			if ctx.TradingStats.ProfitFactor >= 1.5 && ctx.TradingStats.SharpeRatio >= 1 {
				sb.WriteString("表现: 良好 - 保持当前策略\n")
			} else if ctx.TradingStats.ProfitFactor < 1 {
				sb.WriteString("表现: 需要改进 - 提升盈亏比，优化止盈止损\n")
			} else if ctx.TradingStats.MaxDrawdownPct > 30 {
				sb.WriteString("表现: 风险偏高 - 减少仓位，控制回撤\n")
			} else {
				sb.WriteString("表现: 正常 - 仍有优化空间\n")
			}
		} else {
			sb.WriteString("## Historical Trading Statistics\n")
			sb.WriteString(fmt.Sprintf("Total Trades: %d | Profit Factor: %.2f | Sharpe: %.2f | Win/Loss Ratio: %.2f\n",
				ctx.TradingStats.TotalTrades,
				ctx.TradingStats.ProfitFactor,
				ctx.TradingStats.SharpeRatio,
				winLossRatio))
			sb.WriteString(fmt.Sprintf("Total PnL: %+.2f USDT | Avg Win: +%.2f | Avg Loss: -%.2f | Max Drawdown: %.1f%%\n",
				ctx.TradingStats.TotalPnL,
				ctx.TradingStats.AvgWin,
				ctx.TradingStats.AvgLoss,
				ctx.TradingStats.MaxDrawdownPct))

			// Performance hints based on profit factor, sharpe, and drawdown
			if ctx.TradingStats.ProfitFactor >= 1.5 && ctx.TradingStats.SharpeRatio >= 1 {
				sb.WriteString("Performance: GOOD - maintain current strategy\n")
			} else if ctx.TradingStats.ProfitFactor < 1 {
				sb.WriteString("Performance: NEEDS IMPROVEMENT - improve win/loss ratio, optimize TP/SL\n")
			} else if ctx.TradingStats.MaxDrawdownPct > 30 {
				sb.WriteString("Performance: HIGH RISK - reduce position size, control drawdown\n")
			} else {
				sb.WriteString("Performance: NORMAL - room for optimization\n")
			}
		}
		sb.WriteString("\n")
	}

	// Position information
	if len(ctx.Positions) > 0 {
		sb.WriteString("## Current Positions\n")
		for i, pos := range ctx.Positions {
			sb.WriteString(e.formatPositionInfo(i+1, pos, ctx))
		}
	} else {
		sb.WriteString("Current Positions: None\n\n")
	}

	// Candidate coins (exclude coins already in positions to avoid duplicate data)
	positionSymbols := make(map[string]bool)
	for _, pos := range ctx.Positions {
		// Normalize symbol to handle both "ETH" and "ETHUSDT" formats
		normalizedSymbol := market.Normalize(pos.Symbol)
		positionSymbols[normalizedSymbol] = true
	}

	filteredCandidates := make([]CandidateCoin, 0, len(ctx.CandidateCoins))
	for _, coin := range ctx.CandidateCoins {
		normalizedCoinSymbol := market.Normalize(coin.Symbol)
		if normalizedCoinSymbol == "" {
			continue
		}
		// Skip if this coin is already a position (data already shown in positions section)
		if positionSymbols[normalizedCoinSymbol] {
			continue
		}
		if len(allowedCandidates) > 0 && !allowedCandidates[normalizedCoinSymbol] {
			continue
		}
		filteredCandidates = append(filteredCandidates, coin)
	}

	sb.WriteString(fmt.Sprintf("## Candidate Coins (%d coins)\n\n", len(filteredCandidates)))
	displayedCount := 0
	for _, coin := range filteredCandidates {
		normalizedCoinSymbol := market.Normalize(coin.Symbol)
		if normalizedCoinSymbol == "" {
			continue
		}

		displayedCount++

		sourceTags := e.formatCoinSourceTag(coin.Sources)
		sb.WriteString(fmt.Sprintf("### %d. %s%s\n\n", displayedCount, normalizedCoinSymbol, sourceTags))
		if line := formatOTCPeriodQualityLine(coin); line != "" {
			sb.WriteString(line + "\n\n")
		}

		marketData, hasData := ctx.MarketDataMap[normalizedCoinSymbol]
		if !hasData || marketData == nil {
			sb.WriteString("Market Data: unavailable for this symbol in this cycle (treat all technical/quant data as unknown; do not open new positions based on missing data).\n\n")
			continue
		}

		sb.WriteString(e.formatMarketData(marketData))

		if ctx.QuantDataMap != nil {
			if quantData, hasQuant := ctx.QuantDataMap[normalizedCoinSymbol]; hasQuant {
				sb.WriteString(e.formatQuantData(quantData))
			} else if e.config.Indicators.EnableQuantData {
				sb.WriteString("Quantitative Data: unavailable for this symbol in this cycle (treat fund flow / OI deltas as unknown).\n")
			}
		} else if e.config.Indicators.EnableQuantData {
			sb.WriteString("Quantitative Data: unavailable for this symbol in this cycle (treat fund flow / OI deltas as unknown).\n")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	if len(opts.VisionNotes) > 0 {
		sb.WriteString("## Chart Vision Notes (from images)\n\n")
		for _, n := range opts.VisionNotes {
			sym := strings.TrimSpace(n.Symbol)
			note := strings.TrimSpace(n.Note)
			if sym == "" || note == "" {
				continue
			}
			sb.WriteString(fmt.Sprintf("### %s\n%s\n\n", sym, note))
		}
	}

	// OI Ranking data (market-wide open interest changes)
	if ctx.OIRankingData != nil || ctx.NetFlowRankingData != nil || ctx.PriceRankingData != nil {
		lang := detectLanguage(e.config.PromptSections.RoleDefinition)
		nofxLang := toNofxOSLang(lang)
		if ctx.OIRankingData != nil {
			sb.WriteString(nofxos.FormatOIRankingForAI(ctx.OIRankingData, nofxLang))
		}
		if ctx.NetFlowRankingData != nil {
			sb.WriteString(nofxos.FormatNetFlowRankingForAI(ctx.NetFlowRankingData, nofxLang))
		}
		if ctx.PriceRankingData != nil {
			sb.WriteString(nofxos.FormatPriceRankingForAI(ctx.PriceRankingData, nofxLang))
		}
	}

	sb.WriteString("---\n\n")
	sb.WriteString("Now please analyze and output your decision (Chain of Thought + JSON)\n")

	return sb.String()
}

func (e *StrategyEngine) formatPositionInfo(index int, pos PositionInfo, ctx *Context) string {
	var sb strings.Builder

	holdingDuration := ""
	if pos.UpdateTime > 0 {
		durationMs := time.Now().UnixMilli() - pos.UpdateTime
		durationMin := durationMs / (1000 * 60)
		if durationMin < 60 {
			holdingDuration = fmt.Sprintf(" | Holding Duration %d min", durationMin)
		} else {
			durationHour := durationMin / 60
			durationMinRemainder := durationMin % 60
			holdingDuration = fmt.Sprintf(" | Holding Duration %dh %dm", durationHour, durationMinRemainder)
		}
	}

	positionValue := pos.Quantity * pos.MarkPrice
	if positionValue < 0 {
		positionValue = -positionValue
	}

	sb.WriteString(fmt.Sprintf("%d. %s %s | Entry %.4f Current %.4f | Qty %.4f | Position Value %.2f USDT | PnL%+.2f%% | PnL Amount%+.2f USDT | Peak PnL%.2f%% | Leverage %dx | Margin %.0f | Liq Price %.4f%s\n\n",
		index, pos.Symbol, strings.ToUpper(pos.Side),
		pos.EntryPrice, pos.MarkPrice, pos.Quantity, positionValue, pos.UnrealizedPnLPct, pos.UnrealizedPnL, pos.PeakPnLPct,
		pos.Leverage, pos.MarginUsed, pos.LiquidationPrice, holdingDuration))

	if meta := findOTCTopCandidate(ctx, pos.Symbol); meta != nil {
		if line := formatOTCPeriodQualityLine(*meta); line != "" {
			sb.WriteString(line + "\n\n")
		}
	}

	if len(pos.OpenOrders) > 0 {
		dash := func(s string) string {
			s = strings.TrimSpace(s)
			if s == "" {
				return "-"
			}
			return s
		}
		fnum := func(v float64) string {
			if v == 0 {
				return "-"
			}
			return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.8f", v), "0"), ".")
		}

		sb.WriteString("Active Position Orders (open TP/SL/conditional):\n")
		for _, o := range pos.OpenOrders {
			sb.WriteString(fmt.Sprintf("- symbol=%s position_side=%s reduce_only=%t type=%s qty=%s price=%s stop_price=%s time_in_force=%s status=%s\n",
				dash(o.Symbol),
				dash(o.PositionSide),
				o.ReduceOnly,
				dash(o.Type),
				fnum(o.Quantity),
				fnum(o.Price),
				fnum(o.StopPrice),
				dash(o.TimeInForce),
				dash(o.Status),
			))
		}
		sb.WriteString("\n")
	}

	if marketData, ok := ctx.MarketDataMap[pos.Symbol]; ok {
		sb.WriteString(e.formatMarketData(marketData))

		if ctx.QuantDataMap != nil {
			if quantData, hasQuant := ctx.QuantDataMap[pos.Symbol]; hasQuant {
				sb.WriteString(e.formatQuantData(quantData))
			} else if e.config.Indicators.EnableQuantData {
				sb.WriteString("Quantitative Data: unavailable for this symbol in this cycle (treat fund flow / OI deltas as unknown).\n")
			}
		} else if e.config.Indicators.EnableQuantData {
			sb.WriteString("Quantitative Data: unavailable for this symbol in this cycle (treat fund flow / OI deltas as unknown).\n")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (e *StrategyEngine) formatCoinSourceTag(sources []string) string {
	if len(sources) == 0 {
		return ""
	}

	priority := map[string]int{
		"ai500":   1,
		"oi_top":  2,
		"otc_top": 3,
		"static":  4,
	}
	ordered := make([]string, 0, len(sources))
	for _, s := range sources {
		s = strings.TrimSpace(s)
		if s != "" {
			ordered = append(ordered, s)
		}
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		pi, okI := priority[ordered[i]]
		pj, okJ := priority[ordered[j]]
		if okI && okJ {
			return pi < pj
		}
		if okI != okJ {
			return okI
		}
		return strings.ToLower(ordered[i]) < strings.ToLower(ordered[j])
	})

	labels := make([]string, 0, len(ordered))
	for _, src := range ordered {
		switch src {
		case "ai500":
			labels = append(labels, "AI500")
		case "oi_top":
			labels = append(labels, "OI_Top")
		case "otc_top":
			labels = append(labels, "OTC Top")
		case "static":
			labels = append(labels, "Manual")
		default:
			labels = append(labels, src)
		}
	}
	if len(labels) == 1 {
		switch ordered[0] {
		case "oi_top":
			return " (OI_Top position growth)"
		case "static":
			return " (Manual selection)"
		default:
			return fmt.Sprintf(" (%s)", labels[0])
		}
	}
	return fmt.Sprintf(" (%s multi-signal)", strings.Join(labels, "+"))
}

// ============================================================================
// Market Data Formatting
// ============================================================================

func (e *StrategyEngine) formatMarketData(data *market.Data) string {
	var sb strings.Builder
	indicators := e.config.Indicators

	// 明确标注币种
	sb.WriteString(fmt.Sprintf("=== %s Market Data ===\n\n", data.Symbol))
	sb.WriteString(fmt.Sprintf("current_price = %.4f", data.CurrentPrice))

	if indicators.EnableEMA {
		sb.WriteString(fmt.Sprintf(", current_ema21 = %.3f, current_ema55 = %.3f, current_ema100 = %.3f, current_ema200 = %.3f",
			data.CurrentEMA21, data.CurrentEMA55, data.CurrentEMA100, data.CurrentEMA200))
	}

	if indicators.EnableMACD {
		sb.WriteString(fmt.Sprintf(", current_macd = %.3f", data.CurrentMACD))
	}

	if indicators.EnableRSI {
		sb.WriteString(fmt.Sprintf(", current_rsi7 = %.3f", data.CurrentRSI7))
	}

	sb.WriteString("\n\n")

	if indicators.EnableOI || indicators.EnableFundingRate {
		sb.WriteString(fmt.Sprintf("Additional data for %s:\n\n", data.Symbol))

		if indicators.EnableOI && data.OpenInterest != nil {
			sb.WriteString(fmt.Sprintf("Open Interest: Latest: %.2f Average: %.2f\n\n",
				data.OpenInterest.Latest, data.OpenInterest.Average))
		}

		if indicators.EnableFundingRate {
			sb.WriteString(fmt.Sprintf("Funding Rate: %.2e\n\n", data.FundingRate))
		}
	}

	if len(data.TimeframeData) > 0 {
		timeframeOrder := normalizeAndDedupTimeframes(indicators.Klines.SelectedTimeframes)
		if len(timeframeOrder) == 0 {
			timeframeOrder = []string{"1m", "3m", "5m", "15m", "30m", "1h", "2h", "4h", "6h", "8h", "12h", "1d", "3d", "1w"}
		}
		for _, tf := range timeframeOrder {
			if tfData, ok := data.TimeframeData[tf]; ok {
				sb.WriteString(fmt.Sprintf("=== %s Timeframe (oldest -> latest) ===\n\n", strings.ToUpper(tf)))
				e.formatTimeframeSeriesData(&sb, tfData, indicators)
			}
		}
	} else {
		// Compatible with old data format
		if data.IntradaySeries != nil {
			klineConfig := indicators.Klines
			sb.WriteString(fmt.Sprintf("Intraday series (%s intervals, oldest -> latest):\n\n", klineConfig.PrimaryTimeframe))

			if len(data.IntradaySeries.MidPrices) > 0 {
				sb.WriteString(fmt.Sprintf("Mid prices: %s\n\n", formatFloatSlice(data.IntradaySeries.MidPrices)))
			}

			if indicators.EnableEMA && len(data.IntradaySeries.EMA21Values) > 0 {
				sb.WriteString(fmt.Sprintf("EMA indicators (21-period): %s\n\n", formatFloatSlice(data.IntradaySeries.EMA21Values)))
			}

			if indicators.EnableEMA && len(data.IntradaySeries.EMA55Values) > 0 {
				sb.WriteString(fmt.Sprintf("EMA indicators (55-period): %s\n\n", formatFloatSlice(data.IntradaySeries.EMA55Values)))
			}

			if indicators.EnableEMA && len(data.IntradaySeries.EMA100Values) > 0 {
				sb.WriteString(fmt.Sprintf("EMA indicators (100-period): %s\n\n", formatFloatSlice(data.IntradaySeries.EMA100Values)))
			}

			if indicators.EnableEMA && len(data.IntradaySeries.EMA200Values) > 0 {
				sb.WriteString(fmt.Sprintf("EMA indicators (200-period): %s\n\n", formatFloatSlice(data.IntradaySeries.EMA200Values)))
			}

			if indicators.EnableMACD && len(data.IntradaySeries.MACDValues) > 0 {
				sb.WriteString(fmt.Sprintf("MACD indicators: %s\n\n", formatFloatSlice(data.IntradaySeries.MACDValues)))
			}

			if indicators.EnableRSI {
				if len(data.IntradaySeries.RSI7Values) > 0 {
					sb.WriteString(fmt.Sprintf("RSI indicators (7-Period): %s\n\n", formatFloatSlice(data.IntradaySeries.RSI7Values)))
				}
				if len(data.IntradaySeries.RSI14Values) > 0 {
					sb.WriteString(fmt.Sprintf("RSI indicators (14-Period): %s\n\n", formatFloatSlice(data.IntradaySeries.RSI14Values)))
				}
			}

			if indicators.EnableVolume && len(data.IntradaySeries.Volume) > 0 {
				sb.WriteString(fmt.Sprintf("Volume: %s\n\n", formatFloatSlice(data.IntradaySeries.Volume)))
			}

			if indicators.EnableATR {
				sb.WriteString(fmt.Sprintf("3m ATR (14-period): %.3f\n\n", data.IntradaySeries.ATR14))
			}
		}

		if data.LongerTermContext != nil && indicators.Klines.EnableMultiTimeframe {
			sb.WriteString(fmt.Sprintf("Longer-term context (%s timeframe):\n\n", indicators.Klines.LongerTimeframe))

			if indicators.EnableEMA {
				sb.WriteString(fmt.Sprintf("EMA21: %.3f | EMA55: %.3f | EMA100: %.3f | EMA200: %.3f\n\n",
					data.LongerTermContext.EMA21, data.LongerTermContext.EMA55, data.LongerTermContext.EMA100, data.LongerTermContext.EMA200))
			}

			if indicators.EnableATR {
				sb.WriteString(fmt.Sprintf("3-Period ATR: %.3f vs. 14-Period ATR: %.3f\n\n",
					data.LongerTermContext.ATR3, data.LongerTermContext.ATR14))
			}

			if indicators.EnableVolume {
				sb.WriteString(fmt.Sprintf("Current Volume: %.3f vs. Average Volume: %.3f\n\n",
					data.LongerTermContext.CurrentVolume, data.LongerTermContext.AverageVolume))
			}

			if indicators.EnableMACD && len(data.LongerTermContext.MACDValues) > 0 {
				sb.WriteString(fmt.Sprintf("MACD indicators: %s\n\n", formatFloatSlice(data.LongerTermContext.MACDValues)))
			}

			if indicators.EnableRSI && len(data.LongerTermContext.RSI14Values) > 0 {
				sb.WriteString(fmt.Sprintf("RSI indicators (14-Period): %s\n\n", formatFloatSlice(data.LongerTermContext.RSI14Values)))
			}
		}
	}

	// Technical Analysis (pattern detection, wavetrend, trend analysis)
	// Perform analysis according to selected timeframes (from strategy config) if available.
	if len(data.TimeframeData) > 0 {
		klineCfg := indicators.Klines
		analysisEnabled := boolPtrOrDefault(indicators.EnableTechnicalAnalysis, true)
		analysisCfg := analysis.DefaultConfig()
		analysisCfg.EnablePattern = boolPtrOrDefault(indicators.EnablePattern, true)
		analysisCfg.EnableWaveTrend = boolPtrOrDefault(indicators.EnableWaveTrend, true)
		analysisCfg.EnableDivergence = boolPtrOrDefault(indicators.EnableDivergence, true)
		analysisCfg.EnableVolatilityWarning = boolPtrOrDefault(indicators.EnableSqueeze, true)
		analysisCfg.EnableTrend = boolPtrOrDefault(indicators.EnableTrend, true)
		analysisCfg.EnableCVD = boolPtrOrDefault(indicators.EnableCVD, true)

		analysisTimeframes := make([]string, 0, len(klineCfg.SelectedTimeframes)+1)
		if len(klineCfg.SelectedTimeframes) > 0 {
			analysisTimeframes = append(analysisTimeframes, klineCfg.SelectedTimeframes...)
		} else if klineCfg.PrimaryTimeframe != "" {
			analysisTimeframes = append(analysisTimeframes, klineCfg.PrimaryTimeframe)
		}

		analysisTimeframes = normalizeAndDedupTimeframes(analysisTimeframes)
		if len(analysisTimeframes) == 0 {
			analysisTimeframes = []string{"15m", "1h", "4h", "5m"}
		}

		if !analysisEnabled {
			sb.WriteString("=== Technical Analysis ===\n\n")
			sb.WriteString("```json\n")
			sb.WriteString(`{"note":"disabled_by_strategy_config"}`)
			sb.WriteString("\n```\n\n")
			return sb.String()
		}

		for _, tf := range analysisTimeframes {
			sb.WriteString(fmt.Sprintf("=== Technical Analysis (%s) ===\n\n", strings.ToUpper(tf)))

			tfData, ok := data.TimeframeData[tf]
			if !ok || tfData == nil || len(tfData.Klines) == 0 {
				sb.WriteString("```json\n")
				sb.WriteString(`{"note":"timeframe_data_missing_or_empty"}`)
				sb.WriteString("\n```\n\n")
				continue
			}

			klines := convertKlineBarsToKlines(tf, tfData.Klines)
			analysisResult := safeAnalyze(klines, analysisCfg)
			analysisJSON := formatAnalysisEnvelopeJSON(tf, tfData, analysisResult)

			sb.WriteString("```json\n")
			sb.WriteString(analysisJSON)
			sb.WriteString("\n```\n\n")
		}
	}

	return sb.String()
}

func safeAnalyze(klines []market.Kline, cfg analysis.Config) (result *analysis.AnalysisResult) {
	defer func() {
		if r := recover(); r != nil {
			result = nil
		}
	}()
	return analysis.Analyze(klines, cfg)
}

type emaSnapshot struct {
	TF     string   `json:"tf,omitempty"`
	Price  float64  `json:"price,omitempty"`
	EMA21  *float64 `json:"ema21,omitempty"`
	EMA55  *float64 `json:"ema55,omitempty"`
	EMA100 *float64 `json:"ema100,omitempty"`
	EMA200 *float64 `json:"ema200,omitempty"`
}

type analysisEnvelope struct {
	Note string `json:"note,omitempty"`
	*analysis.AnalysisResult
	EMASnapshot *emaSnapshot `json:"ema_snapshot,omitempty"`
}

func formatAnalysisEnvelopeJSON(tf string, tfData *market.TimeframeSeriesData, analysisResult *analysis.AnalysisResult) string {
	envelope := analysisEnvelope{
		AnalysisResult: analysisResult,
		EMASnapshot:    buildEMASnapshot(tf, tfData),
	}
	if envelope.AnalysisResult == nil {
		envelope.Note = "analysis_unavailable"
		envelope.AnalysisResult = &analysis.AnalysisResult{}
	}

	data, err := json.Marshal(envelope)
	if err != nil {
		return `{"note":"analysis_unavailable"}`
	}
	if len(data) == 0 || string(data) == "{}" {
		return `{"note":"analysis_unavailable"}`
	}
	return string(data)
}

func boolPtrOrDefault(v *bool, def bool) bool {
	if v == nil {
		return def
	}
	return *v
}

func buildEMASnapshot(tf string, tfData *market.TimeframeSeriesData) *emaSnapshot {
	if tfData == nil {
		return nil
	}

	price := 0.0
	if len(tfData.Klines) > 0 {
		price = tfData.Klines[len(tfData.Klines)-1].Close
	} else if len(tfData.MidPrices) > 0 {
		price = tfData.MidPrices[len(tfData.MidPrices)-1]
	}
	if price <= 0 {
		return nil
	}

	snap := &emaSnapshot{TF: tf, Price: price}
	if v, ok := lastPositive(tfData.EMA21Values); ok {
		snap.EMA21 = float64Ptr(v)
	}
	if v, ok := lastPositive(tfData.EMA55Values); ok {
		snap.EMA55 = float64Ptr(v)
	}
	if v, ok := lastPositive(tfData.EMA100Values); ok {
		snap.EMA100 = float64Ptr(v)
	}
	if v, ok := lastPositive(tfData.EMA200Values); ok {
		snap.EMA200 = float64Ptr(v)
	}
	return snap
}

func lastPositive(values []float64) (float64, bool) {
	for i := len(values) - 1; i >= 0; i-- {
		if values[i] > 0 {
			return values[i], true
		}
	}
	return 0, false
}

func float64Ptr(v float64) *float64 {
	return &v
}

func normalizeAndDedupTimeframes(timeframes []string) []string {
	seen := make(map[string]bool, len(timeframes))
	out := make([]string, 0, len(timeframes))
	for _, tf := range timeframes {
		normalized := strings.ToLower(strings.TrimSpace(tf))
		if normalized == "" {
			continue
		}
		if seen[normalized] {
			continue
		}
		seen[normalized] = true
		out = append(out, normalized)
	}
	return out
}

func (e *StrategyEngine) formatTimeframeSeriesData(sb *strings.Builder, data *market.TimeframeSeriesData, indicators store.IndicatorConfig) {
	if len(data.Klines) > 0 {
		sb.WriteString("Time(UTC)      Open      High      Low       Close     Volume\n")
		for i, k := range data.Klines {
			t := time.Unix(k.Time/1000, 0).UTC()
			timeStr := t.Format("01-02 15:04")
			marker := ""
			if i == len(data.Klines)-1 {
				marker = "  <- current"
			}
			sb.WriteString(fmt.Sprintf("%-14s %-9.4f %-9.4f %-9.4f %-9.4f %-12.2f%s\n",
				timeStr, k.Open, k.High, k.Low, k.Close, k.Volume, marker))
		}
		sb.WriteString("\n")
	} else if len(data.MidPrices) > 0 {
		sb.WriteString(fmt.Sprintf("Mid prices: %s\n\n", formatFloatSlice(data.MidPrices)))
		if indicators.EnableVolume && len(data.Volume) > 0 {
			sb.WriteString(fmt.Sprintf("Volume: %s\n\n", formatFloatSlice(data.Volume)))
		}
	}

	if indicators.EnableEMA {
		if len(data.EMA21Values) > 0 {
			sb.WriteString(fmt.Sprintf("EMA21: %s\n", formatFloatSlice(data.EMA21Values)))
		}
		if len(data.EMA55Values) > 0 {
			sb.WriteString(fmt.Sprintf("EMA55: %s\n", formatFloatSlice(data.EMA55Values)))
		}
		if len(data.EMA100Values) > 0 {
			sb.WriteString(fmt.Sprintf("EMA100: %s\n", formatFloatSlice(data.EMA100Values)))
		}
		if len(data.EMA200Values) > 0 {
			sb.WriteString(fmt.Sprintf("EMA200: %s\n", formatFloatSlice(data.EMA200Values)))
		}
	}

	if indicators.EnableMACD && len(data.MACDValues) > 0 {
		sb.WriteString(fmt.Sprintf("MACD: %s\n", formatFloatSlice(data.MACDValues)))
	}

	if indicators.EnableRSI {
		if len(data.RSI7Values) > 0 {
			sb.WriteString(fmt.Sprintf("RSI7: %s\n", formatFloatSlice(data.RSI7Values)))
		}
		if len(data.RSI14Values) > 0 {
			sb.WriteString(fmt.Sprintf("RSI14: %s\n", formatFloatSlice(data.RSI14Values)))
		}
	}

	if indicators.EnableATR && data.ATR14 > 0 {
		sb.WriteString(fmt.Sprintf("ATR14: %.4f\n", data.ATR14))
	}

	if indicators.EnableBOLL && len(data.BOLLUpper) > 0 {
		sb.WriteString(fmt.Sprintf("BOLL Upper: %s\n", formatFloatSlice(data.BOLLUpper)))
		sb.WriteString(fmt.Sprintf("BOLL Middle: %s\n", formatFloatSlice(data.BOLLMiddle)))
		sb.WriteString(fmt.Sprintf("BOLL Lower: %s\n", formatFloatSlice(data.BOLLLower)))
	}

	sb.WriteString("\n")
}

func (e *StrategyEngine) formatQuantData(data *QuantData) string {
	if data == nil {
		return ""
	}

	indicators := e.config.Indicators
	if !indicators.EnableQuantOI && !indicators.EnableQuantNetflow {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== %s Quantitative Data ===\n", data.Symbol))

	if len(data.PriceChange) > 0 {
		sb.WriteString("Price Change: ")
		timeframes := []string{"5m", "15m", "1h", "4h", "12h", "24h"}
		parts := []string{}
		for _, tf := range timeframes {
			if v, ok := data.PriceChange[tf]; ok {
				parts = append(parts, fmt.Sprintf("%s: %+.4f%%", tf, v*100))
			}
		}
		sb.WriteString(strings.Join(parts, " | "))
		sb.WriteString("\n")
	}

	if indicators.EnableQuantNetflow && data.Netflow != nil {
		sb.WriteString("Fund Flow (Netflow):\n")
		timeframes := []string{"5m", "15m", "1h", "4h", "12h", "24h"}

		if data.Netflow.Institution != nil {
			if data.Netflow.Institution.Future != nil && len(data.Netflow.Institution.Future) > 0 {
				sb.WriteString("  Institutional Futures:\n")
				for _, tf := range timeframes {
					if v, ok := data.Netflow.Institution.Future[tf]; ok {
						sb.WriteString(fmt.Sprintf("    %s: %s\n", tf, formatFlowValue(v)))
					}
				}
			}
			if data.Netflow.Institution.Spot != nil && len(data.Netflow.Institution.Spot) > 0 {
				sb.WriteString("  Institutional Spot:\n")
				for _, tf := range timeframes {
					if v, ok := data.Netflow.Institution.Spot[tf]; ok {
						sb.WriteString(fmt.Sprintf("    %s: %s\n", tf, formatFlowValue(v)))
					}
				}
			}
		}

		if data.Netflow.Personal != nil {
			if data.Netflow.Personal.Future != nil && len(data.Netflow.Personal.Future) > 0 {
				sb.WriteString("  Retail Futures:\n")
				for _, tf := range timeframes {
					if v, ok := data.Netflow.Personal.Future[tf]; ok {
						sb.WriteString(fmt.Sprintf("    %s: %s\n", tf, formatFlowValue(v)))
					}
				}
			}
			if data.Netflow.Personal.Spot != nil && len(data.Netflow.Personal.Spot) > 0 {
				sb.WriteString("  Retail Spot:\n")
				for _, tf := range timeframes {
					if v, ok := data.Netflow.Personal.Spot[tf]; ok {
						sb.WriteString(fmt.Sprintf("    %s: %s\n", tf, formatFlowValue(v)))
					}
				}
			}
		}
	}

	if indicators.EnableQuantOI && len(data.OI) > 0 {
		for exchange, oiData := range data.OI {
			if len(oiData.Delta) > 0 {
				sb.WriteString(fmt.Sprintf("Open Interest (%s):\n", exchange))
				for _, tf := range []string{"5m", "15m", "1h", "4h", "12h", "24h"} {
					if d, ok := oiData.Delta[tf]; ok {
						sb.WriteString(fmt.Sprintf("    %s: %+.4f%% (%s)\n", tf, d.OIDeltaPercent, formatFlowValue(d.OIDeltaValue)))
					}
				}
			}
		}
	}

	return sb.String()
}

func formatFlowValue(v float64) string {
	sign := ""
	if v >= 0 {
		sign = "+"
	}
	absV := v
	if absV < 0 {
		absV = -absV
	}
	if absV >= 1e9 {
		return fmt.Sprintf("%s%.2fB", sign, v/1e9)
	} else if absV >= 1e6 {
		return fmt.Sprintf("%s%.2fM", sign, v/1e6)
	} else if absV >= 1e3 {
		return fmt.Sprintf("%s%.2fK", sign, v/1e3)
	}
	return fmt.Sprintf("%s%.2f", sign, v)
}

func formatFloatSlice(values []float64) string {
	strValues := make([]string, len(values))
	for i, v := range values {
		strValues[i] = fmt.Sprintf("%.4f", v)
	}
	return "[" + strings.Join(strValues, ", ") + "]"
}

func timeframeToMillis(tf string) int64 {
	switch strings.ToLower(strings.TrimSpace(tf)) {
	case "1m":
		return int64(time.Minute / time.Millisecond)
	case "3m":
		return int64((3 * time.Minute) / time.Millisecond)
	case "5m":
		return int64((5 * time.Minute) / time.Millisecond)
	case "15m":
		return int64((15 * time.Minute) / time.Millisecond)
	case "30m":
		return int64((30 * time.Minute) / time.Millisecond)
	case "1h":
		return int64(time.Hour / time.Millisecond)
	case "2h":
		return int64((2 * time.Hour) / time.Millisecond)
	case "4h":
		return int64((4 * time.Hour) / time.Millisecond)
	case "6h":
		return int64((6 * time.Hour) / time.Millisecond)
	case "8h":
		return int64((8 * time.Hour) / time.Millisecond)
	case "12h":
		return int64((12 * time.Hour) / time.Millisecond)
	case "1d":
		return int64((24 * time.Hour) / time.Millisecond)
	case "3d":
		return int64((72 * time.Hour) / time.Millisecond)
	case "1w":
		return int64((7 * 24 * time.Hour) / time.Millisecond)
	default:
		return int64(time.Minute / time.Millisecond)
	}
}

// convertKlineBarsToKlines converts market.KlineBar slice to market.Kline slice
// for use with the analysis module.
func convertKlineBarsToKlines(timeframe string, bars []market.KlineBar) []market.Kline {
	stepMillis := timeframeToMillis(timeframe)
	klines := make([]market.Kline, len(bars))
	for i, bar := range bars {
		klines[i] = market.Kline{
			OpenTime:            bar.Time,
			Open:                bar.Open,
			High:                bar.High,
			Low:                 bar.Low,
			Close:               bar.Close,
			Volume:              bar.Volume,
			QuoteVolume:         bar.QuoteVolume,
			TakerBuyQuoteVolume: bar.TakerBuyQuoteVolume,
			CloseTime:           bar.Time + stepMillis, // Approximate close time based on timeframe
		}
	}
	return klines
}

// formatAnalysisResult formats the analysis result as JSON for injection into the prompt.
func formatAnalysisResult(result *analysis.AnalysisResult) string {
	if result == nil {
		return ""
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		logger.Warnf("Failed to marshal analysis result: %v", err)
		return ""
	}

	return string(jsonBytes)
}

// ============================================================================
// AI Response Parsing
// ============================================================================

func parseFullDecisionResponse(
	aiResponse string,
	priceLookup priceLookupFunc,
	accountEquity float64,
	btcEthLeverage, altcoinLeverage int,
	btcEthPosRatio, altcoinPosRatio float64,
	minPositionSize float64,
	enforceMinPositionSize bool,
	minRiskRewardRatio float64,
	exitPlanID string,
) (*FullDecision, error) {
	cotTrace := extractCoTTrace(aiResponse)

	decisions, err := extractDecisions(aiResponse)
	if err != nil {
		return &FullDecision{
			CoTTrace:  cotTrace,
			Decisions: []Decision{},
		}, fmt.Errorf("failed to extract decisions: %w", err)
	}

	if err := validateDecisions(
		decisions,
		priceLookup,
		accountEquity,
		btcEthLeverage,
		altcoinLeverage,
		btcEthPosRatio,
		altcoinPosRatio,
		minPositionSize,
		enforceMinPositionSize,
		minRiskRewardRatio,
		exitPlanID,
	); err != nil {
		return &FullDecision{
			CoTTrace:  cotTrace,
			Decisions: decisions,
		}, fmt.Errorf("decision validation failed: %w", err)
	}

	return &FullDecision{
		CoTTrace:  cotTrace,
		Decisions: decisions,
	}, nil
}

func extractCoTTrace(response string) string {
	if match := reReasoningTag.FindStringSubmatch(response); match != nil && len(match) > 1 {
		logger.Infof("[OK] Extracted reasoning chain using <reasoning> tag")
		return strings.TrimSpace(match[1])
	}

	if decisionIdx := strings.Index(response, "<decision>"); decisionIdx > 0 {
		logger.Infof("[OK] Extracted content before <decision> tag as reasoning chain")
		return strings.TrimSpace(response[:decisionIdx])
	}

	jsonStart := strings.Index(response, "[")
	if jsonStart > 0 {
		logger.Infof("[Info] Extracted reasoning chain using old format ([ character separator)")
		return strings.TrimSpace(response[:jsonStart])
	}

	return strings.TrimSpace(response)
}

func extractDecisions(response string) ([]Decision, error) {
	s := removeInvisibleRunes(response)
	s = strings.TrimSpace(s)
	s = fixMissingQuotes(s)

	var jsonPart string
	if match := reDecisionTag.FindStringSubmatch(s); match != nil && len(match) > 1 {
		jsonPart = strings.TrimSpace(match[1])
		logger.Infof("[OK] Extracted JSON using <decision> tag")
	} else {
		jsonPart = s
		logger.Infof("[Info] <decision> tag not found, searching JSON in full text")
	}

	jsonPart = fixMissingQuotes(jsonPart)

	if m := reJSONFence.FindStringSubmatch(jsonPart); m != nil && len(m) > 1 {
		jsonContent := strings.TrimSpace(m[1])
		jsonContent = compactArrayOpen(jsonContent)
		jsonContent = fixMissingQuotes(jsonContent)
		if err := validateJSONFormat(jsonContent); err != nil {
			return nil, fmt.Errorf("JSON format validation failed: %w\nJSON content: %s\nFull response:\n%s", err, jsonContent, response)
		}
		var decisions []Decision
		if err := json.Unmarshal([]byte(jsonContent), &decisions); err != nil {
			return nil, fmt.Errorf("JSON parsing failed: %w\nJSON content: %s", err, jsonContent)
		}
		return decisions, nil
	}

	jsonContent := strings.TrimSpace(reJSONArray.FindString(jsonPart))
	if jsonContent == "" {
		logger.Infof("[SafeFallback] AI didn't output JSON decision, entering safe wait mode")

		cotSummary := jsonPart
		if len(cotSummary) > 240 {
			cotSummary = cotSummary[:240] + "..."
		}

		fallbackDecision := Decision{
			Symbol:    "ALL",
			Action:    "wait",
			Reasoning: fmt.Sprintf("Model didn't output structured JSON decision, entering safe wait; summary: %s", cotSummary),
		}

		return []Decision{fallbackDecision}, nil
	}

	jsonContent = compactArrayOpen(jsonContent)
	jsonContent = fixMissingQuotes(jsonContent)

	if err := validateJSONFormat(jsonContent); err != nil {
		return nil, fmt.Errorf("JSON format validation failed: %w\nJSON content: %s\nFull response:\n%s", err, jsonContent, response)
	}

	var decisions []Decision
	if err := json.Unmarshal([]byte(jsonContent), &decisions); err != nil {
		return nil, fmt.Errorf("JSON parsing failed: %w\nJSON content: %s", err, jsonContent)
	}

	return decisions, nil
}

func fixMissingQuotes(jsonStr string) string {
	jsonStr = strings.ReplaceAll(jsonStr, "\u201c", "\"")
	jsonStr = strings.ReplaceAll(jsonStr, "\u201d", "\"")
	jsonStr = strings.ReplaceAll(jsonStr, "\u2018", "'")
	jsonStr = strings.ReplaceAll(jsonStr, "\u2019", "'")

	jsonStr = strings.ReplaceAll(jsonStr, "［", "[")
	jsonStr = strings.ReplaceAll(jsonStr, "］", "]")
	jsonStr = strings.ReplaceAll(jsonStr, "｛", "{")
	jsonStr = strings.ReplaceAll(jsonStr, "｝", "}")
	jsonStr = strings.ReplaceAll(jsonStr, "：", ":")
	jsonStr = strings.ReplaceAll(jsonStr, "，", ",")

	jsonStr = strings.ReplaceAll(jsonStr, "【", "[")
	jsonStr = strings.ReplaceAll(jsonStr, "】", "]")
	jsonStr = strings.ReplaceAll(jsonStr, "〔", "[")
	jsonStr = strings.ReplaceAll(jsonStr, "〕", "]")
	jsonStr = strings.ReplaceAll(jsonStr, "、", ",")

	jsonStr = strings.ReplaceAll(jsonStr, "　", " ")

	return jsonStr
}

func validateJSONFormat(jsonStr string) error {
	trimmed := strings.TrimSpace(jsonStr)

	if !reArrayHead.MatchString(trimmed) {
		if strings.HasPrefix(trimmed, "[") && !strings.Contains(trimmed[:min(20, len(trimmed))], "{") {
			return fmt.Errorf("not a valid decision array (must contain objects {}), actual content: %s", trimmed[:min(50, len(trimmed))])
		}
		return fmt.Errorf("JSON must start with [{ (whitespace allowed), actual: %s", trimmed[:min(20, len(trimmed))])
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func removeInvisibleRunes(s string) string {
	return reInvisibleRunes.ReplaceAllString(s, "")
}

func compactArrayOpen(s string) string {
	return reArrayOpenSpace.ReplaceAllString(strings.TrimSpace(s), "[{")
}

// ============================================================================
// Decision Validation
// ============================================================================

func makePriceLookup(ctx *Context) priceLookupFunc {
	prices := make(map[string]float64)
	if ctx == nil {
		return func(string) (float64, bool) { return 0, false }
	}

	for symbol, data := range ctx.MarketDataMap {
		if data == nil || data.CurrentPrice <= 0 {
			continue
		}
		norm := market.Normalize(symbol)
		prices[norm] = data.CurrentPrice
		if !strings.HasPrefix(strings.ToLower(norm), "xyz:") {
			prices[market.ToBinanceFuturesSymbol(norm)] = data.CurrentPrice
		}
	}

	return func(symbol string) (float64, bool) {
		norm := market.Normalize(symbol)
		if price, ok := prices[norm]; ok && price > 0 {
			return price, true
		}
		if !strings.HasPrefix(strings.ToLower(norm), "xyz:") {
			if price, ok := prices[market.ToBinanceFuturesSymbol(norm)]; ok && price > 0 {
				return price, true
			}
		}
		return 0, false
	}
}

func validateDecisions(
	decisions []Decision,
	priceLookup priceLookupFunc,
	accountEquity float64,
	btcEthLeverage, altcoinLeverage int,
	btcEthPosRatio, altcoinPosRatio float64,
	minPositionSize float64,
	enforceMinPositionSize bool,
	minRiskRewardRatio float64,
	exitPlanID string,
) error {
	for i := range decisions {
		if err := validateDecision(
			&decisions[i],
			priceLookup,
			accountEquity,
			btcEthLeverage,
			altcoinLeverage,
			btcEthPosRatio,
			altcoinPosRatio,
			minPositionSize,
			enforceMinPositionSize,
			minRiskRewardRatio,
			exitPlanID,
		); err != nil {
			return fmt.Errorf("decision #%d validation failed: %w", i+1, err)
		}
	}
	return nil
}

func validateDecision(
	d *Decision,
	priceLookup priceLookupFunc,
	accountEquity float64,
	btcEthLeverage, altcoinLeverage int,
	btcEthPosRatio, altcoinPosRatio float64,
	minPositionSize float64,
	enforceMinPositionSize bool,
	minRiskRewardRatio float64,
	exitPlanID string,
) error {
	validActions := map[string]bool{
		"open_long":   true,
		"open_short":  true,
		"close_long":  true,
		"close_short": true,
		"hold":        true,
		"wait":        true,
	}

	if !validActions[d.Action] {
		return fmt.Errorf("invalid action: %s", d.Action)
	}

	if d.Action == "open_long" || d.Action == "open_short" {
		maxLeverage := altcoinLeverage
		posRatio := altcoinPosRatio
		maxPositionValue := accountEquity * posRatio
		minOpeningAmount := minPositionSize
		if minOpeningAmount <= 0 {
			minOpeningAmount = 12.0
		}
		if d.Symbol == "BTCUSDT" || d.Symbol == "ETHUSDT" {
			maxLeverage = btcEthLeverage
			posRatio = btcEthPosRatio
			maxPositionValue = accountEquity * posRatio
			if minOpeningAmount < 60.0 {
				minOpeningAmount = 60.0
			}
		}
		if !enforceMinPositionSize {
			minOpeningAmount = 0
		}

		if d.Leverage <= 0 {
			return fmt.Errorf("leverage must be greater than 0: %d", d.Leverage)
		}
		if d.Leverage > maxLeverage {
			logger.Infof("[Leverage Fallback] %s leverage exceeded (%dx > %dx), auto-adjusting to limit %dx",
				d.Symbol, d.Leverage, maxLeverage, maxLeverage)
			d.Leverage = maxLeverage
		}
		if d.PositionSizeUSD <= 0 {
			return fmt.Errorf("position size must be greater than 0: %.2f", d.PositionSizeUSD)
		}

		tolerance := maxPositionValue * 0.01
		if minOpeningAmount > 0 && minOpeningAmount > maxPositionValue+tolerance {
			originalAction := d.Action
			d.Action = "wait"
			if strings.TrimSpace(d.Reasoning) == "" {
				d.Reasoning = fmt.Sprintf("Cannot open: min %.2f USDT > cap %.2f USDT (min position size enforcement)", minOpeningAmount, maxPositionValue)
			} else {
				d.Reasoning = fmt.Sprintf("%s | auto-wait: min %.2f USDT > cap %.2f USDT", strings.TrimSpace(d.Reasoning), minOpeningAmount, maxPositionValue)
			}
			d.Leverage = 0
			d.PositionSizeUSD = 0
			d.StopLoss = 0
			d.TakeProfit = 0
			d.ExitPlan = nil
			d.Confidence = 0
			d.RiskUSD = 0
			logger.Warnf("[Min Position Size] %s cannot open: min %.2f USDT > cap %.2f USDT, converting %s -> wait", d.Symbol, minOpeningAmount, maxPositionValue, originalAction)
			return nil
		}
		if d.PositionSizeUSD < minOpeningAmount {
			if minOpeningAmount > maxPositionValue+tolerance {
				originalAction := d.Action
				d.Action = "wait"
				if strings.TrimSpace(d.Reasoning) == "" {
					d.Reasoning = fmt.Sprintf("Cannot open: min %.2f USDT > cap %.2f USDT (min position size enforcement)", minOpeningAmount, maxPositionValue)
				} else {
					d.Reasoning = fmt.Sprintf("%s | auto-wait: min %.2f USDT > cap %.2f USDT", strings.TrimSpace(d.Reasoning), minOpeningAmount, maxPositionValue)
				}
				d.Leverage = 0
				d.PositionSizeUSD = 0
				d.StopLoss = 0
				d.TakeProfit = 0
				d.ExitPlan = nil
				d.Confidence = 0
				d.RiskUSD = 0
				logger.Warnf("[Min Position Size] %s cannot open: min %.2f USDT > cap %.2f USDT, converting %s -> wait", d.Symbol, minOpeningAmount, maxPositionValue, originalAction)
				return nil
			}
			logger.Infof("[Min Amount Fallback] %s opening amount too small (%.2f USDT), auto-adjusting to minimum %.2f USDT",
				d.Symbol, d.PositionSizeUSD, minOpeningAmount)
			d.PositionSizeUSD = minOpeningAmount
		}

		if d.PositionSizeUSD > maxPositionValue+tolerance {
			logger.Infof("[Position Size Cap] %s position value %.2f exceeds cap %.2f (= equity %.2f * %.1fx), auto-capping",
				d.Symbol, d.PositionSizeUSD, maxPositionValue, accountEquity, posRatio)
			d.PositionSizeUSD = maxPositionValue
		}
		if d.StopLoss <= 0 || d.TakeProfit <= 0 {
			return fmt.Errorf("stop loss and take profit must be greater than 0")
		}

		if d.Action == "open_long" {
			if d.StopLoss >= d.TakeProfit {
				return fmt.Errorf("for long positions, stop loss price must be less than take profit price")
			}
		} else {
			if d.StopLoss <= d.TakeProfit {
				return fmt.Errorf("for short positions, stop loss price must be greater than take profit price")
			}
		}

		if minRiskRewardRatio <= 0 {
			minRiskRewardRatio = 3.0
		}
		if priceLookup == nil {
			return fmt.Errorf("internal error: price lookup unavailable (cannot validate risk/reward)")
		}
		entryPrice, ok := priceLookup(d.Symbol)
		if !ok || entryPrice <= 0 {
			originalAction := d.Action
			d.Action = "wait"
			d.Leverage = 0
			d.PositionSizeUSD = 0
			d.StopLoss = 0
			d.TakeProfit = 0
			d.ExitPlan = nil
			d.Confidence = 0
			d.RiskUSD = 0

			if strings.TrimSpace(d.Reasoning) == "" {
				d.Reasoning = fmt.Sprintf("auto-wait: missing current price for %s (cannot validate risk/reward)", d.Symbol)
			} else {
				d.Reasoning = fmt.Sprintf("%s | auto-wait: missing current price for %s", strings.TrimSpace(d.Reasoning), d.Symbol)
			}
			logger.Warnf("[Missing Price] %s missing current price, converting %s -> wait", d.Symbol, originalAction)
			return nil
		}

		if d.Action == "open_long" {
			if entryPrice <= d.StopLoss {
				return fmt.Errorf("current price %.6f is below/at stop_loss %.6f for %s", entryPrice, d.StopLoss, d.Symbol)
			}
			if entryPrice >= d.TakeProfit {
				return fmt.Errorf("current price %.6f is above/at take_profit %.6f for %s", entryPrice, d.TakeProfit, d.Symbol)
			}
		} else {
			if entryPrice >= d.StopLoss {
				return fmt.Errorf("current price %.6f is above/at stop_loss %.6f for %s", entryPrice, d.StopLoss, d.Symbol)
			}
			if entryPrice <= d.TakeProfit {
				return fmt.Errorf("current price %.6f is below/at take_profit %.6f for %s", entryPrice, d.TakeProfit, d.Symbol)
			}
		}

		var riskPercent, rewardPercent, riskRewardRatio float64
		if d.Action == "open_long" {
			riskPercent = (entryPrice - d.StopLoss) / entryPrice * 100
			rewardPercent = (d.TakeProfit - entryPrice) / entryPrice * 100
			if riskPercent > 0 {
				riskRewardRatio = rewardPercent / riskPercent
			}
		} else {
			riskPercent = (d.StopLoss - entryPrice) / entryPrice * 100
			rewardPercent = (entryPrice - d.TakeProfit) / entryPrice * 100
			if riskPercent > 0 {
				riskRewardRatio = rewardPercent / riskPercent
			}
		}

		if riskRewardRatio < minRiskRewardRatio {
			requiredEntry := (d.TakeProfit + minRiskRewardRatio*d.StopLoss) / (1.0 + minRiskRewardRatio)
			originalAction := d.Action

			d.Action = "wait"
			d.Leverage = 0
			d.PositionSizeUSD = 0
			d.StopLoss = 0
			d.TakeProfit = 0
			d.ExitPlan = nil
			d.Confidence = 0
			d.RiskUSD = 0

			if strings.TrimSpace(d.Reasoning) == "" {
				if originalAction == "open_long" {
					d.Reasoning = fmt.Sprintf("auto-wait: RR %.2f < %.2f at %.6f; need entry <= %.6f", riskRewardRatio, minRiskRewardRatio, entryPrice, requiredEntry)
				} else {
					d.Reasoning = fmt.Sprintf("auto-wait: RR %.2f < %.2f at %.6f; need entry >= %.6f", riskRewardRatio, minRiskRewardRatio, entryPrice, requiredEntry)
				}
			} else {
				if originalAction == "open_long" {
					d.Reasoning = fmt.Sprintf("%s | auto-wait: RR %.2f < %.2f at %.6f; need entry <= %.6f", strings.TrimSpace(d.Reasoning), riskRewardRatio, minRiskRewardRatio, entryPrice, requiredEntry)
				} else {
					d.Reasoning = fmt.Sprintf("%s | auto-wait: RR %.2f < %.2f at %.6f; need entry >= %.6f", strings.TrimSpace(d.Reasoning), riskRewardRatio, minRiskRewardRatio, entryPrice, requiredEntry)
				}
			}

			return nil
		}

		if err := validateExitPlan(d, exitPlanID); err != nil {
			return err
		}
	}

	return nil
}

// ============================================================================
// Helper Functions
// ============================================================================

func toNofxOSLang(lang Language) nofxos.Language {
	if lang == LangChinese {
		return nofxos.LangChinese
	}
	return nofxos.LangEnglish
}

// detectLanguage detects language from text content
// Returns LangChinese if text contains Chinese characters, otherwise LangEnglish
func detectLanguage(text string) Language {
	for _, r := range text {
		if r >= 0x4E00 && r <= 0x9FFF {
			return LangChinese
		}
	}
	return LangEnglish
}
