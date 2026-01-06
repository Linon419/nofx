package vision

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"sort"
	"strings"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"nofx/analysis/indicator"
	"nofx/market"
)

type IndicatorRenderConfig struct {
	ShowEMA        bool
	ShowMACD       bool
	ShowCVD        bool
	ShowWaveTrend  bool
	ShowSqueeze    bool
	ShowDivergence bool
}

type RenderConfig struct {
	Width  int
	Height int
	// MaxBars is the maximum bars rendered (from the end). Defaults to 200.
	MaxBars int
	// PlotBars is the number of bars plotted in the visible window (from the end).
	// Defaults to 100 to match BRALE analysis_slice.
	PlotBars int
	// DropTailBars drops the latest N bars before plotting (after MaxBars truncation).
	// Defaults to 1 to match BRALE slice_drop_tail.
	DropTailBars int
	// Indicators controls which overlays/panels to render.
	Indicators IndicatorRenderConfig
}

func DefaultRenderConfig() RenderConfig {
	return RenderConfig{
		Width:        1600,
		Height:       1396,
		MaxBars:      300,
		PlotBars:     100,
		DropTailBars: 1,
		Indicators: IndicatorRenderConfig{
			ShowEMA:        true,
			ShowMACD:       true,
			ShowCVD:        true,
			ShowWaveTrend:  true,
			ShowSqueeze:    false,
			ShowDivergence: true,
		},
	}
}

func PNGToDataURI(pngBytes []byte) string {
	if len(pngBytes) == 0 {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)
}

