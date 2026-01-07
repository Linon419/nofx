package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"nofx/analysis/vision"
	"nofx/market"
)

func main() {
	var (
		symbol      = flag.String("symbol", "LDOUSDT", "symbol, e.g. LDOUSDT")
		timeframe   = flag.String("timeframe", "15m", "timeframe, e.g. 15m")
		outPath     = flag.String("out", "ctf-out/nofx_vision_preview.png", "output png path")
		width       = flag.Int("width", 1600, "image width")
		height      = flag.Int("height", 1396, "image height")
		maxBars     = flag.Int("maxbars", 300, "history bars to fetch")
		plotBars    = flag.Int("plotbars", 100, "bars to plot in window")
		showSqueeze = flag.Bool("squeeze", false, "enable squeeze overlay in volume panel")
	)
	flag.Parse()

	klines, err := market.GetKlines(*symbol, *timeframe, *maxBars)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetKlines failed: %v\n", err)
		os.Exit(1)
	}
	bars := make([]market.KlineBar, 0, len(klines))
	for _, k := range klines {
		bars = append(bars, market.KlineBar{
			Time:     k.OpenTime,
			Open:     k.Open,
			High:     k.High,
			Low:      k.Low,
			Close:    k.Close,
			Volume:   k.Volume,
			IsClosed: k.IsClosed,
		})
	}
	tf := &market.TimeframeSeriesData{Timeframe: *timeframe, Klines: bars}

	cfg := vision.RenderConfig{
		Width:    *width,
		Height:   *height,
		MaxBars:  *maxBars,
		PlotBars: *plotBars,
		Indicators: vision.IndicatorRenderConfig{
			ShowEMA:        true,
			ShowMACD:       true,
			ShowWaveTrend:  true,
			ShowSqueeze:    *showSqueeze,
			ShowDivergence: true,
		},
	}
	pngBytes, err := vision.RenderTimeframeChartPNG(*symbol, *timeframe, tf, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Render failed: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir failed: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*outPath, pngBytes, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (%d bytes)\n", *outPath, len(pngBytes))
}
