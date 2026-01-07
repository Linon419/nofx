package market

import "fmt"

// GetKlines fetches recent klines for the given symbol and timeframe.
func GetKlines(symbol, interval string, limit int) ([]Kline, error) {
	symbol = Normalize(symbol)
	if symbol == "" {
		return nil, fmt.Errorf("symbol is empty")
	}
	tf, err := NormalizeTimeframe(interval)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 200
	}

	var klines []Kline
	if IsXyzDexAsset(symbol) {
		// Hyperliquid does not support 3m; use 5m as the closest intraday proxy.
		if tf == "3m" {
			tf = "5m"
		}
		klines, err = getKlinesFromHyperliquid(symbol, tf, limit)
	} else {
		// Prefer Binance futures public klines for freshness + consistency.
		// Fall back to CoinAnk if Binance is unavailable/rate-limited.
		binanceSymbol := ToBinanceFuturesSymbol(symbol)
		klines, err = NewAPIClient().GetKlines(binanceSymbol, tf, limit)
		if err != nil {
			klines, err = getKlinesFromCoinAnk(symbol, tf, limit)
		}
	}
	if err != nil {
		return nil, err
	}
	return MarkKlinesClosed(klines, tf), nil
}
