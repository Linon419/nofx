package market

import "testing"

func TestBinanceFuturesSymbolAlias_PEPE(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"PEPEUSDT", "1000PEPEUSDT"},
		{"pepeusdt", "1000PEPEUSDT"},
		{"PEPE/USDT", "1000PEPEUSDT"},
		{"PEPE", "1000PEPEUSDT"},
		{"BTCUSDT", "BTCUSDT"},
	}

	for _, tt := range tests {
		if got := ToBinanceFuturesSymbol(tt.in); got != tt.want {
			t.Fatalf("ToBinanceFuturesSymbol(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}

	if got := FromBinanceFuturesSymbol("1000PEPEUSDT"); got != "PEPEUSDT" {
		t.Fatalf("FromBinanceFuturesSymbol(%q) = %q, want %q", "1000PEPEUSDT", got, "PEPEUSDT")
	}
	if got := FromBinanceFuturesSymbol("BTCUSDT"); got != "BTCUSDT" {
		t.Fatalf("FromBinanceFuturesSymbol(%q) = %q, want %q", "BTCUSDT", got, "BTCUSDT")
	}
}

