package kernel

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"nofx/analysis/vision"
	"nofx/logger"
	"nofx/market"
	"nofx/mcp"
	"nofx/store"
)

type VisionNote struct {
	Symbol string
	Note   string
}

func saveVisionChartImage(traderID string, cycleNumber int, symbol string, timeframe string, pngBytes []byte) (*store.VisionImageMeta, error) {
	traderID = strings.TrimSpace(traderID)
	if traderID == "" || cycleNumber <= 0 || len(pngBytes) == 0 {
		return nil, nil
	}

	symbol = market.Normalize(symbol)
	timeframe = strings.ToLower(strings.TrimSpace(timeframe))
	if symbol == "" || timeframe == "" {
		return nil, nil
	}

	filename := fmt.Sprintf("%s_%s.png", symbol, timeframe)
	relPath := filepath.ToSlash(filepath.Join("vision_images", traderID, fmt.Sprintf("%d", cycleNumber), filename))
	diskPath := filepath.Join("data", filepath.FromSlash(relPath))

	if err := os.MkdirAll(filepath.Dir(diskPath), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(diskPath, pngBytes, 0o644); err != nil {
		return nil, err
	}

	return &store.VisionImageMeta{
		Name:      filename,
		Symbol:    symbol,
		Timeframe: timeframe,
		Path:      relPath,
		SizeBytes: int64(len(pngBytes)),
	}, nil
}

const (
	visionKeepCyclesEnv     = "VISION_IMAGE_KEEP_CYCLES"
	visionKeepCyclesDefault = 5
)

func visionKeepCycles() int {
	raw := strings.TrimSpace(os.Getenv(visionKeepCyclesEnv))
	if raw == "" {
		return visionKeepCyclesDefault
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return visionKeepCyclesDefault
	}
	return n
}

type visionCycleDir struct {
	cycle int
	path  string
}

func pruneOldVisionCyclesForTrader(traderID string, keep int) (int, error) {
	traderID = strings.TrimSpace(traderID)
	if traderID == "" || keep <= 0 {
		return 0, nil
	}

	baseDir := filepath.Join("data", "vision_images")
	traderDir := filepath.Join(baseDir, traderID)

	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return 0, err
	}
	absTrader, err := filepath.Abs(traderDir)
	if err != nil {
		return 0, err
	}
	sep := string(os.PathSeparator)
	if absTrader != absBase && !strings.HasPrefix(absTrader, absBase+sep) {
		return 0, fmt.Errorf("refusing to prune outside base dir: %s", absTrader)
	}

	return pruneOldVisionCyclesInDir(traderDir, keep)
}