func RenderTimeframeChartPNG(symbol, timeframe string, tf *market.TimeframeSeriesData, cfg RenderConfig) ([]byte, error) {
	if tf == nil || len(tf.Klines) == 0 {
		return nil, fmt.Errorf("missing kline data for %s %s", symbol, timeframe)
	}

	def := DefaultRenderConfig()
	if cfg.Width <= 0 {
		cfg.Width = def.Width
	}
	if cfg.Height <= 0 {
		cfg.Height = def.Height
	}
	if cfg.MaxBars <= 0 {
		cfg.MaxBars = def.MaxBars
	}
	if cfg.PlotBars <= 0 {
		cfg.PlotBars = def.PlotBars
	}
	if cfg.DropTailBars == 0 {
		cfg.DropTailBars = def.DropTailBars
	}
	if cfg.DropTailBars < 0 {
		cfg.DropTailBars = 0
	}

	history := tf.Klines
	if len(history) > cfg.MaxBars {
		history = history[len(history)-cfg.MaxBars:]
	}

	end := len(history)
	if cfg.DropTailBars > 0 && end > cfg.DropTailBars+1 {
		end -= cfg.DropTailBars
	}
	if end < 2 {
		return nil, fmt.Errorf("not enough bars for %s %s (n=%d)", symbol, timeframe, end)
	}
	start := 0
	if cfg.PlotBars > 0 && end > cfg.PlotBars {
		start = end - cfg.PlotBars
	}

	klinesAll := history[:end]
	klinesPlot := history[start:end]
	nAll := len(klinesAll)
	nPlot := len(klinesPlot)
	if nPlot < 2 {
		return nil, fmt.Errorf("not enough plot bars for %s %s (n=%d)", symbol, timeframe, nPlot)
	}

	closesAll := make([]float64, nAll)
	opensAll := make([]float64, nAll)
	highsAll := make([]float64, nAll)
	lowsAll := make([]float64, nAll)
	volumesAll := make([]float64, nAll)
	for i, b := range klinesAll {
		opensAll[i] = b.Open
		closesAll[i] = b.Close
		highsAll[i] = b.High
		lowsAll[i] = b.Low
		volumesAll[i] = b.Volume
	}

	closes := closesAll[start:end]
	opens := opensAll[start:end]
	volumes := volumesAll[start:end]

	hasWT := cfg.Indicators.ShowWaveTrend
	hasMACD := cfg.Indicators.ShowMACD
	hasCVD := cfg.Indicators.ShowCVD && hasAnyCVDData(klinesAll)
	// Squeeze is supported as an optional overlay/panel, but BRALE-style default is off.
	hasSqueezePanel := cfg.Indicators.ShowSqueeze

	// Layout (BRALE-ish: Price + Volume + CVD + MACD + WT+MFI)
	left := 88
	right := 18
	headerH := 54
	bottom := 16
	panelGap := 14

	plotW := float64(cfg.Width - left - right)
	if plotW <= 0 {
		return nil, fmt.Errorf("invalid image width")
	}

	availableH := cfg.Height - headerH - bottom
	if availableH <= 50 {
		return nil, fmt.Errorf("invalid image height")
	}

	// Panel layout: Price + Volume + (optional) CVD + (optional) MACD + (optional) WT+MFI.
	hasVolume := true
	subCount := 0
	if hasVolume {
		subCount++
	}
	if hasCVD {
		subCount++
	}
	if hasMACD {
		subCount++
	}
	if hasWT {
		subCount++
	}

	gapCount := subCount // gaps between price and each sub-panel
	remainingH := availableH - panelGap*gapCount
	if remainingH <= 200 {
		return nil, fmt.Errorf("invalid image height for panels")
	}

	wPrice := 52.0
	wVol := 18.0
	wCVD := 15.0
	wMACD := 15.0
	wWT := 15.0
	if !hasVolume {
		wVol = 0
	}
	if !hasCVD {
		wCVD = 0
	}
	if !hasMACD {
		wMACD = 0
	}
	if !hasWT {
		wWT = 0
	}
	sumW := wPrice + wVol + wCVD + wMACD + wWT
	if sumW <= 0 {
		sumW = 1
	}

	priceH := int(math.Round(float64(remainingH) * wPrice / sumW))
	volH := int(math.Round(float64(remainingH) * wVol / sumW))
	cvdH := int(math.Round(float64(remainingH) * wCVD / sumW))
	macdH := int(math.Round(float64(remainingH) * wMACD / sumW))
	wtH := remainingH - priceH - volH - cvdH - macdH
	if wtH < 0 {
		wtH = 0
	}

	pricePanel := rect(left, headerH, cfg.Width-right, headerH+priceH)
	nextY := pricePanel.Max.Y + panelGap

	volumePanel := image.Rectangle{}
	if hasVolume {
		volumePanel = rect(left, nextY, cfg.Width-right, nextY+volH)
		nextY = volumePanel.Max.Y + panelGap
	}
	cvdPanel := image.Rectangle{}
	if hasCVD {
		cvdPanel = rect(left, nextY, cfg.Width-right, nextY+cvdH)
		nextY = cvdPanel.Max.Y + panelGap
	}
	macdPanel := image.Rectangle{}
	if hasMACD {
		macdPanel = rect(left, nextY, cfg.Width-right, nextY+macdH)
		nextY = macdPanel.Max.Y + panelGap
	}
	wtPanel := image.Rectangle{}
	if hasWT {
		wtPanel = rect(left, nextY, cfg.Width-right, nextY+wtH)
	}

	dx := plotW / float64(nPlot-1)
	bodyW := int(math.Max(1, math.Min(dx*0.55, 10)))

	indexToX := func(i int) int {
		x := float64(left) + float64(i)*dx
		return clampInt(int(math.Round(x)), left, cfg.Width-right)
	}

	// Prepare indicators
	var ema21, ema55, ema100, ema200 []float64
	if cfg.Indicators.ShowEMA {
		ema21 = emaSeries(closesAll, 21)[start:end]
		ema55 = emaSeries(closesAll, 55)[start:end]
		ema100 = emaSeries(closesAll, 100)[start:end]
		ema200 = emaSeries(closesAll, 200)[start:end]
	}

	var macdLine, macdSignal, macdHist []float64
	if hasMACD {
		allLine, allSig, allHist := macdSeries(closesAll, 12, 26, 9)
		macdLine = allLine[start:end]
		macdSignal = allSig[start:end]
		macdHist = allHist[start:end]
	}

	var cvdSeries []float64
	if hasCVD {
		allCVD := cvdSeriesFromBars(klinesAll)
		if len(allCVD) >= end {
			cvdSeries = allCVD[start:end]
			if len(cvdSeries) > 0 {
				base := cvdSeries[0]
				for i := range cvdSeries {
					cvdSeries[i] -= base
				}
			}
		}
	}

	var wtSeries []float64
	var squeezeSeries squeezeSeriesData
	var divEvents []indicator.DivergenceChartEvent
	var divResult *indicator.DivergenceResult
	if cfg.Indicators.ShowWaveTrend || cfg.Indicators.ShowSqueeze || cfg.Indicators.ShowDivergence {
		k := toMarketKlines(klinesAll)
		if cfg.Indicators.ShowWaveTrend {
			allWT := indicator.CalculateWaveTrendSeriesForChart(k, indicator.DefaultWaveTrendConfig())
			if len(allWT) >= end {
				wtSeries = allWT[start:end]
			}
		}
		if cfg.Indicators.ShowSqueeze {
			squeezeSeries = computeSqueezeSeries(closesAll, highsAll, lowsAll).slice(start, end)
		}
		if cfg.Indicators.ShowDivergence {
			divCfg := indicator.DefaultDivergenceConfig()
			divResult = indicator.CalculateDivergence(k, divCfg)
			divEvents = indicator.CalculateDivergenceChartEventsInRange(k, divCfg, start, end)
		}
	}

	// Colors (BRALE-ish dark blue)
	bg := color.RGBA{R: 6, G: 10, B: 20, A: 255}       // #060A14
	panelBg := color.RGBA{R: 9, G: 14, B: 28, A: 255}  // #090E1C
	grid := color.RGBA{R: 30, G: 41, B: 59, A: 255}    // #1E293B
	text := color.RGBA{R: 234, G: 236, B: 239, A: 255} // #EAECEF
	muted := color.RGBA{R: 132, G: 142, B: 156, A: 255}

	upColor := color.RGBA{R: 14, G: 203, B: 129, A: 255}  // Binance green
	downColor := color.RGBA{R: 246, G: 70, B: 93, A: 255} // Binance red
	wickColor := color.RGBA{R: 107, G: 114, B: 128, A: 255}

	ema21C := color.RGBA{R: 125, G: 211, B: 252, A: 255}  // light blue
	ema55C := color.RGBA{R: 110, G: 231, B: 183, A: 255}  // light green
	ema100C := color.RGBA{R: 244, G: 114, B: 182, A: 255} // pink
	ema200C := color.RGBA{R: 34, G: 211, B: 238, A: 255}  // cyan

	// User-specified EMA palette (match BRALE preference):
	// EMA21 red, EMA55 yellow, EMA100 green, EMA200 white.
	ema21C = color.RGBA{R: 246, G: 70, B: 93, A: 255}    // red
	ema55C = color.RGBA{R: 240, G: 185, B: 11, A: 255}   // yellow
	ema100C = color.RGBA{R: 14, G: 203, B: 129, A: 255}  // green
	ema200C = color.RGBA{R: 234, G: 236, B: 239, A: 255} // white-ish

	img := image.NewRGBA(image.Rect(0, 0, cfg.Width, cfg.Height))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)
	draw.Draw(img, pricePanel, &image.Uniform{C: panelBg}, image.Point{}, draw.Src)
	if volumePanel.Dx() > 0 && volumePanel.Dy() > 0 {
		draw.Draw(img, volumePanel, &image.Uniform{C: panelBg}, image.Point{}, draw.Src)
	}
	if cvdPanel.Dx() > 0 && cvdPanel.Dy() > 0 {
		draw.Draw(img, cvdPanel, &image.Uniform{C: panelBg}, image.Point{}, draw.Src)
	}
	if macdPanel.Dx() > 0 && macdPanel.Dy() > 0 {
		draw.Draw(img, macdPanel, &image.Uniform{C: panelBg}, image.Point{}, draw.Src)
	}
	if wtPanel.Dx() > 0 && wtPanel.Dy() > 0 {
		draw.Draw(img, wtPanel, &image.Uniform{C: panelBg}, image.Point{}, draw.Src)
	}

	// Header: title + summary (BRALE-ish)
	title := fmt.Sprintf("%s %s", formatSymbolPair(symbol), timeframe)
	drawText(img, left, 20, title, text)

	summary := buildBraleHeaderSummary(wtSeries, macdHist, divResult, cfg.Indicators.ShowDivergence)
	if summary != "" {
		drawText(img, left, 38, summary, muted)
	}

	drawBraleLegendRow(img, left+360, 20, cfg.Width-right, timeframe, cfg.Indicators, muted)

	// Price panel bounds
	minP, maxP := priceBounds(klinesPlot)
	if minP == maxP {
		maxP = minP * 1.01
	}
	pad := (maxP - minP) * 0.06
	if pad <= 0 {
		pad = math.Max(1, math.Abs(maxP)*0.01)
	}
	minP -= pad
	maxP += pad

	// Reserve space in the price panel for time labels + range slider (BRALE-ish).
	pricePlot := pricePanel
	timeLabelH := 16
	sliderH := 18
	reserve := timeLabelH + sliderH + 6
	if pricePlot.Dy() > reserve+60 {
		pricePlot.Max.Y = pricePanel.Max.Y - reserve
	}

	priceToY := func(p float64) int {
		y := float64(pricePlot.Min.Y) + (maxP-p)/(maxP-minP)*float64(pricePlot.Dy())
		return clampInt(int(math.Round(y)), pricePlot.Min.Y, pricePlot.Max.Y)
	}

	// Grid + Y labels (price)
	for j := 0; j <= 5; j++ {
		y := pricePlot.Min.Y + int(float64(j)/5.0*float64(pricePlot.Dy()))
		drawLine(img, pricePanel.Min.X, y, pricePanel.Max.X, y, grid)
		p := maxP - float64(j)/5.0*(maxP-minP)
		drawText(img, 6, y+4, formatAxisPrice(p), muted)
	}

	// Vertical time grid + time labels (use close-time like BRALE: open + duration - 1ms).
	overallBottom := pricePanel.Max.Y
	if volumePanel.Max.Y > overallBottom {
		overallBottom = volumePanel.Max.Y
	}
	if cvdPanel.Max.Y > overallBottom {
		overallBottom = cvdPanel.Max.Y
	}
	if macdPanel.Max.Y > overallBottom {
		overallBottom = macdPanel.Max.Y
	}
	if wtPanel.Max.Y > overallBottom {
		overallBottom = wtPanel.Max.Y
	}
	dur, _ := market.TFDuration(timeframe)
	durMs := dur.Milliseconds()
	if durMs <= 0 {
		durMs = 0
	}
	tickCount := 10
	if nPlot < tickCount {
		tickCount = nPlot
	}
	if tickCount >= 2 {
		labelY := clampInt(pricePlot.Max.Y+timeLabelH-2, pricePanel.Min.Y+12, pricePanel.Max.Y-6)
		for t := 0; t < tickCount; t++ {
			idx := int(math.Round(float64(t) * float64(nPlot-1) / float64(tickCount-1)))
			x := indexToX(idx)
			drawLine(img, x, pricePanel.Min.Y, x, overallBottom, grid)
			closeMs := klinesPlot[idx].Time
			if durMs > 0 {
				closeMs = closeMs + durMs - 1
			}
			label := time.UnixMilli(closeMs).UTC().Format("01-02 15:04")
			drawText(img, clampInt(x-28, left, cfg.Width-right-56), labelY, label, muted)
		}
	}

	// Bottom range slider (visual cue only, like BRALE).
	sliderRect := rect(left+10, pricePlot.Max.Y+timeLabelH+2, cfg.Width-right-10, pricePanel.Max.Y-6)
	drawRangeSlider(img, sliderRect, grid)

	// Candles
	for i, b := range klinesPlot {
		x := indexToX(i)
		yH := priceToY(b.High)
		yL := priceToY(b.Low)
		yO := priceToY(b.Open)
		yC := priceToY(b.Close)

		drawLine(img, x, yH, x, yL, wickColor)

		c := downColor
		if b.Close >= b.Open {
			c = upColor
		}
		yTop := yC
		yBot := yO
		if yTop > yBot {
			yTop, yBot = yBot, yTop
		}
		if yBot == yTop {
			yBot = yTop + 1
		}
		drawRect(img, x-bodyW/2, yTop, x+bodyW/2, yBot, c)
	}

	// EMA overlays (aligned)
	if cfg.Indicators.ShowEMA {
		drawSeriesDots(img, indexToX, priceToY, ema21, ema21C, 2)
		drawSeriesDots(img, indexToX, priceToY, ema55, ema55C, 2)
		drawSeriesDots(img, indexToX, priceToY, ema100, ema100C, 2)
		drawSeriesDots(img, indexToX, priceToY, ema200, ema200C, 2)
	}

	if cfg.Indicators.ShowDivergence && len(divEvents) > 0 {
		drawDivergenceBubbles(img, pricePlot, indexToX, priceToY, klinesAll, divEvents, start, end)
	}

	// Sub-panels
	if volumePanel.Dx() > 0 && volumePanel.Dy() > 0 {
		if hasSqueezePanel {
			volMA := smaSeries(volumes, 20)
			volRatio := make([]float64, len(volumes))
			for i := range volumes {
				if math.IsNaN(volMA[i]) || volMA[i] <= 0 {
					volRatio[i] = math.NaN()
					continue
				}
				volRatio[i] = volumes[i] / volMA[i]
			}
			drawSqueezeVolumePanel(
				img,
				volumePanel,
				indexToX,
				bodyW,
				opens,
				closes,
				volumes,
				volRatio,
				squeezeSeries,
				grid,
				text,
				muted,
			)
		} else {
			drawVolumePanel(img, volumePanel, indexToX, bodyW, opens, closes, volumes, grid, text, muted, timeframe)
		}
	}
	if cvdPanel.Dx() > 0 && cvdPanel.Dy() > 0 && hasCVD {
		drawCVDPanelBrale(img, cvdPanel, indexToX, cvdSeries, grid, text, muted, timeframe)
	}
	if macdPanel.Dx() > 0 && macdPanel.Dy() > 0 && hasMACD {
		drawMACDPanelBrale(img, macdPanel, indexToX, macdLine, macdSignal, macdHist, grid, text, muted, timeframe)
	}
	if wtPanel.Dx() > 0 && wtPanel.Dy() > 0 && hasWT {
		drawWTMFIPanel(img, wtPanel, indexToX, wtSeries, grid, text, muted, timeframe)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func buildLegend(ind IndicatorRenderConfig) string {
	items := make([]string, 0, 6)
	if ind.ShowEMA {
		items = append(items, "EMA21/55/100/200")
	}
	if ind.ShowWaveTrend {
		items = append(items, "WT")
	}
	if ind.ShowMACD {
		items = append(items, "MACD")
	}
	if ind.ShowCVD {
		items = append(items, "CVD")
	}
	if ind.ShowSqueeze {
		items = append(items, "SQZ+VOL")
	}
	if ind.ShowDivergence {
		items = append(items, "DIV")
	}
	if len(items) == 0 {
		return ""
	}
	return "Indicators: " + join(items, "  ")
}

func formatSymbolPair(symbol string) string {
	s := strings.ToUpper(strings.TrimSpace(symbol))
	if s == "" {
		return ""
	}
	if strings.Contains(s, "/") || strings.Contains(s, ":") {
		return s
	}
	for _, q := range []string{"USDT", "USDC", "USD"} {
		if strings.HasSuffix(s, q) && len(s) > len(q) {
			return s[:len(s)-len(q)] + "/" + q
		}
	}
	return s
}

func buildBraleHeaderSummary(wtSeries []float64, macdHist []float64, div *indicator.DivergenceResult, showDiv bool) string {
	wt := math.NaN()
	if len(wtSeries) > 0 {
		wt = wtSeries[len(wtSeries)-1]
	}
	macd := math.NaN()
	if len(macdHist) > 0 {
		macd = macdHist[len(macdHist)-1]
	}
	divText := ""
	if showDiv {
		divText = strings.TrimSpace(formatDivergence(div))
		if divText == "" {
			divText = "DIV: none"
		}
	}

	if (math.IsNaN(wt) || math.IsInf(wt, 0)) && (math.IsNaN(macd) || math.IsInf(macd, 0)) && divText == "" {
		return ""
	}
	parts := make([]string, 0, 2)
	if !math.IsNaN(wt) && !math.IsInf(wt, 0) {
		parts = append(parts, fmt.Sprintf("WTMFI %.1f", wt))
	}
	if !math.IsNaN(macd) && !math.IsInf(macd, 0) {
		if macd >= 0 {
			parts = append(parts, "MACD positive")
		} else {
			parts = append(parts, "MACD negative")
		}
	}
	if divText != "" {
		parts = append(parts, divText)
	}
	out := join(parts, " | ")
	if len(out) > 160 {
		out = out[:157] + "..."
	}
	return out
}

func drawBraleLegendRow(img *image.RGBA, x0, y, x1 int, timeframe string, ind IndicatorRenderConfig, muted color.RGBA) {
	if img == nil || x1 <= x0 {
		return
	}
	type item struct {
		Label string
		Color color.RGBA
	}

	items := make([]item, 0, 8)
	items = append(items, item{Label: "Price_" + timeframe, Color: color.RGBA{R: 246, G: 70, B: 93, A: 255}})
	if ind.ShowEMA {
		items = append(items,
			item{Label: "EMA21", Color: color.RGBA{R: 246, G: 70, B: 93, A: 255}},    // red
			item{Label: "EMA55", Color: color.RGBA{R: 240, G: 185, B: 11, A: 255}},   // yellow
			item{Label: "EMA100", Color: color.RGBA{R: 14, G: 203, B: 129, A: 255}},  // green
			item{Label: "EMA200", Color: color.RGBA{R: 234, G: 236, B: 239, A: 255}}, // white-ish
		)
	}
	if ind.ShowDivergence {
		items = append(items, item{Label: "Div.", Color: color.RGBA{R: 125, G: 211, B: 252, A: 255}})
	}
	if ind.ShowCVD {
		items = append(items, item{Label: "CVD", Color: color.RGBA{R: 34, G: 211, B: 238, A: 255}})
	}

	x := x0
	for _, it := range items {
		if x >= x1-20 {
			break
		}
		// Color swatch
		drawRect(img, x, y-9, x+10, y+1, it.Color)
		drawText(img, x+14, y, it.Label, muted)
		// Approx advance: 7px per char + padding
		x += 14 + 7*len(it.Label) + 18
	}
}

func formatSqueeze(s *indicator.VolatilityWarningResult) string {
	if s == nil {
		return ""
	}
	if !s.IsSqueeze {
		return "SQZ: off"
	}
	level := "on"
	if s.WarningL2 {
		level = "L2"
	} else if s.WarningL1 {
		level = "L1"
	}
	return fmt.Sprintf("SQZ: %s  x%d", level, s.SqueezeCount)
}

func squeezeColor(s *indicator.VolatilityWarningResult) color.RGBA {
	if s == nil {
		return color.RGBA{R: 132, G: 142, B: 156, A: 255}
	}
	if !s.IsSqueeze {
		return color.RGBA{R: 132, G: 142, B: 156, A: 255}
	}
	if s.WarningL2 {
		return color.RGBA{R: 246, G: 70, B: 93, A: 255}
	}
	if s.WarningL1 {
		return color.RGBA{R: 240, G: 185, B: 11, A: 255}
	}
	return color.RGBA{R: 59, G: 130, B: 246, A: 255}
}

func formatDivergence(d *indicator.DivergenceResult) string {
	if d == nil || d.Total <= 0 {
		return ""
	}
	ev := append([]indicator.DivergenceEvent(nil), d.Events...)
	sort.Slice(ev, func(i, j int) bool { return ev[i].BarsAgo < ev[j].BarsAgo })
	if len(ev) > 3 {
		ev = ev[:3]
	}
	parts := make([]string, 0, 4)
	for _, e := range ev {
		parts = append(parts, fmt.Sprintf("%s:%s@%d", e.Indicator, abbrevDivType(e.Type), e.BarsAgo))
	}
	return "DIV: " + join(parts, "  ")
}

func abbrevDivType(t indicator.DivergenceType) string {
	switch t {
	case indicator.DivergencePositiveRegular:
		return "+R"
	case indicator.DivergenceNegativeRegular:
		return "-R"
	case indicator.DivergencePositiveHidden:
		return "+H"
	case indicator.DivergenceNegativeHidden:
		return "-H"
	default:
		return string(t)
	}
}

func drawWaveTrendPanel(img *image.RGBA, r image.Rectangle, indexToX func(int) int, series []float64, grid, text, muted color.RGBA) {
	// Grid
	drawLine(img, r.Min.X, r.Min.Y, r.Max.X, r.Min.Y, grid)
	drawLine(img, r.Min.X, r.Max.Y, r.Max.X, r.Max.Y, grid)

	// Fixed bounds (indicator is clamped to [-60,60]).
	minV := -60.0
	maxV := 60.0
	toY := func(v float64) int {
		y := float64(r.Min.Y) + (maxV-v)/(maxV-minV)*float64(r.Dy())
		return clampInt(int(math.Round(y)), r.Min.Y, r.Max.Y)
	}

	// Reference lines
	drawLine(img, r.Min.X, toY(0), r.Max.X, toY(0), grid)
	drawLine(img, r.Min.X, toY(50), r.Max.X, toY(50), color.RGBA{R: 43, G: 49, B: 57, A: 255})
	drawLine(img, r.Min.X, toY(-50), r.Max.X, toY(-50), color.RGBA{R: 43, G: 49, B: 57, A: 255})

	drawText(img, r.Min.X+6, r.Min.Y+14, "WT", text)

	if len(series) < 2 {
		return
	}
	drawSeriesLine(img, indexToX, toY, series, color.RGBA{R: 59, G: 130, B: 246, A: 255})
}

func drawWTMFIPanel(img *image.RGBA, r image.Rectangle, indexToX func(int) int, series []float64, grid, text, muted color.RGBA, timeframe string) {
	drawLine(img, r.Min.X, r.Min.Y, r.Max.X, r.Min.Y, grid)
	drawLine(img, r.Min.X, r.Max.Y, r.Max.X, r.Max.Y, grid)
	drawText(img, r.Min.X+6, r.Min.Y+14, "WT+MFI "+timeframe, text)

	// Reserve header space so y-axis labels and threshold lines don't overlap title/legend.
	plotTop := r.Min.Y + 18
	plotBot := r.Max.Y - 2
	if plotBot <= plotTop+10 {
		return
	}

	// Fixed bounds and thresholds (match BRALE config: 53/-53).
	minV := -60.0
	maxV := 60.0
	toY := func(v float64) int {
		y := float64(plotTop) + (maxV-v)/(maxV-minV)*float64(plotBot-plotTop)
		return clampInt(int(math.Round(y)), plotTop, plotBot)
	}

	zero := color.RGBA{R: 148, G: 163, B: 184, A: 140}
	ref := color.RGBA{R: 148, G: 163, B: 184, A: 80}
	over := color.RGBA{R: 125, G: 211, B: 252, A: 160}
	under := color.RGBA{R: 244, G: 114, B: 182, A: 160}

	drawLine(img, r.Min.X, toY(0), r.Max.X, toY(0), zero)
	drawLine(img, r.Min.X, toY(53), r.Max.X, toY(53), ref)
	drawLine(img, r.Min.X, toY(-53), r.Max.X, toY(-53), ref)

	// Y labels
	for _, v := range []float64{60, 40, 20, 0, -20, -40, -60} {
		drawText(img, 6, toY(v)+4, fmt.Sprintf("%.0f", v), muted)
	}

	// Legend (compact)
	legendY := r.Min.Y + 14
	legendX := r.Min.X + 220
	drawRect(img, legendX, legendY-9, legendX+10, legendY+1, color.RGBA{R: 250, G: 204, B: 21, A: 255})
	drawText(img, legendX+14, legendY, "WT+MFI", muted)
	legendX += 14 + 7*len("WT+MFI") + 18
	drawRect(img, legendX, legendY-9, legendX+10, legendY+1, over)
	drawText(img, legendX+14, legendY, "Overbought", muted)
	legendX += 14 + 7*len("Overbought") + 18
	drawRect(img, legendX, legendY-9, legendX+10, legendY+1, under)
	drawText(img, legendX+14, legendY, "Oversold", muted)
	legendX += 14 + 7*len("Oversold") + 18
	drawRect(img, legendX, legendY-9, legendX+10, legendY+1, zero)
	drawText(img, legendX+14, legendY, "Zero", muted)

	if len(series) < 2 {
		return
	}
	wtColor := color.RGBA{R: 250, G: 204, B: 21, A: 255} // yellow
	drawSeriesDots(img, indexToX, toY, series, wtColor, 2)
}

func drawVolumePanel(img *image.RGBA, r image.Rectangle, indexToX func(int) int, bodyW int, opens, closes, volumes []float64, grid, text, muted color.RGBA, timeframe string) {
	drawLine(img, r.Min.X, r.Min.Y, r.Max.X, r.Min.Y, grid)
	drawLine(img, r.Min.X, r.Max.Y, r.Max.X, r.Max.Y, grid)
	drawText(img, r.Min.X+6, r.Min.Y+14, "Volume "+timeframe, text)

	plotTop := r.Min.Y + 18
	plotBot := r.Max.Y - 2
	if plotBot <= plotTop {
		return
	}

	maxVol := 0.0
	for _, v := range volumes {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		if v > maxVol {
			maxVol = v
		}
	}
	if maxVol <= 0 {
		return
	}

	// Grid + Y labels
	for j := 0; j <= 4; j++ {
		y := plotTop + int(float64(j)/4.0*float64(plotBot-plotTop))
		drawLine(img, r.Min.X, y, r.Max.X, y, grid)
		v := maxVol * (1.0 - float64(j)/4.0)
		drawText(img, 6, y+4, formatAxisVolume(v), muted)
	}

	upVol := color.RGBA{R: 14, G: 203, B: 129, A: 255}
	downVol := color.RGBA{R: 246, G: 70, B: 93, A: 255}

	for i := 0; i < len(volumes); i++ {
		v := volumes[i]
		if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
			continue
		}
		x := indexToX(i)
		x0 := x - bodyW/2
		x1 := x + bodyW/2
		h := int(math.Round(v / maxVol * float64(plotBot-plotTop)))
		if h < 1 {
			h = 1
		}
		y0 := plotBot - h
		y1 := plotBot
		c := downVol
		if i < len(opens) && i < len(closes) && closes[i] >= opens[i] {
			c = upVol
		}
		drawRect(img, x0, y0, x1, y1, c)
	}
}

