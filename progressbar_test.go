package progressbar

import (
	"bytes"
	"regexp"
	"testing"
	"time"
)

// Golden outputs captured before the Render/BarString/RateETA refactor: Progress() output must not change.
func TestProgressOutput(t *testing.T) {
	var buf bytes.Buffer
	cfg := DefaultConfig()
	cfg.Width = 10
	cfg.UpdateInterval = 0
	cfg.ScreenWriter = &buf
	cfg.Prefix = "pre "
	cfg.Suffix = " suf"
	bar := cfg.NewBar()
	for _, tc := range []struct {
		percent float64
		out     string
	}{
		{0, "\rpre ⣾ \x1b[32;100m          \x1b[0m 0.0% suf"},
		{12.5, "\rpre ⣷ \x1b[32;100m█▎        \x1b[0m 12.5% suf"},
		{50, "\rpre ⣯ \x1b[32;100m█████     \x1b[0m 50.0% suf"},
		{100, "\rpre ✓ \x1b[32;100m██████████\x1b[0m 100.0% suf"},
		{-1, "\rpre ⡿  suf"},
	} {
		buf.Reset()
		bar.Progress(tc.percent)
		if buf.String() != tc.out {
			t.Errorf("Progress(%v) = %q, expected %q", tc.percent, buf.String(), tc.out)
		}
	}
	// No colors: delimiters instead, no spinner, no percent, with extra.
	buf.Reset()
	cfg.UseColors = false
	cfg.Spinner = false
	cfg.NoPercent = true
	cfg.Prefix, cfg.Suffix = "", ""
	bar = cfg.NewBar()
	bar.Extra = func(_ *Bar, _ float64) string { return " extra" }
	bar.Progress(25)
	if expected := "\r◅██▌       ▻ extra"; buf.String() != expected {
		t.Errorf("no color Progress(25) = %q, expected %q", buf.String(), expected)
	}
	// Multi bar index: relative cursor moves around the line.
	buf.Reset()
	bar.index = 2
	bar.Progress(25)
	if expected := "\r\x1b[2B◅██▌       ▻ extra\x1b[2A"; buf.String() != expected {
		t.Errorf("multi Progress(25) = %q, expected %q", buf.String(), expected)
	}
}

func TestAutoProgressExtra(t *testing.T) {
	a := &AutoProgress{Bar: NewBar(), total: 4 << 20, start: time.Now().Add(-2 * time.Second)}
	if got := a.Extra(nil, 0); got != " 0/4194304" {
		t.Errorf("unexpected start extra %q", got)
	}
	a.current = 1 << 20
	inProgress := regexp.MustCompile(`^ 1\.000 Mb out of 4\.000 Mb, 2s elapsed, 51[12].* Kb/s, 6s remaining  $`)
	if got := a.Extra(nil, 25); !inProgress.MatchString(got) {
		t.Errorf("unexpected in progress extra %q", got)
	}
	a.current = 4 << 20
	if got := a.Extra(nil, 100); got != " 4.000 Mb in 2s, 2.000 Mb/s\x1b[K" {
		t.Errorf("unexpected done extra %q", got)
	}
	a.total = -1
	a.current = 1 << 20
	if got := a.Extra(nil, -1); !regexp.MustCompile(`^ 1\.000 Mb, 2s elapsed, 51[12].* Kb/s  $`).MatchString(got) {
		t.Errorf("unexpected unknown total extra %q", got)
	}
}

func TestBarString(t *testing.T) {
	for _, tc := range []struct {
		width   int
		percent float64
		color   string
		out     string
	}{
		{4, 0, GreenBar, GreenBar + "    " + Reset},
		{4, 50, GreenBar, GreenBar + "██  " + Reset},
		{4, 100, BlueBar, BlueBar + "████" + Reset},
		{2, 37.5, "", "◅▊ ▻"}, // 0.75 block: 1/8th resolution.
		{4, -1, GreenBar, ""},
		{4, 101, GreenBar, ""},
	} {
		if got := BarString(tc.width, tc.percent, tc.color); got != tc.out {
			t.Errorf("BarString(%d, %v, %q) = %q, expected %q", tc.width, tc.percent, tc.color, got, tc.out)
		}
	}
}

func TestRender(t *testing.T) {
	var buf bytes.Buffer
	cfg := DefaultConfig()
	cfg.Width = 4
	cfg.ScreenWriter = &buf
	cfg.Prefix = "p "
	bar := cfg.NewBar()
	bar.Extra = func(_ *Bar, _ float64) string { return " x" }
	if got := bar.Render(50); got != "p ⣾ "+GreenBar+"██  "+Reset+" 50.0% x" {
		t.Errorf("unexpected render %q", got)
	}
	if got := bar.Render(50); got != "p ⣷ "+GreenBar+"██  "+Reset+" 50.0% x" { // spinner advanced.
		t.Errorf("unexpected 2nd render %q", got)
	}
	if buf.Len() != 0 {
		t.Errorf("Render should not output anything, got %q", buf.String())
	}
	// Same as what Progress outputs (minus cursor control).
	bar.Progress(100)
	if got := bar.Render(100); "\r"+got != buf.String() {
		t.Errorf("Render %q vs Progress %q", got, buf.String())
	}
}

func TestRateETA(t *testing.T) {
	for _, tc := range []struct {
		current, total int64
		elapsed        time.Duration
		out            string
	}{
		{0, 100, time.Second, "0/100"},
		{10, 100, 0, "10/100"},
		{1 << 20, 4 << 20, 2 * time.Second, "1.000 Mb out of 4.000 Mb, 2s elapsed, 512 Kb/s, 6s remaining"},
		{4 << 20, 4 << 20, 2 * time.Second, "4.000 Mb in 2s, 2.000 Mb/s"},
		{1 << 20, -1, 2 * time.Second, "1.000 Mb, 2s elapsed, 512 Kb/s"},
		{1536, 3072, 1500 * time.Millisecond, "1.5 Kb out of 3 Kb, 1.5s elapsed, 1 Kb/s, 1.5s remaining"},
	} {
		if got := RateETA(tc.current, tc.total, tc.elapsed); got != tc.out {
			t.Errorf("RateETA(%d, %d, %v) = %q, expected %q", tc.current, tc.total, tc.elapsed, got, tc.out)
		}
	}
}