func pruneOldVisionCyclesInDir(traderDir string, keep int) (int, error) {
	if keep <= 0 || strings.TrimSpace(traderDir) == "" {
		return 0, nil
	}

	entries, err := os.ReadDir(traderDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	cycles := make([]visionCycleDir, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := strings.TrimSpace(e.Name())
		n, err := strconv.Atoi(name)
		if err != nil || n <= 0 {
			continue
		}
		cycles = append(cycles, visionCycleDir{
			cycle: n,
			path:  filepath.Join(traderDir, name),
		})
	}
	if len(cycles) <= keep {
		return 0, nil
	}

	sort.Slice(cycles, func(i, j int) bool { return cycles[i].cycle > cycles[j].cycle })

	deleted := 0
	for i := keep; i < len(cycles); i++ {
		if err := os.RemoveAll(cycles[i].path); err != nil {
			return deleted, err
		}
		deleted++
	}
	return deleted, nil
}

func normalizeVisionConfig(cfg store.VisionConfig) store.VisionConfig {
	if cfg.MaxSymbols <= 0 {
		cfg.MaxSymbols = 5
	}
	if len(cfg.Timeframes) == 0 {
		cfg.Timeframes = []string{"1h", "15m"}
	}
	if cfg.ImageWidth <= 0 {
		cfg.ImageWidth = 1600
	}
	if cfg.ImageHeight <= 0 {
		cfg.ImageHeight = 1396
	}
	// Migrate legacy defaults (old UI default was 1024x640).
	if cfg.ImageWidth == 1024 && cfg.ImageHeight == 640 {
		cfg.ImageWidth = 1600
		cfg.ImageHeight = 1396
		if cfg.Indicators.ShowMACD != nil && cfg.Indicators.ShowSqueeze != nil {
			if !*cfg.Indicators.ShowMACD && *cfg.Indicators.ShowSqueeze {
				cfg.Indicators.ShowMACD = boolPtr(true)
				cfg.Indicators.ShowSqueeze = boolPtr(false)
			}
		}
	}
	if cfg.RenderConcurrency <= 0 {
		cfg.RenderConcurrency = 1
	}
	if cfg.Indicators.ShowEMA == nil {
		cfg.Indicators.ShowEMA = boolPtr(true)
	}
	if cfg.Indicators.ShowMACD == nil {
		cfg.Indicators.ShowMACD = boolPtr(true)
	}
	if cfg.Indicators.ShowWaveTrend == nil {
		cfg.Indicators.ShowWaveTrend = boolPtr(true)
	}
	if cfg.Indicators.ShowSqueeze == nil {
		cfg.Indicators.ShowSqueeze = boolPtr(false)
	}
	if cfg.Indicators.ShowDivergence == nil {
		cfg.Indicators.ShowDivergence = boolPtr(true)
	}
	return cfg
}

func boolPtr(v bool) *bool { return &v }

func boolOrDefault(v *bool, def bool) bool {
	if v == nil {
		return def
	}
	return *v
}

func clientSupportsVision(c mcp.AIClient) bool {
	switch v := c.(type) {
	case *mcp.SplitClient:
		visionOK := v.VisionClient() != nil && clientSupportsVision(v.VisionClient())
		decisionOK := v.DecisionClient() != nil && clientSupportsVision(v.DecisionClient())
		return visionOK || decisionOK
	case *mcp.QwenClient, *mcp.KimiClient, *mcp.GrokClient:
		// These clients use OpenAI-compatible payloads (including image_url parts).
		// Whether a given model supports vision depends on the model name, but we allow attempts here
		// and rely on per-symbol error handling to avoid blocking the main decision cycle.
		return true
	case *mcp.Client:
		// mcp.Client can encode image parts using OpenAI-compatible "image_url" content.
		return v.Provider == mcp.ProviderCustom ||
			v.Provider == mcp.ProviderOpenAI ||
			v.Provider == mcp.ProviderQwen ||
			v.Provider == mcp.ProviderKimi ||
			v.Provider == mcp.ProviderGrok
	case *mcp.OpenAIClient, *mcp.GeminiClient, *mcp.ClaudeClient:
		return true
	default:
		return false
	}
}

func (e *StrategyEngine) buildVisionSystemPrompt() string {
	toggles := e.config.PromptToggles
	if !boolOrDefault(toggles.UseVerboseVisionSystemPrompt, true) {
		return strings.TrimSpace(`
You are a strict chart-reading assistant.

Rules:
- Only describe what is directly visible in the images.
- No predictions or recommendations; do NOT output any trading actions.
- If unclear/not visible, say so.
- Plain text only (no code blocks). Keep it concise.

中文（同样只做读图要点，不做交易结论）：
- 只描述图上可见信息，不做预测/建议/开平仓指令。
- 不清楚就说不清楚。
- 纯文本，尽量简短。`)
	}
	return e.buildVisionSystemPromptBilingual()
}

func (e *StrategyEngine) buildVisionUserText(ctx *Context, symbol string, timeframes []string, data *market.Data) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Symbol: %s\n", symbol))
	sb.WriteString("Task: read the attached charts and output chart-reading notes only.\n\n")

	if data == nil && ctx != nil {
		if ctxData, ok := ctx.MarketDataMap[symbol]; ok && ctxData != nil {
			data = ctxData
		}
	}

	if data != nil {
		sb.WriteString("Text supplement (do not override the chart):\n")
		sb.WriteString(e.formatMarketData(data))
		sb.WriteString("\n")
	} else {
		sb.WriteString("Text supplement: missing market data for this symbol.\n\n")
	}

	sb.WriteString("Charts provided:\n")
	for _, tf := range timeframes {
		sb.WriteString(fmt.Sprintf("- %s\n", tf))
	}
	sb.WriteString("\nKeep it concise.\n")
	return sb.String()
}

