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
		klines, err = getKlinesFromHyperliquid(symbol, tf, limit)
	} else {
		klines, err = getKlinesFromCoinAnk(symbol, tf, limit)
	}
	if err != nil {
		return nil, err
	}
	return DropUnclosedKlines(klines, tf), nil
}