func drawCVDPanelBrale(img *image.RGBA, r image.Rectangle, indexToX func(int) int, cvd []float64, grid, text, muted color.RGBA, timeframe string) {
	drawLine(img, r.Min.X, r.Min.Y, r.Max.X, r.Min.Y, grid)
	drawLine(img, r.Min.X, r.Max.Y, r.Max.X, r.Max.Y, grid)
	drawText(img, r.Min.X+6, r.Min.Y+14, "CVD "+timeframe, text)

	cvdC := color.RGBA{R: 34, G: 211, B: 238, A: 255}

	// Legend centered-ish
	legendY := r.Min.Y + 14
	legendX := r.Min.X + 520
	drawRect(img, legendX, legendY-9, legendX+10, legendY+1, cvdC)
	drawText(img, legendX+14, legendY, "CVD (quote)", muted)

	minV := 0.0
	maxV := 0.0
	ok := false
	for _, v := range cvd {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		if !ok {
			minV, maxV = v, v
			ok = true
			continue
		}
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}
	if !ok {
		return
	}
	if minV == maxV {
		maxV = minV + 1
	}
	minV = math.Min(minV, 0)
	maxV = math.Max(maxV, 0)
	pad := (maxV - minV) * 0.12
	if pad <= 0 {
		pad = 1
	}
	minV -= pad
	maxV += pad

	toY := func(v float64) int {
		y := float64(r.Min.Y) + (maxV-v)/(maxV-minV)*float64(r.Dy())
		return clampInt(int(math.Round(y)), r.Min.Y, r.Max.Y)
	}

	// Y grid + labels
	for j := 0; j <= 4; j++ {
		y := r.Min.Y + int(float64(j)/4.0*float64(r.Dy()))
		drawLine(img, r.Min.X, y, r.Max.X, y, grid)
		v := maxV - float64(j)/4.0*(maxV-minV)
		drawText(img, 6, y+4, formatAxisSignedVolume(v), muted)
	}

	// Zero line
	drawLine(img, r.Min.X, toY(0), r.Max.X, toY(0), color.RGBA{R: 148, G: 163, B: 184, A: 120})

	if len(cvd) < 2 {
		return
	}
	drawSeriesDots(img, indexToX, toY, cvd, cvdC, 2)
}