// buildVisionSystemPromptBilingual returns a bilingual (EN+CN) prompt for the vision/chart-reading layer,
// so the same guidance applies regardless of the trader's main prompt language.
func (e *StrategyEngine) buildVisionSystemPromptBilingual() string {
	return strings.TrimSpace(`
You are a strict chart-reading assistant. You will see one or more charts (commonly 1h and 15m) that may include:
- Candles
- EMA overlays (if enabled)
- Volume panel
- CVD panel (if enabled; shown as ΔCVD candles per bar: net taker buy/sell quote delta, in quote currency)
- MACD panel (hist + DIF/DEA) (if enabled)
- WT+MFI panel (if enabled)
- Divergence dots on the main chart (if enabled)

Rules:
- Only describe what is directly visible in the images.
- No directional judgement, predictions, or recommendations. Do NOT use words like bullish/bearish, uptrend/downtrend, likely to rise/fall, etc.
- Do not guess or invent numbers/price levels. Only mention specific prices if clearly visible; otherwise use relative references (recent swing low/high, prior high/low, near EMA200, etc.).
- If something is not visible/unclear, explicitly say so.
- This stage outputs chart-reading notes only. Do NOT output trading decision JSON or any open/close instructions.

ΔCVD usage principles (for later decision stage; still chart-reading only here):
- Priority: structure/levels > volume > ΔCVD (ΔCVD is confirmation/divergence clue only).
- If structure and ΔCVD conflict: explicitly note the conflict; do not rationalize it away.
- ΔCVD is exchange/microstructure-dependent and can be noisy (missing data, delays, wash trading, order splitting); say "unclear" when appropriate.

中文参考（仍然只做读图要点，不做交易结论）：
- 优先级：结构/关键价位 > 成交量/放量位置 > ΔCVD（ΔCVD 只作确认或背离线索）。
- 当结构与 ΔCVD 冲突：明确写“冲突/不一致”，不要强行解释。
- ΔCVD 可能受数据缺失/延迟/刷量/拆单影响；不清楚就说不确定。

Plain text only (no code blocks). Keep it concise.`)
}

