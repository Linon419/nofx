package provider

import (
	"reflect"
	"testing"
)

func TestParseOTCTopResponseSortsAndFilters(t *testing.T) {
	body := []byte(`{
		"success": true,
		"items": [
			{"symbol": "ZEC", "otc_index": 10},
			{"symbol": "UNI", "otc_index": 30},
			{"symbol": "LDO", "otc_index": 30},
			{"symbol": " ", "otc_index": 50}
		]
	}`)

	items, err := parseOTCTopResponse(body)
	if err != nil {
		t.Fatalf("parseOTCTopResponse error: %v", err)
	}

	got := make([]string, 0, len(items))
	for _, item := range items {
		got = append(got, item.Symbol)
	}
	want := []string{"LDO", "UNI", "ZEC"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected order: got=%v want=%v", got, want)
	}
}

func TestParseOTCTopResponseFailure(t *testing.T) {
	body := []byte(`{"success": false, "items": []}`)
	if _, err := parseOTCTopResponse(body); err == nil {
		t.Fatalf("expected error for failure response")
	}
}
