package trader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func http200JSON(body string) *http.Response {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

func TestBitget_SetStopLoss_PlanTypeFallback(t *testing.T) {
	var endpoints []string

	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		endpoints = append(endpoints, r.URL.Path)
		return http200JSON(`{"code":"00000","msg":"ok","data":{}}`), nil
	})

	bt := &BitgetTrader{
		apiKey:             "k",
		secretKey:          "s",
		passphrase:         "p",
		httpClient:         &http.Client{Transport: rt, Timeout: 5 * time.Second},
		cacheDuration:      time.Hour,
		contractsCache:     map[string]*BitgetContract{"BSVUSDT": {Symbol: "BSVUSDT", VolumePlace: 2}},
		contractsCacheTime: time.Now(),
		marginModeCache:    map[string]string{"BSVUSDT": "crossed"},
	}

	if err := bt.SetStopLoss("BSVUSDT", "LONG", 3.74, 20.30); err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if len(endpoints) != 1 || endpoints[0] != "/api/mix/v1/plan/placePositionsTPSL" {
		t.Fatalf("expected position TPSL endpoint, got: %#v", endpoints)
	}
}

func TestBitget_SetTakeProfit_PlanTypeFallback(t *testing.T) {
	var endpoints []string

	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		endpoints = append(endpoints, r.URL.Path)
		return http200JSON(`{"code":"00000","msg":"ok","data":{}}`), nil
	})

	bt := &BitgetTrader{
		apiKey:             "k",
		secretKey:          "s",
		passphrase:         "p",
		httpClient:         &http.Client{Transport: rt, Timeout: 5 * time.Second},
		cacheDuration:      time.Hour,
		contractsCache:     map[string]*BitgetContract{"BSVUSDT": {Symbol: "BSVUSDT", VolumePlace: 2}},
		contractsCacheTime: time.Now(),
		marginModeCache:    map[string]string{"BSVUSDT": "crossed"},
	}

	if err := bt.SetTakeProfit("BSVUSDT", "LONG", 3.74, 20.76); err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if len(endpoints) != 1 || endpoints[0] != "/api/mix/v1/plan/placePositionsTPSL" {
		t.Fatalf("expected position TPSL endpoint, got: %#v", endpoints)
	}
}

func TestBitget_SetStopLoss_SendsTriggerTypeAndFormattedTriggerPrice(t *testing.T) {
	var got map[string]interface{}

	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		_ = json.Unmarshal(b, &got)
		return http200JSON(`{"code":"00000","msg":"ok","data":{}}`), nil
	})

	bt := &BitgetTrader{
		apiKey:             "k",
		secretKey:          "s",
		passphrase:         "p",
		httpClient:         &http.Client{Transport: rt, Timeout: 5 * time.Second},
		cacheDuration:      time.Hour,
		contractsCache:     map[string]*BitgetContract{"BSVUSDT": {Symbol: "BSVUSDT", VolumePlace: 2, PricePlace: 2}},
		contractsCacheTime: time.Now(),
		marginModeCache:    map[string]string{"BSVUSDT": "crossed"},
	}

	if err := bt.SetStopLoss("BSVUSDT", "LONG", 3.74, 20.3061); err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if got["triggerType"] != "market_price" {
		t.Fatalf("expected triggerType=market_price, got: %#v", got["triggerType"])
	}
	if got["triggerPrice"] != "20.31" {
		t.Fatalf("expected triggerPrice=20.31, got: %#v", got["triggerPrice"])
	}
}

func TestBitget_placePlanOrderWithFallback_DoesNotRetryOnOtherErrors(t *testing.T) {
	var calls int
	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return http200JSON(`{"code":"400000","msg":"some other error","data":{}}`), nil
	})

	bt := &BitgetTrader{
		apiKey:             "k",
		secretKey:          "s",
		passphrase:         "p",
		httpClient:         &http.Client{Transport: rt, Timeout: 5 * time.Second},
		cacheDuration:      time.Hour,
		contractsCache:     map[string]*BitgetContract{"BSVUSDT": {Symbol: "BSVUSDT", VolumePlace: 2}},
		contractsCacheTime: time.Now(),
		marginModeCache:    map[string]string{"BSVUSDT": "crossed"},
	}

	err := bt.placePlanOrderWithFallback("BSVUSDT", map[string]interface{}{
		"symbol":       "BSVUSDT",
		"productType":  "USDT-FUTURES",
		"marginCoin":   "USDT",
		"triggerPrice": "20.30",
		"holdSide":     "long",
		"size":         "3.74",
	}, []string{"loss_plan", "normal_plan"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if calls != 1 {
		t.Fatalf("expected 1 call (no retry), got %d", calls)
	}
}

func TestBitget_isBitgetPlanTypeIllegal(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{err: nil, want: false},
		{err: errString("Bitget API error: code=400172, msg=planType Illegal type"), want: true},
		{err: errString("planType illegal type"), want: true},
		{err: errString("Bitget API error: code=401, msg=unauthorized"), want: false},
	}
	for _, tc := range cases {
		if got := isBitgetPlanTypeIllegal(tc.err); got != tc.want {
			t.Fatalf("isBitgetPlanTypeIllegal(%v)=%v want=%v", tc.err, got, tc.want)
		}
	}
}