type squeezeSeriesData struct {
	IsSqueeze []bool
	WarningL1 []bool
	WarningL2 []bool
	Count     []int
	Tightness []float64
}

func (s squeezeSeriesData) slice(start, end int) squeezeSeriesData {
	if start < 0 {
		start = 0
	}
	if end < start {
		end = start
	}
	clampBool := func(in []bool) []bool {
		if len(in) == 0 || start >= len(in) {
			return nil
		}
		e := end
		if e > len(in) {
			e = len(in)
		}
		out := make([]bool, e-start)
		copy(out, in[start:e])
		return out
	}
	clampInt := func(in []int) []int {
		if len(in) == 0 || start >= len(in) {
			return nil
		}
		e := end
		if e > len(in) {
			e = len(in)
		}
		out := make([]int, e-start)
		copy(out, in[start:e])
		return out
	}
	clampF := func(in []float64) []float64 {
		if len(in) == 0 || start >= len(in) {
			return nil
		}
		e := end
		if e > len(in) {
			e = len(in)
		}
		out := make([]float64, e-start)
		copy(out, in[start:e])
		return out
	}
	return squeezeSeriesData{
		IsSqueeze: clampBool(s.IsSqueeze),
		WarningL1: clampBool(s.WarningL1),
		WarningL2: clampBool(s.WarningL2),
		Count:     clampInt(s.Count),
		Tightness: clampF(s.Tightness),
	}
}