func (e *StrategyEngine) collectVisionNotes(ctx *Context, mcpClient mcp.AIClient, traderID string, cycleNumber int) (notes []VisionNote, selectedSymbols []string, images []store.VisionImageMeta) {
	if e == nil || ctx == nil || mcpClient == nil {
		return nil, nil, nil
	}
	if !e.config.Vision.Enabled || !clientSupportsVision(mcpClient) {
		return nil, nil, nil
	}

	vcfg := normalizeVisionConfig(e.config.Vision)
	selectedSymbols = selectVisionSymbols(ctx, vcfg.MaxSymbols, vcfg.Timeframes)
	if len(selectedSymbols) == 0 {
		return nil, nil, nil
	}

	system := e.buildVisionSystemPrompt()
	renderCfg := vision.RenderConfig{
		Width:    vcfg.ImageWidth,
		Height:   vcfg.ImageHeight,
		MaxBars:  300,
		PlotBars: 100,
		Indicators: vision.IndicatorRenderConfig{
			ShowEMA:        boolOrDefault(vcfg.Indicators.ShowEMA, true),
			ShowMACD:       boolOrDefault(vcfg.Indicators.ShowMACD, true),
			ShowCVD:        boolOrDefault(vcfg.Indicators.ShowCVD, true),
			ShowWaveTrend:  boolOrDefault(vcfg.Indicators.ShowWaveTrend, true),
			ShowSqueeze:    boolOrDefault(vcfg.Indicators.ShowSqueeze, true),
			ShowDivergence: boolOrDefault(vcfg.Indicators.ShowDivergence, true),
		},
	}

	type result struct {
		idx   int
		sym   string
		note  string
		metas []store.VisionImageMeta
		err   error
	}

	workerCount := vcfg.RenderConcurrency
	if workerCount > len(selectedSymbols) {
		workerCount = len(selectedSymbols)
	}
	if workerCount <= 0 {
		workerCount = 1
	}

	jobs := make(chan int)
	results := make(chan result, len(selectedSymbols))
	var wg sync.WaitGroup

	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				sym := selectedSymbols[idx]
				note, metas, err := e.callVisionForSymbol(ctx, mcpClient, system, traderID, cycleNumber, sym, vcfg.Timeframes, renderCfg)
				results <- result{idx: idx, sym: sym, note: strings.TrimSpace(note), metas: metas, err: err}
			}
		}()
	}

	go func() {
		for i := range selectedSymbols {
			jobs <- i
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	collected := make([]result, 0, len(selectedSymbols))
	for r := range results {
		if len(r.metas) > 0 {
			images = append(images, r.metas...)
		}
		if r.err != nil {
			logger.Infof("⚠️  vision note failed for %s: %v", r.sym, r.err)
			continue
		}
		if r.note == "" {
			continue
		}
		collected = append(collected, r)
	}
	sort.Slice(collected, func(i, j int) bool { return collected[i].idx < collected[j].idx })

	for _, r := range collected {
		notes = append(notes, VisionNote{Symbol: r.sym, Note: r.note})
	}

	if deleted, err := pruneOldVisionCyclesForTrader(traderID, visionKeepCycles()); err != nil {
		logger.Infof("⚠️  vision image cleanup failed for %s: %v", traderID, err)
	} else if deleted > 0 {
		logger.Infof("🧹  pruned %d old vision cycle(s) for %s", deleted, traderID)
	}

	return notes, selectedSymbols, images
}

func selectVisionSymbols(ctx *Context, max int, _ []string) []string {
	if ctx == nil || max <= 0 {
		return nil
	}

	positionSymbols := make(map[string]bool, len(ctx.Positions))
	for _, pos := range ctx.Positions {
		positionSymbols[market.Normalize(pos.Symbol)] = true
	}

	out := make([]string, 0, max)
	seen := make(map[string]bool, max)

	for _, coin := range ctx.CandidateCoins {
		sym := market.Normalize(coin.Symbol)
		if sym == "" {
			continue
		}
		if seen[sym] {
			continue
		}
		if positionSymbols[sym] {
			continue
		}

		seen[sym] = true
		out = append(out, sym)
		if len(out) >= max {
			break
		}
	}

	return out
}

func (e *StrategyEngine) callVisionForSymbol(ctx *Context, mcpClient mcp.AIClient, systemPrompt, traderID string, cycleNumber int, symbol string, timeframes []string, renderCfg vision.RenderConfig) (string, []store.VisionImageMeta, error) {
	var data *market.Data
	if ctx != nil {
		if ctxData, ok := ctx.MarketDataMap[symbol]; ok {
			data = ctxData
		}
	}
	if data == nil {
		// Vision charts should not depend on the strategy engine's market-data filters.
		// Try to fetch a minimal market snapshot for text supplement; chart klines are fetched per-timeframe below.
		fetched, err := market.GetWithTimeframes(symbol, timeframes, "", 60)
		if err == nil {
			data = fetched
		}
	}

	parts := make([]mcp.ContentPart, 0, 1+len(timeframes)*2)
	parts = append(parts, mcp.NewTextPart(e.buildVisionUserText(ctx, symbol, timeframes, data)))
	metas := make([]store.VisionImageMeta, 0, len(timeframes))

	for _, tf := range timeframes {
		tf = strings.TrimSpace(tf)
		if tf == "" {
			continue
		}
		var tfData *market.TimeframeSeriesData
		if data != nil && data.TimeframeData != nil {
			tfData = data.TimeframeData[tf]
		}

		// For chart alignment, fetch fresh klines with enough history (independent of strategy primary_count).
		fetched, fetchErr := market.GetKlines(symbol, tf, renderCfg.MaxBars)
		if fetchErr == nil && len(fetched) >= 2 {
			bars := make([]market.KlineBar, 0, len(fetched))
			for _, k := range fetched {
				bars = append(bars, market.KlineBar{
					Time:                k.OpenTime,
					Open:                k.Open,
					High:                k.High,
					Low:                 k.Low,
					Close:               k.Close,
					Volume:              k.Volume,
					IsClosed:            k.IsClosed,
					QuoteVolume:         k.QuoteVolume,
					TakerBuyQuoteVolume: k.TakerBuyQuoteVolume,
				})
			}
			tfData = &market.TimeframeSeriesData{
				Timeframe: tf,
				Klines:    bars,
			}
		}
		if tfData == nil {
			return "", nil, fmt.Errorf("missing timeframe data %s for %s", tf, symbol)
		}

		pngBytes, err := vision.RenderTimeframeChartPNG(symbol, tf, tfData, renderCfg)
		if err != nil {
			return "", nil, err
		}
		if meta, saveErr := saveVisionChartImage(traderID, cycleNumber, symbol, tf, pngBytes); saveErr != nil {
			logger.Infof("⚠️  vision image save failed for %s %s: %v", symbol, tf, saveErr)
		} else if meta != nil {
			metas = append(metas, *meta)
		}
		dataURI := vision.PNGToDataURI(pngBytes)
		parts = append(parts, mcp.NewTextPart(fmt.Sprintf("Chart: %s", tf)))
		parts = append(parts, mcp.NewImagePartDataURI(dataURI))
	}

	req := mcp.NewRequestBuilder().
		WithSystemPrompt(systemPrompt).
		WithUserParts(parts).
		WithTemperature(0.2).
		WithMaxTokens(500).
		MustBuild()

	out, err := mcpClient.CallWithRequest(req)
	return out, metas, err
}
