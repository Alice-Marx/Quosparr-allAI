package glm

import (
	"math"
	"testing"
)

func TestLookupIsCaseInsensitive(t *testing.T) {
	for _, name := range []string{"glm-5.3", "GLM-5.3", " glm-5.3 ", "zai/glm-5.3"} {
		rate, ok := Lookup(name)
		if !ok {
			t.Fatalf("%q was not priced", name)
		}
		if rate.Input != 1.4 || rate.Output != 4.4 || rate.CacheRead != 0.26 {
			t.Fatalf("%q = %+v", name, rate)
		}
	}
}

func TestLookupReportsHighspeedAsUnpriced(t *testing.T) {
	for _, name := range []string{"glm-5.3-highspeed", "glm-5.2-highspeed", "", "unknown"} {
		if _, ok := Lookup(name); ok {
			t.Fatalf("%q was priced; a zero upstream cost is a data gap", name)
		}
	}
}

func TestPriceCostChargesCacheReadsSeparately(t *testing.T) {
	rate, _ := Lookup("glm-5.3")
	got := rate.Cost(1_000_000, 0, 0)
	if math.Abs(got-1.4) > 1e-9 {
		t.Fatalf("input only = %v", got)
	}
	got = rate.Cost(0, 0, 1_000_000)
	if math.Abs(got-0.26) > 1e-9 {
		t.Fatalf("cache only = %v", got)
	}
	got = rate.Cost(0, 1_000_000, 0)
	if math.Abs(got-4.4) > 1e-9 {
		t.Fatalf("output only = %v", got)
	}
	// 10M input, 1M output and 100M cache reads stay close to the ceiling of
	// the sample in the specification instead of the 65% overstatement that
	// a cache read priced as input produces.
	if got := rate.Cost(10_000_000, 1_000_000, 100_000_000); math.Abs(got-44.4) > 1e-6 {
		t.Fatalf("mixed = %v", got)
	}
}

func TestFormatUSDAndPercent(t *testing.T) {
	if got := FormatUSD(629.126); got != "$629.13" {
		t.Fatalf("usd = %q", got)
	}
	cases := map[float64]string{63: "63", 63.25: "63.3", 0: "0", 100: "100"}
	for value, want := range cases {
		if got := FormatPercent(value); got != want {
			t.Fatalf("%v = %q, want %q", value, got, want)
		}
	}
}