type errString string

func (e errString) Error() string { return string(e) }

func TestBitget_OpenLongWithPresetTPSL_SendsPresetParams(t *testing.T) {
	var placeOrderBodies []map[string]interface{}

	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/api/v2/mix/order/orders-pending":
			return http200JSON(`{"code":"00000","msg":"ok","data":{"entrustedList":[]}}`), nil
		case "/api/v2/mix/order/orders-plan-pending":
			return http200JSON(`{"code":"00000","msg":"ok","data":{"entrustedList":[]}}`), nil
		case "/api/v2/mix/account/set-leverage":
			return http200JSON(`{"code":"00000","msg":"ok","data":{}}`), nil
		case "/api/v2/mix/order/place-order":
			b, _ := io.ReadAll(r.Body)
			_ = r.Body.Close()
			var got map[string]interface{}
			_ = json.Unmarshal(b, &got)
			placeOrderBodies = append(placeOrderBodies, got)
			return http200JSON(`{"code":"00000","msg":"ok","data":{"orderId":"1","clientOid":"cid"}}`), nil
		default:
			return http200JSON(`{"code":"00000","msg":"ok","data":{}}`), nil
		}
	})

	bt := &BitgetTrader{
		apiKey:             "k",
		secretKey:          "s",
		passphrase:         "p",
		httpClient:         &http.Client{Transport: rt, Timeout: 5 * time.Second},
		cacheDuration:      time.Hour,
		contractsCache:     map[string]*BitgetContract{"BSVUSDT": {Symbol: "BSVUSDT", VolumePlace: 2, PricePlace: 2}},
		contractsCacheTime: time.Now(),
		marginModeCache:    map[string]string{"BSVUSDT": "crossed"},
	}

	_, slApplied, tpApplied, err := bt.OpenLongWithPresetTPSL("BSVUSDT", 3.74, 10, 20.30, 20.76)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !slApplied || !tpApplied {
		t.Fatalf("expected both presets applied, got sl=%v tp=%v", slApplied, tpApplied)
	}
	if len(placeOrderBodies) != 1 {
		t.Fatalf("expected 1 place-order call, got %d", len(placeOrderBodies))
	}
	if placeOrderBodies[0]["presetStopLossPrice"] != "20.30" {
		t.Fatalf("expected presetStopLossPrice=20.30, got %#v", placeOrderBodies[0]["presetStopLossPrice"])
	}
	if placeOrderBodies[0]["presetStopSurplusPrice"] != "20.76" {
		t.Fatalf("expected presetStopSurplusPrice=20.76, got %#v", placeOrderBodies[0]["presetStopSurplusPrice"])
	}
}