func computeSqueezeSeries(closes, highs, lows []float64) squeezeSeriesData {
	n := len(closes)
	out := squeezeSeriesData{
		IsSqueeze: make([]bool, n),
		WarningL1: make([]bool, n),
		WarningL2: make([]bool, n),
		Count:     make([]int, n),
		Tightness: make([]float64, n),
	}

	const squeezeLen = 20
	const bbMult = 2.0
	const kcMult = 1.5
	const atrLen = 20
	const minDur = 5
	const l2Ratio = 0.80

	bbMid := smaSeries(closes, squeezeLen)
	bbStd := stdDevSeries(closes, squeezeLen)
	kcMid := emaSeries(closes, squeezeLen)
	atr := atrSeries(highs, lows, closes, atrLen)

	for i := 0; i < n; i++ {
		if math.IsNaN(bbMid[i]) || math.IsNaN(bbStd[i]) || math.IsNaN(kcMid[i]) || math.IsNaN(atr[i]) {
			out.Tightness[i] = math.NaN()
			continue
		}

		bbTop := bbMid[i] + bbMult*bbStd[i]
		bbBot := bbMid[i] - bbMult*bbStd[i]
		kcTop := kcMid[i] + kcMult*atr[i]
		kcBot := kcMid[i] - kcMult*atr[i]

		isSq := bbTop < kcTop && bbBot > kcBot
		out.IsSqueeze[i] = isSq
		if isSq {
			prev := 0
			if i > 0 {
				prev = out.Count[i-1]
			}
			out.Count[i] = prev + 1
		} else {
			out.Count[i] = 0
		}

		kcWidth := kcTop - kcBot
		bbWidth := bbTop - bbBot
		ratio := 1.0
		if kcWidth > 0 {
			ratio = bbWidth / kcWidth
		}
		out.Tightness[i] = ratio

		l1 := isSq && out.Count[i] >= minDur
		out.WarningL1[i] = l1
		out.WarningL2[i] = l1 && ratio <= l2Ratio
	}

	return out
}

func drawSqueezeVolumePanel(
	img *image.RGBA,
	r image.Rectangle,
	indexToX func(int) int,
	bodyW int,
	opens, closes, volumes, volRatio []float64,
	sq squeezeSeriesData,
	grid, text, muted color.RGBA,
) {
	drawLine(img, r.Min.X, r.Min.Y, r.Max.X, r.Min.Y, grid)
	drawLine(img, r.Min.X, r.Max.Y, r.Max.X, r.Max.Y, grid)
	drawText(img, r.Min.X+6, r.Min.Y+14, "SQZ+VOL", text)

	plotTop := r.Min.Y + 18
	plotBot := r.Max.Y - 2
	if plotBot <= plotTop {
		return
	}

	maxVol := 0.0
	for _, v := range volumes {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		if v > maxVol {
			maxVol = v
		}
	}
	if maxVol <= 0 {
		return
	}

	// Header stats (last bar)
	last := len(volumes) - 1
	if last >= 0 && last < len(sq.IsSqueeze) {
		sqText := "off"
		if sq.IsSqueeze[last] {
			if sq.WarningL2[last] {
				sqText = "L2"
			} else if sq.WarningL1[last] {
				sqText = "L1"
			} else {
				sqText = "on"
			}
		}
		vr := volRatio[last]
		if !math.IsNaN(vr) && !math.IsInf(vr, 0) {
			drawText(img, r.Min.X+70, r.Min.Y+14, fmt.Sprintf("SQZ:%s x%d  VOLx%.1f", sqText, sq.Count[last], vr), muted)
		} else {
			drawText(img, r.Min.X+70, r.Min.Y+14, fmt.Sprintf("SQZ:%s x%d", sqText, sq.Count[last]), muted)
		}
	}

	squeezeBg := color.RGBA{R: 59, G: 130, B: 246, A: 16}
	squeezeBgL1 := color.RGBA{R: 240, G: 185, B: 11, A: 16}
	squeezeBgL2 := color.RGBA{R: 246, G: 70, B: 93, A: 16}

	upVol := color.RGBA{R: 14, G: 203, B: 129, A: 120}
	downVol := color.RGBA{R: 246, G: 70, B: 93, A: 120}
	yellow := color.RGBA{R: 240, G: 185, B: 11, A: 230}
	hotRed := color.RGBA{R: 246, G: 70, B: 93, A: 230}

	for i := 0; i < len(volumes); i++ {
		x := indexToX(i)
		x0 := x - bodyW/2
		x1 := x + bodyW/2

		if i < len(sq.IsSqueeze) && sq.IsSqueeze[i] {
			bg := squeezeBg
			if sq.WarningL2[i] {
				bg = squeezeBgL2
			} else if sq.WarningL1[i] {
				bg = squeezeBgL1
			}
			drawRect(img, x0, plotTop, x1, plotBot, bg)
			// Tiny dot near top for squeeze visibility.
			dc := color.RGBA{R: 59, G: 130, B: 246, A: 220}
			if sq.WarningL2[i] {
				dc = hotRed
			} else if sq.WarningL1[i] {
				dc = yellow
			}
			drawRect(img, x-1, plotTop+1, x+1, plotTop+3, dc)
		}

		v := volumes[i]
		if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
			continue
		}
		h := int(math.Round(v / maxVol * float64(plotBot-plotTop)))
		if h < 1 {
			h = 1
		}
		y0 := plotBot - h
		y1 := plotBot

		c := downVol
		if i < len(opens) && i < len(closes) && closes[i] >= opens[i] {
			c = upVol
		}
		if i < len(volRatio) {
			vr := volRatio[i]
			if !math.IsNaN(vr) && !math.IsInf(vr, 0) {
				if vr >= 3.5 {
					c = hotRed
				} else if vr >= 2.5 {
					c = yellow
				}
			}
		}
		drawRect(img, x0, y0, x1, y1, c)
	}
}

func drawDivergenceMarkers(
	img *image.RGBA,
	r image.Rectangle,
	indexToX func(int) int,
	priceToY func(float64) int,
	klines []market.KlineBar,
	events []indicator.DivergenceEvent,
) {
	if len(klines) < 2 || len(events) == 0 {
		return
	}

	type key struct {
		Idx    int
		Abbrev string
	}
	group := make(map[key][]string)
	lastIdx := len(klines) - 1
	for _, e := range events {
		if e.BarsAgo <= 0 {
			continue
		}
		idx := lastIdx - e.BarsAgo
		if idx < 0 || idx >= len(klines) {
			continue
		}
		k := key{Idx: idx, Abbrev: abbrevDivType(e.Type)}
		group[k] = append(group[k], e.Indicator)
	}
	if len(group) == 0 {
		return
	}

	keys := make([]key, 0, len(group))
	for k := range group {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Idx == keys[j].Idx {
			return keys[i].Abbrev < keys[j].Abbrev
		}
		return keys[i].Idx < keys[j].Idx
	})

	colorByAbbrev := func(ab string) color.RGBA {
		switch ab {
		case "+R":
			return color.RGBA{R: 34, G: 197, B: 94, A: 255}
		case "-R":
			return color.RGBA{R: 246, G: 70, B: 93, A: 255}
		case "+H":
			return color.RGBA{R: 59, G: 130, B: 246, A: 255}
		case "-H":
			return color.RGBA{R: 245, G: 158, B: 11, A: 255}
		default:
			return color.RGBA{R: 132, G: 142, B: 156, A: 255}
		}
	}

	for _, k := range keys {
		names := uniqueStrings(group[k])
		sort.Strings(names)
		if len(names) > 3 {
			names = append(names[:3], fmt.Sprintf("+%d", len(group[k])-3))
		}

		c := colorByAbbrev(k.Abbrev)
		x := indexToX(k.Idx)
		label := fmt.Sprintf("DIV%s %s", k.Abbrev, join(names, "/"))

		isBull := len(k.Abbrev) > 0 && k.Abbrev[0] == '+'
		if isBull {
			y := priceToY(klines[k.Idx].Low) + 14
			y = clampInt(y, r.Min.Y+16, r.Max.Y-6)
			drawLine(img, x, y-10, x, y-2, c)
			drawLine(img, x, y-10, x-3, y-6, c)
			drawLine(img, x, y-10, x+3, y-6, c)
			tx := clampInt(x+6, r.Min.X+2, r.Max.X-220)
			drawText(img, tx, y, label, c)
		} else {
			y := priceToY(klines[k.Idx].High) - 6
			y = clampInt(y, r.Min.Y+16, r.Max.Y-6)
			drawLine(img, x, y+2, x, y+10, c)
			drawLine(img, x, y+10, x-3, y+6, c)
			drawLine(img, x, y+10, x+3, y+6, c)
			tx := clampInt(x+6, r.Min.X+2, r.Max.X-220)
			drawText(img, tx, y, label, c)
		}
	}
}

