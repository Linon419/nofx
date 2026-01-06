package trader

import (
	"bytes"
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
	var planTypes []string

	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/api/v2/mix/order/place-plan-order" {
			return http200JSON(`{"code":"00000","msg":"ok","data":{}}`), nil
		}

		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		planType := ""
		s := string(b)
		if strings.Contains(s, `"planType":"loss_plan"`) {
			planType = "loss_plan"
		} else if strings.Contains(s, `"planType":"normal_plan"`) {
			planType = "normal_plan"
		}
		planTypes = append(planTypes, planType)

		if planType == "loss_plan" {
			return http200JSON(`{"code":"400172","msg":"planType Illegal type","data":{}}`), nil
		}
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
	if len(planTypes) != 2 || planTypes[0] != "loss_plan" || planTypes[1] != "normal_plan" {
		t.Fatalf("expected fallback loss_plan -> normal_plan, got: %#v", planTypes)
	}
}

func TestBitget_SetTakeProfit_PlanTypeFallback(t *testing.T) {
	var planTypes []string

	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/api/v2/mix/order/place-plan-order" {
			return http200JSON(`{"code":"00000","msg":"ok","data":{}}`), nil
		}

		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		planType := ""
		s := string(b)
		if strings.Contains(s, `"planType":"profit_plan"`) {
			planType = "profit_plan"
		} else if strings.Contains(s, `"planType":"normal_plan"`) {
			planType = "normal_plan"
		}
		planTypes = append(planTypes, planType)

		if planType == "profit_plan" {
			return http200JSON(`{"code":"400172","msg":"planType Illegal type","data":{}}`), nil
		}
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
	if len(planTypes) != 2 || planTypes[0] != "profit_plan" || planTypes[1] != "normal_plan" {
		t.Fatalf("expected fallback profit_plan -> normal_plan, got: %#v", planTypes)
	}
}

func TestBitget_placePlanOrderWithFallback_DoesNotRetryOnOtherErrors(t *testing.T) {
	var calls int
	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/api/v2/mix/order/place-plan-order" {
			return http200JSON(`{"code":"00000","msg":"ok","data":{}}`), nil
		}
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

	err := bt.placePlanOrderWithFallback("BSVUSDT", map[string]interface{}{"symbol": "BSVUSDT"}, []string{"loss_plan", "normal_plan"})
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

