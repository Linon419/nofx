package market

import "strings"

// Internal symbols use the common "<BASE>USDT" form (e.g. PEPEUSDT).
// Some Binance Futures contracts use multiplier tickers (e.g. 1000PEPEUSDT).
//
// These helpers provide a narrow, exchange-specific alias mapping so:
// - core logic can keep using canonical internal symbols
// - Binance Futures + CoinAnk(Binance) calls use Binance's actual contract symbols

func normalizeCryptoSymbolLoose(symbol string) string {
	s := strings.ToUpper(strings.TrimSpace(symbol))
	if s == "" {
		return ""
	}

	// Strip common separators.
	replacer := strings.NewReplacer("/", "", ":", "", "-", "", "_", "", " ", "")
	s = replacer.Replace(s)

	// Ensure USDT quote.
	if !strings.HasSuffix(s, "USDT") {
		s += "USDT"
	}
	return s
}

// ToBinanceFuturesSymbol maps an internal canonical symbol to Binance Futures contract symbol.
func ToBinanceFuturesSymbol(symbol string) string {
	s := normalizeCryptoSymbolLoose(symbol)
	switch s {
	case "PEPEUSDT":
		return "1000PEPEUSDT"
	default:
		return s
	}
}

// FromBinanceFuturesSymbol maps a Binance Futures contract symbol back to internal canonical symbol.
func FromBinanceFuturesSymbol(symbol string) string {
	s := normalizeCryptoSymbolLoose(symbol)
	switch s {
	case "1000PEPEUSDT":
		return "PEPEUSDT"
	default:
		return s
	}
}