func TestBitget_OpenLongWithPresetTPSL_FallsBackOnAPIError(t *testing.T) {
	var placeOrderBodies []map[string]interface{}
	var placeCalls int

	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/api/v2/mix/order/orders-pending":
			return http200JSON(`{"code":"00000","msg":"ok","data":{"entrustedList":[]}}`), nil
		case "/api/v2/mix/order/orders-plan-pending":
			return http200JSON(`{"code":"00000","msg":"ok","data":{"entrustedList":[]}}`), nil
		case "/api/v2/mix/account/set-leverage":
			return http200JSON(`{"code":"00000","msg":"ok","data":{}}`), nil
		case "/api/v2/mix/order/place-order":
			b, _ := io.ReadAll(r.Body)
			_ = r.Body.Close()
			var got map[string]interface{}
			_ = json.Unmarshal(b, &got)
			placeOrderBodies = append(placeOrderBodies, got)

			placeCalls++
			if placeCalls == 1 {
				return http200JSON(`{"code":"400000","msg":"unknown field presetStopLossPrice","data":{}}`), nil
			}
			return http200JSON(`{"code":"00000","msg":"ok","data":{"orderId":"2","clientOid":"cid"}}`), nil
		default:
			return http200JSON(`{"code":"00000","msg":"ok","data":{}}`), nil
		}
	})

	bt := &BitgetTrader{
		apiKey:             "k",
		secretKey:          "s",
		passphrase:         "p",
		httpClient:         &http.Client{Transport: rt, Timeout: 5 * time.Second},
		cacheDuration:      time.Hour,
		contractsCache:     map[string]*BitgetContract{"BSVUSDT": {Symbol: "BSVUSDT", VolumePlace: 2, PricePlace: 2}},
		contractsCacheTime: time.Now(),
		marginModeCache:    map[string]string{"BSVUSDT": "crossed"},
	}

	_, slApplied, tpApplied, err := bt.OpenLongWithPresetTPSL("BSVUSDT", 3.74, 10, 20.30, 20.76)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if slApplied || tpApplied {
		t.Fatalf("expected presets not applied after fallback, got sl=%v tp=%v", slApplied, tpApplied)
	}
	if len(placeOrderBodies) != 2 {
		t.Fatalf("expected 2 place-order calls, got %d", len(placeOrderBodies))
	}
	if _, ok := placeOrderBodies[0]["presetStopLossPrice"]; !ok {
		t.Fatalf("expected first request to include presetStopLossPrice, got keys=%v", keysOf(placeOrderBodies[0]))
	}
	if _, ok := placeOrderBodies[0]["presetStopSurplusPrice"]; !ok {
		t.Fatalf("expected first request to include presetStopSurplusPrice, got keys=%v", keysOf(placeOrderBodies[0]))
	}
	if _, ok := placeOrderBodies[1]["presetStopLossPrice"]; ok {
		t.Fatalf("expected fallback request to omit presetStopLossPrice, got %#v", placeOrderBodies[1]["presetStopLossPrice"])
	}
	if _, ok := placeOrderBodies[1]["presetStopSurplusPrice"]; ok {
		t.Fatalf("expected fallback request to omit presetStopSurplusPrice, got %#v", placeOrderBodies[1]["presetStopSurplusPrice"])
	}
}

func TestBitget_OpenLongWithPresetTPSL_DoesNotRetryOnTransportError(t *testing.T) {
	var placeCalls int

	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/api/v2/mix/order/orders-pending":
			return http200JSON(`{"code":"00000","msg":"ok","data":{"entrustedList":[]}}`), nil
		case "/api/v2/mix/order/orders-plan-pending":
			return http200JSON(`{"code":"00000","msg":"ok","data":{"entrustedList":[]}}`), nil
		case "/api/v2/mix/account/set-leverage":
			return http200JSON(`{"code":"00000","msg":"ok","data":{}}`), nil
		case "/api/v2/mix/order/place-order":
			placeCalls++
			return nil, fmt.Errorf("dial tcp timeout")
		default:
			return http200JSON(`{"code":"00000","msg":"ok","data":{}}`), nil
		}
	})

	bt := &BitgetTrader{
		apiKey:             "k",
		secretKey:          "s",
		passphrase:         "p",
		httpClient:         &http.Client{Transport: rt, Timeout: 5 * time.Second},
		cacheDuration:      time.Hour,
		contractsCache:     map[string]*BitgetContract{"BSVUSDT": {Symbol: "BSVUSDT", VolumePlace: 2, PricePlace: 2}},
		contractsCacheTime: time.Now(),
		marginModeCache:    map[string]string{"BSVUSDT": "crossed"},
	}

	_, _, _, err := bt.OpenLongWithPresetTPSL("BSVUSDT", 3.74, 10, 20.30, 20.76)
	if err == nil {
		t.Fatalf("expected error")
	}
	if placeCalls != 1 {
		t.Fatalf("expected 1 place-order attempt (no retry), got %d", placeCalls)
	}
}

func keysOf(m map[string]interface{}) string {
	if m == nil {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return strings.Join(keys, ",")
}

func TestHTTP200JSONHelper(t *testing.T) {
	resp := http200JSON(`{"ok":true}`)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200")
	}
	b, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if !bytes.Contains(b, []byte(`"ok"`)) {
		t.Fatalf("unexpected body: %s", string(b))
	}
}
