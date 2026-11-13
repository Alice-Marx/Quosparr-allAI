package opencode

import (
	"errors"
	"testing"
	"time"
)

const usageBody = `{"usage":{
  "rolling":{"status":"ok","percent":1,"resetsAt":"2026-11-13T11:03:34.000Z"},
  "weekly" :{"status":"ok","percent":8,"resetsAt":"2026-11-16T00:00:00.000Z"},
  "monthly":{"status":"rate-limited","percent":75,"resetsAt":"2026-11-29T10:49:23.000Z"}}}`

func TestDecodeUsageReadsThreeWindows(t *testing.T) {
	usage, err := decodeUsage([]byte(usageBody))
	if err != nil {
		t.Fatal(err)
	}
	if !usage.Any() || usage.Rolling.Percent != 1 || usage.Weekly.Percent != 8 || usage.Monthly.Percent != 75 {
		t.Fatalf("usage = %+v", usage)
	}
	if usage.Rolling.Limited || !usage.Monthly.Limited {
		t.Fatalf("limited flags = %+v", usage)
	}
	want := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	if !usage.Weekly.HasReset || !usage.Weekly.ResetsAt.Equal(want) {
		t.Fatalf("weekly reset = %v", usage.Weekly.ResetsAt)
	}
}

func TestDecodeUsageClampsAndKeepsMissingFieldsMissing(t *testing.T) {
	usage, err := decodeUsage([]byte(`{"usage":{"rolling":{"status":"ok","percent":180},"weekly":{"status":"ok"},"monthly":{"status":"ok","percent":-4}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if usage.Rolling.Percent != 100 {
		t.Errorf("rolling = %v, want the clamp at 100", usage.Rolling.Percent)
	}
	if usage.Monthly.Percent != 0 {
		t.Errorf("monthly = %v, want the clamp at 0", usage.Monthly.Percent)
	}
	if usage.Weekly.HasPercent {
		t.Error("a window without a percent must stay missing, not become 0")
	}
	if usage.Rolling.HasReset {
		t.Error("a window without a reset must stay missing")
	}
}

func TestDecodeUsageRejectsAForeignAnswer(t *testing.T) {
	for _, body := range []string{`[]`, `{"usage":null}`, `not json`, `{}`} {
		if _, err := decodeUsage([]byte(body)); !errors.Is(err, ErrDecode) {
			t.Errorf("%q: err = %v", body, err)
		}
	}
}

func TestFormatPercentAndUSD(t *testing.T) {
	if got := FormatPercent(75); got != "75" {
		t.Errorf("FormatPercent(75) = %q", got)
	}
	if got := FormatPercent(8.25); got != "8.3" {
		t.Errorf("FormatPercent(8.25) = %q", got)
	}
	if got := FormatUSD(12.3456); got != "$12.35" {
		t.Errorf("FormatUSD(12.3456) = %q", got)
	}
	if got := FormatUSD(0.0332612); got != "$0.03" {
		t.Errorf("FormatUSD(0.0332612) = %q", got)
	}
	if got := FormatUSD(50); got != "$50" {
		t.Errorf("FormatUSD(50) = %q", got)
	}
}
