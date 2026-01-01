package decision

import (
	"fmt"
	"nofx/market"
	"nofx/provider"
	"strings"
	"time"
)

type otcPeriodQualityMeta struct {
	Quality string
	Time    string
	Expired bool
}

func buildOTCPeriodQualityMeta(item provider.OTCTopItem, now time.Time) otcPeriodQualityMeta {
	quality := strings.TrimSpace(item.PeriodQuality)
	if quality == "" {
		return otcPeriodQualityMeta{}
	}

	meta := otcPeriodQualityMeta{
		Quality: quality,
	}

	if !item.AsOf.IsZero() {
		asOf := item.AsOf.UTC()
		meta.Time = asOf.Format(time.RFC3339)
		meta.Expired = now.Sub(asOf) > 24*time.Hour
		return meta
	}

	if strings.TrimSpace(item.AsOfRaw) != "" {
		meta.Time = strings.TrimSpace(item.AsOfRaw)
	}

	return meta
}

func hasSource(sources []string, source string) bool {
	for _, item := range sources {
		if item == source {
			return true
		}
	}
	return false
}

func formatOTCPeriodQualityLine(coin CandidateCoin) string {
	if !hasSource(coin.Sources, "otc_top") {
		return ""
	}
	quality := strings.TrimSpace(coin.PeriodQuality)
	if quality == "" {
		return ""
	}

	line := fmt.Sprintf("period_quality = %s", quality)
	timeValue := strings.TrimSpace(coin.PeriodQualityTime)
	if timeValue != "" {
		line += fmt.Sprintf(" | as_of = %s", timeValue)
		if coin.PeriodQualityExpired {
			line += " | expired = true"
		} else {
			line += " | expired = false"
		}
	} else {
		line += " | expired = unknown"
	}
	return line
}

func findOTCTopCandidate(ctx *Context, symbol string) *CandidateCoin {
	normalized := market.Normalize(symbol)
	for i := range ctx.CandidateCoins {
		coin := &ctx.CandidateCoins[i]
		if market.Normalize(coin.Symbol) == normalized && hasSource(coin.Sources, "otc_top") {
			return coin
		}
	}
	return nil
}