func uniqueStrings(items []string) []string {
	seen := make(map[string]bool, len(items))
	out := make([]string, 0, len(items))
	for _, s := range items {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func drawMACDPanelBrale(img *image.RGBA, r image.Rectangle, indexToX func(int) int, macdLine, signal, hist []float64, grid, text, muted color.RGBA, timeframe string) {
	drawLine(img, r.Min.X, r.Min.Y, r.Max.X, r.Min.Y, grid)
	drawLine(img, r.Min.X, r.Max.Y, r.Max.X, r.Max.Y, grid)
	drawText(img, r.Min.X+6, r.Min.Y+14, "MACD "+timeframe, text)

	// Legend centered-ish
	legendY := r.Min.Y + 14
	legendX := r.Min.X + 520
	histPosC := color.RGBA{R: 14, G: 203, B: 129, A: 200}
	histNegC := color.RGBA{R: 246, G: 70, B: 93, A: 200}
	difC := color.RGBA{R: 125, G: 211, B: 252, A: 255}
	deaC := color.RGBA{R: 244, G: 114, B: 182, A: 255}
	// Split swatch (green/red) for histogram
	drawRect(img, legendX, legendY-9, legendX+5, legendY+1, histPosC)
	drawRect(img, legendX+5, legendY-9, legendX+10, legendY+1, histNegC)
	drawText(img, legendX+14, legendY, "MACD Hist", muted)
	legendX += 14 + 7*len("MACD Hist") + 18
	drawRect(img, legendX, legendY-9, legendX+10, legendY+1, difC)
	drawText(img, legendX+14, legendY, "DIF", muted)
	legendX += 14 + 7*len("DIF") + 18
	drawRect(img, legendX, legendY-9, legendX+10, legendY+1, deaC)
	drawText(img, legendX+14, legendY, "DEA", muted)

	minV := 0.0
	maxV := 0.0
	ok := false
	for _, series := range [][]float64{hist, macdLine, signal} {
		for i := 0; i < len(series); i++ {
			v := series[i]
			if math.IsNaN(v) || math.IsInf(v, 0) {
				continue
			}
			if !ok {
				minV, maxV = v, v
				ok = true
				continue
			}
			if v < minV {
				minV = v
			}
			if v > maxV {
				maxV = v
			}
		}
	}
	if !ok {
		minV, maxV = -1, 1
	}
	if minV == maxV {
		maxV = minV + 1
	}
	minV = math.Min(minV, 0)
	maxV = math.Max(maxV, 0)
	pad := (maxV - minV) * 0.18
	if pad <= 0 {
		pad = 1
	}
	minV -= pad
	maxV += pad

	toY := func(v float64) int {
		y := float64(r.Min.Y) + (maxV-v)/(maxV-minV)*float64(r.Dy())
		return clampInt(int(math.Round(y)), r.Min.Y, r.Max.Y)
	}

	// Y grid + labels
	for j := 0; j <= 4; j++ {
		y := r.Min.Y + int(float64(j)/4.0*float64(r.Dy()))
		drawLine(img, r.Min.X, y, r.Max.X, y, grid)
		v := maxV - float64(j)/4.0*(maxV-minV)
		drawText(img, 6, y+4, formatAxisOsc(v), muted)
	}

	// Zero line
	drawLine(img, r.Min.X, toY(0), r.Max.X, toY(0), color.RGBA{R: 148, G: 163, B: 184, A: 120})

	// Histogram
	if len(hist) >= 2 {
		for i := 0; i < len(hist); i++ {
			v := hist[i]
			if math.IsNaN(v) || math.IsInf(v, 0) {
				continue
			}
			x := indexToX(i)
			y0 := toY(0)
			y1 := toY(v)
			c := histNegC
			if v >= 0 {
				c = histPosC
			}
			if y1 < y0 {
				y0, y1 = y1, y0
			}
			// Thicker bars for better visibility at 1600px width.
			drawRect(img, x-1, y0, x+2, y1+1, c)
		}
	}

	drawSeriesDots(img, indexToX, toY, macdLine, difC, 2)
	drawSeriesDots(img, indexToX, toY, signal, deaC, 2)
}

func drawMACDPanel(img *image.RGBA, r image.Rectangle, indexToX func(int) int, macdLine, signal, hist []float64, grid, text, muted color.RGBA) {
	drawLine(img, r.Min.X, r.Min.Y, r.Max.X, r.Min.Y, grid)
	drawLine(img, r.Min.X, r.Max.Y, r.Max.X, r.Max.Y, grid)
	drawText(img, r.Min.X+6, r.Min.Y+14, "MACD", text)

	minV := 0.0
	maxV := 0.0
	ok := false
	for i := 0; i < len(hist); i++ {
		v := hist[i]
		if math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		if !ok {
			minV, maxV = v, v
			ok = true
			continue
		}
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}
	for _, series := range [][]float64{macdLine, signal} {
		for i := 0; i < len(series); i++ {
			v := series[i]
			if math.IsNaN(v) || math.IsInf(v, 0) {
				continue
			}
			if !ok {
				minV, maxV = v, v
				ok = true
				continue
			}
			if v < minV {
				minV = v
			}
			if v > maxV {
				maxV = v
			}
		}
	}
	if !ok {
		minV, maxV = -1, 1
	}
	if minV == maxV {
		maxV = minV + 1
	}
	minV = math.Min(minV, 0)
	maxV = math.Max(maxV, 0)
	pad := (maxV - minV) * 0.15
	if pad <= 0 {
		pad = 1
	}
	minV -= pad
	maxV += pad

	toY := func(v float64) int {
		y := float64(r.Min.Y) + (maxV-v)/(maxV-minV)*float64(r.Dy())
		return clampInt(int(math.Round(y)), r.Min.Y, r.Max.Y)
	}

	// Zero line
	drawLine(img, r.Min.X, toY(0), r.Max.X, toY(0), grid)

	// Histogram
	if len(hist) >= 2 {
		for i := 0; i < len(hist); i++ {
			v := hist[i]
			if math.IsNaN(v) || math.IsInf(v, 0) {
				continue
			}
			x := indexToX(i)
			y0 := toY(0)
			y1 := toY(v)
			c := color.RGBA{R: 107, G: 114, B: 128, A: 255}
			if v >= 0 {
				c = color.RGBA{R: 14, G: 203, B: 129, A: 180}
			} else {
				c = color.RGBA{R: 246, G: 70, B: 93, A: 180}
			}
			drawLine(img, x, y0, x, y1, c)
		}
	}

	drawSeriesLine(img, indexToX, toY, macdLine, color.RGBA{R: 59, G: 130, B: 246, A: 255})
	drawSeriesLine(img, indexToX, toY, signal, color.RGBA{R: 240, G: 185, B: 11, A: 255})
}

func smaSeries(values []float64, period int) []float64 {
	out := make([]float64, len(values))
	if period <= 0 {
		fillNaN(out)
		return out
	}
	sum := 0.0
	for i, v := range values {
		sum += v
		if i >= period {
			sum -= values[i-period]
		}
		if i >= period-1 {
			out[i] = sum / float64(period)
		} else {
			out[i] = math.NaN()
		}
	}
	return out
}

func stdDevSeries(values []float64, period int) []float64 {
	out := make([]float64, len(values))
	if period <= 0 {
		fillNaN(out)
		return out
	}
	sum := 0.0
	sumSq := 0.0
	for i, v := range values {
		sum += v
		sumSq += v * v
		if i >= period {
			removed := values[i-period]
			sum -= removed
			sumSq -= removed * removed
		}
		if i >= period-1 {
			mean := sum / float64(period)
			variance := (sumSq / float64(period)) - (mean * mean)
			if variance < 0 {
				variance = 0
			}
			out[i] = math.Sqrt(variance)
		} else {
			out[i] = math.NaN()
		}
	}
	return out
}

func atrSeries(highs, lows, closes []float64, period int) []float64 {
	n := len(closes)
	out := make([]float64, n)
	if period <= 0 || n == 0 || n < period {
		fillNaN(out)
		return out
	}
	tr := make([]float64, n)
	for i := 0; i < n; i++ {
		if i == 0 {
			tr[i] = highs[i] - lows[i]
			continue
		}
		hl := highs[i] - lows[i]
		hc := math.Abs(highs[i] - closes[i-1])
		lc := math.Abs(lows[i] - closes[i-1])
		tr[i] = math.Max(hl, math.Max(hc, lc))
	}
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += tr[i]
		out[i] = math.NaN()
	}
	atr := sum / float64(period)
	out[period-1] = atr
	for i := period; i < n; i++ {
		atr = (atr*float64(period-1) + tr[i]) / float64(period)
		out[i] = atr
	}
	return out
}

func drawTimeLabels(img *image.RGBA, klines []market.KlineBar, x0, x1, y int, c color.RGBA) {
	if len(klines) < 2 {
		return
	}
	points := []int{0, len(klines) / 2, len(klines) - 1}
	seen := map[int]bool{}
	for _, idx := range points {
		if idx < 0 || idx >= len(klines) || seen[idx] {
			continue
		}
		seen[idx] = true
		tm := time.UnixMilli(klines[idx].Time).UTC()
		label := tm.Format("01-02 15:04")
		x := x0 + int(float64(x1-x0)*float64(idx)/float64(len(klines)-1))
		drawText(img, clampInt(x-28, x0, x1-56), y, label, c)
	}
}

func toMarketKlines(bars []market.KlineBar) []market.Kline {
	out := make([]market.Kline, 0, len(bars))
	for _, b := range bars {
		out = append(out, market.Kline{
			OpenTime:            b.Time,
			Open:                b.Open,
			High:                b.High,
			Low:                 b.Low,
			Close:               b.Close,
			Volume:              b.Volume,
			CloseTime:           b.Time,
			QuoteVolume:         b.QuoteVolume,
			TakerBuyQuoteVolume: b.TakerBuyQuoteVolume,
		})
	}
	return out
}

func hasAnyCVDData(bars []market.KlineBar) bool {
	for _, b := range bars {
		if b.QuoteVolume > 0 && b.TakerBuyQuoteVolume > 0 && b.TakerBuyQuoteVolume <= b.QuoteVolume {
			return true
		}
	}
	return false
}

func cvdSeriesFromBars(bars []market.KlineBar) []float64 {
	out := make([]float64, len(bars))
	var sum float64
	for i, b := range bars {
		qv := b.QuoteVolume
		tbq := b.TakerBuyQuoteVolume
		if qv <= 0 || tbq <= 0 || tbq > qv || math.IsNaN(qv) || math.IsNaN(tbq) || math.IsInf(qv, 0) || math.IsInf(tbq, 0) {
			out[i] = sum
			continue
		}
		sum += (2 * tbq) - qv
		out[i] = sum
	}
	return out
}

func priceBounds(klines []market.KlineBar) (minP, maxP float64) {
	minP = klines[0].Low
	maxP = klines[0].High
	for _, b := range klines {
		if b.Low < minP {
			minP = b.Low
		}
		if b.High > maxP {
			maxP = b.High
		}
	}
	return minP, maxP
}

func formatAxisPrice(p float64) string {
	abs := math.Abs(p)
	switch {
	case abs >= 10000:
		return fmt.Sprintf("%.0f", p)
	case abs >= 100:
		return fmt.Sprintf("%.2f", p)
	case abs >= 1:
		return fmt.Sprintf("%.4f", p)
	default:
		return fmt.Sprintf("%.6f", p)
	}
}

func formatAxisVolume(v float64) string {
	if v < 0 {
		v = -v
	}
	return formatWithCommas(int64(math.Round(v)))
}

func formatAxisSignedVolume(v float64) string {
	if v < 0 {
		return "-" + formatWithCommas(int64(math.Round(-v)))
	}
	return formatWithCommas(int64(math.Round(v)))
}

func formatAxisOsc(v float64) string {
	abs := math.Abs(v)
	switch {
	case abs >= 10:
		return fmt.Sprintf("%.2f", v)
	case abs >= 1:
		return fmt.Sprintf("%.3f", v)
	default:
		return fmt.Sprintf("%.4f", v)
	}
}

func formatWithCommas(n int64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	if s[0] == '-' {
		b.WriteByte('-')
		s = s[1:]
	}
	pre := len(s) % 3
	if pre == 0 {
		pre = 3
	}
	b.WriteString(s[:pre])
	for i := pre; i < len(s); i += 3 {
		b.WriteByte(',')
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

func formatPrice(p float64) string {
	abs := math.Abs(p)
	switch {
	case abs >= 10000:
		return fmt.Sprintf("%.2f", p)
	case abs >= 100:
		return fmt.Sprintf("%.2f", p)
	case abs >= 1:
		return fmt.Sprintf("%.4f", p)
	default:
		return fmt.Sprintf("%.6f", p)
	}
}

func emaSeries(values []float64, period int) []float64 {
	out := make([]float64, len(values))
	if period <= 0 || len(values) == 0 || len(values) < period {
		fillNaN(out)
		return out
	}

	sum := 0.0
	for i := 0; i < period; i++ {
		sum += values[i]
		out[i] = math.NaN()
	}
	ema := sum / float64(period)
	out[period-1] = ema
	m := 2.0 / float64(period+1)
	for i := period; i < len(values); i++ {
		ema = (values[i]-ema)*m + ema
		out[i] = ema
	}
	return out
}

func macdSeries(values []float64, fast, slow, signal int) (macdLine, macdSignal, macdHist []float64) {
	n := len(values)
	macdLine = make([]float64, n)
	macdSignal = make([]float64, n)
	macdHist = make([]float64, n)

	if n == 0 || fast <= 0 || slow <= 0 || signal <= 0 || n < slow {
		fillNaN(macdLine)
		fillNaN(macdSignal)
		fillNaN(macdHist)
		return macdLine, macdSignal, macdHist
	}

	fastE := emaSeries(values, fast)
	slowE := emaSeries(values, slow)
	for i := 0; i < n; i++ {
		if math.IsNaN(fastE[i]) || math.IsNaN(slowE[i]) {
			macdLine[i] = math.NaN()
			continue
		}
		macdLine[i] = fastE[i] - slowE[i]
	}

	macdSignal = emaSeries(macdLineWithNaNToZero(macdLine), signal)
	for i := 0; i < n; i++ {
		if math.IsNaN(macdLine[i]) || math.IsNaN(macdSignal[i]) {
			macdHist[i] = math.NaN()
			continue
		}
		macdHist[i] = macdLine[i] - macdSignal[i]
	}
	return macdLine, macdSignal, macdHist
}

func macdLineWithNaNToZero(values []float64) []float64 {
	out := make([]float64, len(values))
	for i, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			out[i] = 0
			continue
		}
		out[i] = v
	}
	return out
}

func fillNaN(values []float64) {
	for i := range values {
		values[i] = math.NaN()
	}
}

func drawSeriesDots(img *image.RGBA, indexToX func(int) int, toY func(float64) int, series []float64, c color.RGBA, radius int) {
	if img == nil || len(series) == 0 || radius <= 0 {
		return
	}
	for i := 0; i < len(series); i++ {
		v := series[i]
		if math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		x := indexToX(i)
		y := toY(v)
		drawFilledCircle(img, x, y, radius, c)
	}
}

func drawFilledCircle(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	if img == nil || r <= 0 {
		return
	}
	r2 := r * r
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if dx*dx+dy*dy > r2 {
				continue
			}
			x := cx + dx
			y := cy + dy
			if x < 0 || y < 0 || x >= img.Bounds().Dx() || y >= img.Bounds().Dy() {
				continue
			}
			img.SetRGBA(x, y, c)
		}
	}
}

