package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIsQuietHours(t *testing.T) {
	cases := map[int]bool{0: false, 1: false, 2: true, 5: true, 6: false, 23: false}
	for h, want := range cases {
		tm := time.Date(2026, 1, 1, h, 30, 0, 0, time.UTC)
		if got := isQuietHours(tm); got != want {
			t.Errorf("hour %d: got %v, want %v", h, got, want)
		}
	}
}

func TestIsQuietHoursUsesUTC(t *testing.T) {
	loc := time.FixedZone("UTC+3", 3*3600)
	tm := time.Date(2026, 1, 1, 6, 0, 0, 0, loc)
	if !isQuietHours(tm) {
		t.Error("06:00 UTC+3 is 03:00 UTC and should be quiet")
	}
}

func TestEffectiveInterval(t *testing.T) {
	quiet := time.Date(2026, 1, 1, 3, 0, 0, 0, time.UTC)
	busy := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	if got := effectiveInterval(quiet, time.Minute); got != 5*time.Minute {
		t.Errorf("quiet: got %v", got)
	}
	if got := effectiveInterval(busy, time.Minute); got != time.Minute {
		t.Errorf("busy: got %v", got)
	}
}

func TestPriceFilter(t *testing.T) {
	cases := []struct {
		min, max int
		want     string
	}{
		{0, 0, ""},
		{100, 0, "price:[100..],priceCurrency:GBP"},
		{0, 250, "price:[..250],priceCurrency:GBP"},
		{100, 250, "price:[100..250],priceCurrency:GBP"},
	}
	for _, c := range cases {
		if got := priceFilter(c.min, c.max); got != c.want {
			t.Errorf("priceFilter(%d, %d) = %q, want %q", c.min, c.max, got, c.want)
		}
	}
}

func TestPrimedKeyChangesWithFilters(t *testing.T) {
	a := search{Query: "x", MinPrice: 10}
	b := search{Query: "x", MinPrice: 20}
	if primedKey(a) == primedKey(b) {
		t.Error("changing min price should produce a different primed key")
	}
}

func writeSearches(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "searches.json")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadSearches(t *testing.T) {
	p := writeSearches(t, `[{"query":"Wooting","maxPrice":150,"interval":"90s"}]`)
	got, err := loadSearches(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d searches", len(got))
	}
	s := got[0]
	if s.Name != "Wooting" {
		t.Errorf("name should default to query, got %q", s.Name)
	}
	if time.Duration(s.Interval) != 90*time.Second {
		t.Errorf("interval: got %v", time.Duration(s.Interval))
	}
	if s.MaxPrice != 150 {
		t.Errorf("maxPrice: got %d", s.MaxPrice)
	}
}

func TestLoadSearchesErrors(t *testing.T) {
	cases := map[string]string{
		"empty list":       `[]`,
		"missing query":    `[{"name":"a","interval":"1m"}]`,
		"missing interval": `[{"query":"a"}]`,
		"bad interval":     `[{"query":"a","interval":"soon"}]`,
		"numeric interval": `[{"query":"a","interval":60}]`,
		"invalid json":     `{`,
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := loadSearches(writeSearches(t, content)); err == nil {
				t.Error("expected error")
			}
		})
	}
	if _, err := loadSearches(filepath.Join(t.TempDir(), "nope.json")); err == nil {
		t.Error("expected error for missing file")
	}
}
