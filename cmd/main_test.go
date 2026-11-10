package main

import "testing"

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