func drawRangeSlider(img *image.RGBA, r image.Rectangle, grid color.RGBA) {
	if img == nil || r.Dx() <= 10 || r.Dy() <= 6 {
		return
	}
	track := color.RGBA{R: 30, G: 41, B: 59, A: 255} // same as grid-ish
	fill := color.RGBA{R: 71, G: 85, B: 105, A: 255} // slate
	handle := color.RGBA{R: 148, G: 163, B: 184, A: 255}

	// Track
	drawRect(img, r.Min.X, r.Min.Y+int(math.Round(float64(r.Dy())*0.35)), r.Max.X, r.Min.Y+int(math.Round(float64(r.Dy())*0.65)), track)

	// Selection window (static 80% centered)
	w := r.Dx()
	selW := int(math.Round(float64(w) * 0.82))
	if selW < 20 {
		selW = 20
	}
	selX0 := r.Min.X + (w-selW)/2
	selX1 := selX0 + selW
	y0 := r.Min.Y + int(math.Round(float64(r.Dy())*0.33))
	y1 := r.Min.Y + int(math.Round(float64(r.Dy())*0.67))
	drawRect(img, selX0, y0, selX1, y1, fill)

	// Handles
	drawRect(img, selX0-1, y0-1, selX0+2, y1+1, handle)
	drawRect(img, selX1-2, y0-1, selX1+1, y1+1, handle)

	// Top/bottom border
	drawLine(img, r.Min.X, r.Min.Y, r.Max.X, r.Min.Y, grid)
	drawLine(img, r.Min.X, r.Max.Y, r.Max.X, r.Max.Y, grid)
}

func divergenceIndicatorLabel(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "hist":
		return "Hist"
	case "rsi":
		return "RSI"
	case "stoch":
		return "Stoch"
	case "obv":
		return "OBV"
	case "cmf":
		return "CMF"
	case "mom":
		return "MOM"
	case "mfi":
		return "MFI"
	case "cci":
		return "CCI"
	case "vwmacd":
		return "VWMACD"
	case "macd":
		return "MACD"
	default:
		return strings.ToUpper(name)
	}
}

