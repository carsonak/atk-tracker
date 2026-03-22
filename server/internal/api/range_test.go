package api

import (
	"net/http"
	"net/url"
	"testing"
)

func TestParseRangeDefaults(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: ""}}
	from, to, err := parseRange(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !to.After(from) {
		t.Fatal("expected to > from")
	}
}

func TestParseRangeCustom(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: "from=2026-03-01&to=2026-03-07"}}
	from, to, err := parseRange(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if from.Format("2006-01-02") != "2026-03-01" {
		t.Fatalf("unexpected from %s", from)
	}
	if to.Format("2006-01-02") != "2026-03-08" {
		t.Fatalf("unexpected to %s", to)
	}
}