func divergenceIndicatorOrder(label string) int {
	switch label {
	case "Hist":
		return 0
	case "RSI":
		return 1
	case "Stoch":
		return 2
	case "OBV":
		return 3
	case "CMF":
		return 4
	case "MOM":
		return 5
	case "MFI":
		return 6
	case "CCI":
		return 7
	case "VWMACD":
		return 8
	case "MACD":
		return 9
	default:
		return 99
	}
}

func drawDivergenceBubbles(img *image.RGBA, plot image.Rectangle, indexToX func(int) int, priceToY func(float64) int, klines []market.KlineBar, events []indicator.DivergenceChartEvent, plotStart, plotEnd int) {
	if img == nil || len(klines) < 2 || len(events) == 0 {
		return
	}
	if plotStart < 0 {
		plotStart = 0
	}
	if plotEnd > len(klines) {
		plotEnd = len(klines)
	}
	type bucket struct {
		Bull map[string]bool
		Bear map[string]bool
	}

	byIdx := make(map[int]*bucket, 16)
	for _, e := range events {
		idx := e.Index
		if idx < plotStart || idx >= plotEnd {
			continue
		}
		b := byIdx[idx]
		if b == nil {
			b = &bucket{Bull: make(map[string]bool), Bear: make(map[string]bool)}
			byIdx[idx] = b
		}
		lbl := divergenceIndicatorLabel(e.Indicator)
		isBull := e.Type == indicator.DivergencePositiveRegular || e.Type == indicator.DivergencePositiveHidden
		if isBull {
			b.Bull[lbl] = true
		} else {
			b.Bear[lbl] = true
		}
	}

	// Render in left-to-right order (older -> newer) for deterministic stacking.
	idxs := make([]int, 0, len(byIdx))
	for idx := range byIdx {
		idxs = append(idxs, idx)
	}
	sort.Ints(idxs)

	// Bear (top divergence): keep the original indigo style.
	bearFill := color.RGBA{R: 30, G: 27, B: 75, A: 230} // indigo-ish
	bearBorder := color.RGBA{R: 99, G: 102, B: 241, A: 220}
	bearText := color.RGBA{R: 234, G: 236, B: 239, A: 255}
	bearStem := color.RGBA{R: 246, G: 70, B: 93, A: 255}

	// Bull (bottom divergence): yellow style (like reference).
	bullFill := color.RGBA{R: 240, G: 185, B: 11, A: 235}  // yellow
	bullBorder := color.RGBA{R: 202, G: 138, B: 4, A: 235} // darker yellow
	bullText := color.RGBA{R: 17, G: 24, B: 39, A: 255}    // near-black
	bullStem := color.RGBA{R: 240, G: 185, B: 11, A: 255}  // yellow stem
	dot := color.RGBA{R: 148, G: 163, B: 184, A: 255}

	// Stack offsets per bar (if both bull and bear show up at same idx).
	for _, idx := range idxs {
		b := byIdx[idx]
		if b == nil {
			continue
		}
		plotIdx := idx - plotStart
		x := indexToX(plotIdx)

		// Helper to draw one bubble (above for bear, below for bull).
		drawOne := func(isBull bool, labelsMap map[string]bool, extraOffset int) {
			if len(labelsMap) == 0 {
				return
			}
			labels := make([]string, 0, len(labelsMap))
			for k := range labelsMap {
				labels = append(labels, k)
			}
			sort.Slice(labels, func(i, j int) bool {
				ai := divergenceIndicatorOrder(labels[i])
				aj := divergenceIndicatorOrder(labels[j])
				if ai == aj {
					return labels[i] < labels[j]
				}
				return ai < aj
			})

			// Lines: each indicator + final count (like the reference image).
			const maxNames = 8
			lines := make([]string, 0, maxNames+1)
			if len(labels) > maxNames {
				lines = append(lines, labels[:maxNames]...)
				lines = append(lines, "…")
			} else {
				lines = append(lines, labels...)
			}
			lines = append(lines, fmt.Sprintf("%d", len(labels)))

			// Bubble sizing (basicfont 7x13).
			maxLen := 0
			for _, s := range lines {
				if len(s) > maxLen {
					maxLen = len(s)
				}
			}
			padX := 10
			padTop := 10
			lineH := 14
			bubbleW := padX*2 + 7*maxLen
			bubbleH := padTop + lineH*len(lines) + 6
			if bubbleW < 64 {
				bubbleW = 64
			}

			stemLen := 18 + extraOffset
			anchorY := 0
			if isBull {
				anchorY = priceToY(klines[idx].Low) + 10
			} else {
				anchorY = priceToY(klines[idx].High) - 10
			}
			anchorY = clampInt(anchorY, plot.Min.Y+2, plot.Max.Y-2)

			x0 := x - bubbleW/2
			x1 := x0 + bubbleW - 1
			if x0 < plot.Min.X+2 {
				x0 = plot.Min.X + 2
				x1 = x0 + bubbleW - 1
			}
			if x1 > plot.Max.X-2 {
				x1 = plot.Max.X - 2
				x0 = x1 - (bubbleW - 1)
			}

			y0 := 0
			y1 := 0
			stemColor := bearStem
			fill := bearFill
			border := bearBorder
			txt := bearText
			if isBull {
				stemColor = bullStem
				fill = bullFill
				border = bullBorder
				txt = bullText
				y0 = anchorY + stemLen
				y1 = y0 + bubbleH - 1
				if y1 > plot.Max.Y-2 {
					// Clamp upward if it would go out of plot.
					shift := y1 - (plot.Max.Y - 2)
					y0 -= shift
					y1 -= shift
				}
				if y0 < plot.Min.Y+2 {
					y0 = plot.Min.Y + 2
					y1 = y0 + bubbleH - 1
				}
				drawLine(img, x, anchorY, x, y0, stemColor)
				drawFilledCircle(img, x, anchorY, 3, dot)
			} else {
				y1 = anchorY - stemLen
				y0 = y1 - (bubbleH - 1)
				if y0 < plot.Min.Y+2 {
					shift := (plot.Min.Y + 2) - y0
					y0 += shift
					y1 += shift
				}
				if y1 > plot.Max.Y-2 {
					y1 = plot.Max.Y - 2
					y0 = y1 - bubbleH
				}
				drawLine(img, x, y1, x, anchorY, stemColor)
				drawFilledCircle(img, x, anchorY, 3, dot)
			}

			drawRect(img, x0, y0, x1, y1, fill)
			drawLine(img, x0, y0, x1, y0, border)
			drawLine(img, x0, y1, x1, y1, border)
			drawLine(img, x0, y0, x0, y1, border)
			drawLine(img, x1, y0, x1, y1, border)

			ty := y0 + 14
			for _, line := range lines {
				drawText(img, x0+padX, ty, line, txt)
				ty += lineH
			}
		}

		// Draw bear above and bull below (independent).
		drawOne(false, b.Bear, 0)
		drawOne(true, b.Bull, 0)
	}
}

func drawSeriesLine(img *image.RGBA, indexToX func(int) int, toY func(float64) int, series []float64, c color.RGBA) {
	if len(series) < 2 {
		return
	}
	prevOK := false
	prevX, prevY := 0, 0
	for i := 0; i < len(series); i++ {
		v := series[i]
		if math.IsNaN(v) || math.IsInf(v, 0) {
			prevOK = false
			continue
		}
		x := indexToX(i)
		y := toY(v)
		if prevOK {
			drawLine(img, prevX, prevY, x, y, c)
		}
		prevOK = true
		prevX, prevY = x, y
	}
}

func rect(x0, y0, x1, y1 int) image.Rectangle {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	return image.Rect(x0, y0, x1, y1)
}

func join(items []string, sep string) string {
	if len(items) == 0 {
		return ""
	}
	out := items[0]
	for i := 1; i < len(items); i++ {
		out += sep + items[i]
	}
	return out
}

func drawRect(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	r := image.Rect(x0, y0, x1+1, y1+1).Intersect(img.Bounds())
	if r.Empty() {
		return
	}
	draw.Draw(img, r, &image.Uniform{C: c}, image.Point{}, draw.Src)
}

func drawLine(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	// Bresenham
	dx := int(math.Abs(float64(x1 - x0)))
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	dy := -int(math.Abs(float64(y1 - y0)))
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		if image.Pt(x0, y0).In(img.Bounds()) {
			img.SetRGBA(x0, y0, c)
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func drawText(img *image.RGBA, x, y int, text string, c color.RGBA) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: basicfont.Face7x13,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(text)
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
