// Package progressbar is a simple zero dependency cross platform (only need ANSI(*) compatible terminal)
// progress bar library for your golang terminal / command line interface (CLI) applications. Shows a spinner
// and/or a progress bar with optional prefix and extra info.
// The package provides reader/writer wrappers to automatically show progress of downloads/uploads
// or other io operations. As well as a Writer that can be used concurrently with the progress bar
// to show other output on screen.
//
// ANSI codes can be disabled completely for non terminal output or testing or as needed.
// Color is enabled by default but can also be disabled.
package progressbar

import (
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	DefaultWidth = 40
	Space        = " "
	Full         = "█"
	// Green FG, Grey BG.

	GreenBar     = "\033[32;100m"
	RedBar       = "\033[91;100m"
	YellowBar    = "\033[93;100m"
	BlueBar      = "\033[34;100m"
	WhiteBar     = "\033[97;100m"
	DefaultColor = GreenBar
	Reset        = "\033[0m"
	ClearAfter   = "\033[J"
	DoneSpinner  = "✓ "
	// DefaultMaxUpdateInterval is the default max refresh to avoid slowing down transfers because of progress bar updates.
	DefaultMaxUpdateInterval = 100 * time.Millisecond
	// ExpectedMaxLength is the expected max length of a progress bar line (prefix + spinner + bar + percentage + extra).
	// used for initial buffer size, will resize if needed but should be avoided. Note it includes
	// non printable ANSI sequences and utf8 encoded characters so it's the same as the onscreen width.
	ExpectedMaxLength = 256
)

var (
	// 1/8th of a full block to 7/8ths of a full block (ie fractional part of a block to
	// get 8x resolution per character).

	FractionalBlocks = [...]string{"", "▏", "▎", "▍", "▌", "▋", "▊", "▉"}
	SpinnerChars     = [...]string{"⣾ ", "⣷ ", "⣯ ", "⣟ ", "⡿ ", "⢿ ", "⣻ ", "⣽ "}
)

// Config is the common configuration for the progress bar and multi bars.
type Config struct {
	// Width of the progress bar in characters (0 will use DefaultWidth).
	Width int
	// UseColors to use colors in the progress bar (default is true).
	UseColors bool
	// Which color to use for the bar (default is green) if UseColors is true.
	Color string
	// Spinner to also show a spinner in front of the progress bar.
	Spinner bool
	// Prefix to show before the progress bar (can be updated while running using UpdatePrefix() or through Extra()).
	Prefix string
	// Suffix to show after the percentage information on the bar (can be updated while running using UpdateSuffix()).
	Suffix string
	// If NoPercent is true, the percentage is not shown on the bar (if the default %.1f%% format is not adequate).
	NoPercent bool
	// Minimum duration between updates (0 to update every time).
	UpdateInterval time.Duration
	// Option to avoid all ANSI sequences (useful for non terminal output/test/go playground),
	// Implies UseColors=false. Not applicable to multi bars (which need ANSI cursor moves to update each bar).
	NoAnsi bool
	// Underlying specific destination writer for the screen/terminal.
	// (defaults when nil will use the shared screen writer based on os.Stderr).
	// Sets to os.Stdout or os.Stderr or any other Writer (that ends up outputting to ANSI aware terminal) to use
	// this with your existing code if the os.Stderr default global shared screen writer doesn't work for you.
	ScreenWriter io.Writer
	// Extra lines between each bar for multibars.
	ExtraLines int
}

type Bar struct {
	Config
	// Extra string to show after the progress bar. Keep nil for no extra.
	Extra func(cfg *Bar, progressPercent float64) string
	// Internal last update time (used to skip updates coming before UpdateInterval has elapsed).
	lastUpdate time.Time
	// Writer to write to.
	out *writer
	// Index for multi bar (to move the cursor up/down).
	index int
	// Current/last progress percentage (to refresh multi bars upon resize of prefix).
	percent float64
}

// UpdatePrefix changes the prefix while the progress bar is running.
// This is thread safe / acquires a shared lock to avoid issues on the output.
func (bar *Bar) UpdatePrefix(p string) {
	bar.out.Lock()
	bar.Prefix = p
	bar.out.Unlock()
}

// UpdateSuffix changes the suffix while the progress bar is running.
// This is thread safe / acquires a shared lock to avoid issues on the output.
func (bar *Bar) UpdateSuffix(s string) {
	bar.out.Lock()
	bar.Suffix = s
	bar.out.Unlock()
}

// Progress shows a progress bar percentage (0-100%). On the same line as current line,
// so when call repeatedly it will overwrite/update itself.
// Use MoveCursorUp to move up to update other lines as needed or use Writer()
// to write output without mixing with a progress bar.
// This is thread safe / acquires a shared lock to avoid issues on the output.
// Of note it will work best if every output to the Writer() ends with a \n.
// The bar state must be obtained from NewBar() or cfg.NewBar() to setup the shared lock.
func (bar *Bar) Progress(progressPercent float64) {
	bar.out.Lock()
	bar.progressLocked(progressPercent)
	bar.out.Unlock()
}

func (bar *Bar) progressLocked(progressPercent float64) {
	isDone := isDone(progressPercent)
	bar.percent = progressPercent
	// Skip if last write was too recent and we're not done and nothing else was written in between.
	if bar.UpdateInterval > 0 && !isDone && bar.out.needErase {
		now := time.Now()
		if now.Sub(bar.lastUpdate) < bar.UpdateInterval {
			return
		}
		bar.lastUpdate = now
	}
	barStr := ""
	progressPercentStr := ""
	if progressPercent >= 0 && progressPercent <= 100 { //nolint:nestif // it's not that bad here
		width := float64(bar.Width)
		if width == 0 {
			width = DefaultWidth
		}
		count := int(8*width*progressPercent/100. + 0.5)
		fullCount := count / 8
		remainder := count % 8
		spaceCount := int(width) - fullCount - 1
		if remainder == 0 {
			spaceCount++
		}
		color := bar.Color
		reset := Reset
		if !bar.UseColors || bar.NoAnsi {
			color = "◅" // "◢"
			reset = "▻" // "◣"
		}
		barStr = color + strings.Repeat(Full, fullCount) + FractionalBlocks[remainder] + strings.Repeat(Space, spaceCount) + reset
		if !bar.NoPercent {
			progressPercentStr = fmt.Sprintf(" %.1f%%", progressPercent)
		}
	}
	spinner := ""
	if bar.Spinner {
		spinner = SpinnerChars[bar.out.count]
		bar.out.count = (bar.out.count + 1) % len(SpinnerChars)
		if isDone {
			spinner = DoneSpinner
		}
	}
	extra := ""
	if bar.Extra != nil {
		extra = bar.Extra(bar, progressPercent)
	}
	bar.out.buf = bar.out.buf[:0]
	bar.out.buf = append(bar.out.buf, bar.indexBasedMoveDown()...) // does \r in single bar mode.
	bar.out.buf = append(bar.out.buf, bar.Prefix...)
	bar.out.buf = append(bar.out.buf, spinner...)
	bar.out.buf = append(bar.out.buf, barStr...)
	bar.out.buf = append(bar.out.buf, progressPercentStr...)
	bar.out.buf = append(bar.out.buf, bar.Suffix...)
	bar.out.buf = append(bar.out.buf, extra...)
	bar.out.buf = append(bar.out.buf, bar.indexBasedMoveUp()...)
	// bar.out.buf = append(bar.out.buf, '\n') // Uncomment to debug/see all the incremental updates.
	_, _ = bar.out.out.Write(bar.out.buf)
	bar.out.buf = bar.out.buf[:0]
	bar.out.needErase = true
	bar.out.noAnsi = bar.NoAnsi
}

// Redraw force the redraw the progress bar with last known percentage.
func (bar *Bar) Redraw() {
	var zero time.Time
	bar.lastUpdate = zero
	bar.Progress(bar.percent)
}

// Approximate check if the progress is done (percent > 99.999).
func isDone(percent float64) bool {
	return percent > 99.999
}

// Spinner is a standalone spinner when the total or progress toward 100% isn't known.
// (but a progressbar with -1 total or with negative % progress does that too).
func Spinner() {
	screenWriter.Lock()
	fmt.Fprintf(screenWriter, "\r%s", SpinnerChars[screenWriter.count])
	screenWriter.count = (screenWriter.count + 1) % len(SpinnerChars)
	screenWriter.Unlock()
}

// MoveCursorUp moves the cursor up n lines and clears that line.
// If NoAnsi is configured, this just issue a new line.
func (bar *Bar) MoveCursorUp(n int) {
	if bar.NoAnsi {
		fmt.Fprintf(bar.out.out, "\n")
		return
	}
	// ANSI escape codes used:
	// nA   = move up n lines
	// \r   = beginning of the line
	// (0)K = erase from current position to end of line
	fmt.Fprintf(bar.out.out, "\033[%dA\r\033[K", n)
}

// WriteAbove is for multibars with extra lines, writes (1 line) above the bar.
func (bar *Bar) WriteAbove(msg string) {
	bar.out.Lock()
	if bar.index > 0 {
		fmt.Fprintf(bar.out.out, "\r\033[%dB%s\n%s", bar.index-1, msg, bar.indexBasedMoveUp())
	} else {
		fmt.Fprintf(bar.out.out, "\r\033[1A%s\n", msg)
	}
	bar.out.Unlock()
}

func (bar *Bar) indexBasedMoveUp() []byte {
	if bar.index <= 0 || bar.NoAnsi {
		return nil
	}
	return []byte(fmt.Sprintf("\033[%dA", bar.index))
}

func (bar *Bar) indexBasedMoveDown() []byte {
	if bar.index <= 0 || bar.NoAnsi {
		return []byte{'\r'}
	}
	return []byte(fmt.Sprintf("\r\033[%dB", bar.index))
}

type writer struct {
	sync.Mutex
	out       io.Writer
	buf       []byte
	count     int
	needErase bool
	noAnsi    bool
}

func (w *writer) Write(buf []byte) (n int, err error) {
	w.Lock()
	if w.needErase {
		if w.noAnsi {
			_, _ = w.out.Write([]byte("\r")) // just carriage return and pray it's enough
		} else {
			_, _ = w.out.Write([]byte("\r\033[K")) // erase current progress bar line
		}
		w.needErase = false
	}
	n, err = w.out.Write(buf)
	w.Unlock()
	return n, err
}

// Global write with lock and reused buffer.
// Outside of testing there is generally only 1 valid output for ansi progress bar:
// os.Stdout or os.Stderr.
var screenWriter = &writer{out: os.Stderr, buf: make([]byte, 0, ExpectedMaxLength)}

// Writer returns the io.Writer that can be safely used concurrently with associated with the progress bar.
// Any writes will clear the current line/progress bar and write the new content, and
// then rewrite the progress bar at the next update.
func (bar *Bar) Writer() io.Writer {
	return bar.out
}

// HumanBytes shows bytes in `b`, `Kb`, `Mb`, `Gb` and can also be used for speed/rate
// (by appending "/s") in addition to raw bytes quantities.
func HumanBytes[T int64 | float64](inp T) string {
	n := float64(inp)
	if n < 1024 {
		return fmt.Sprintf("%.0f b", n)
	}
	n /= 1024
	if n < 1024 {
		// io.Copy etc tends to round number of Kb so let's not add .0 uncessarily.
		if math.Floor(n) == n {
			return fmt.Sprintf("%.0f Kb", n)
		}
		return fmt.Sprintf("%.1f Kb", n)
	}
	n /= 1024
	if n < 1024 {
		return fmt.Sprintf("%.3f Mb", n)
	}
	n /= 1024
	return fmt.Sprintf("%.3f Gb", n)
}

// HumanDuration shows a duration rounded according to it's order of magnitude (minute/100ms/1ms resolution).
func HumanDuration(d time.Duration) string {
	if d <= time.Second {
		return d.Round(time.Millisecond).String()
	}
	if d < time.Hour {
		return d.Round(100 * time.Millisecond).String()
	}
	return d.Round(time.Minute).String()
}

// AutoProgress is base progress bar for auto reader/writers.
type AutoProgress struct {
	*Bar
	total   int64
	current int64
	start   time.Time
}

// NewAutoProgress returns a new AutoProgress associated with
// the given bar and total size. [AutoProgress.Update] can be called
// from multiple goroutines to increment the progress by <n>.
func NewAutoProgress(bar *Bar, total int64) *AutoProgress {
	res := &AutoProgress{}
	res.Bar = bar
	res.total = total
	return res
}

func (a *AutoProgress) Update(n int) {
	a.out.Lock()
	if a.current == 0 {
		a.start = time.Now()
	}
	a.current += int64(n)
	if a.current > 0 || a.total > 0 {
		a.progressLocked(float64(a.current) * 100. / float64(a.total))
	}
	a.out.Unlock()
}

// Extra provides the extra information on the right of the progress bar: currrent transfer amount, speed and estimated time left.
func (a *AutoProgress) Extra(_ *Bar, progressPercent float64) string {
	elapsed := time.Since(a.start)
	if a.current == 0 {
		return fmt.Sprintf(" %d/%d", a.current, a.total)
	}
	speed := float64(a.current) / elapsed.Seconds()
	switch {
	case a.total <= 0:
		// No total, show current, elapsed and speed.
		return fmt.Sprintf(" %s, %v elapsed, %s/s  ",
			HumanBytes(a.current), elapsed.Round(time.Millisecond), HumanBytes(speed))
	case !isDone(progressPercent):
		bytesLeft := a.total - a.current
		timeLeft := time.Duration(float64(time.Second) * float64(bytesLeft) / speed)
		return fmt.Sprintf(" %s out of %s, %s elapsed, %s/s, %s remaining  ",
			HumanBytes(a.current), HumanBytes(a.total),
			HumanDuration(elapsed), HumanBytes(speed),
			HumanDuration(timeLeft))
	default:
		// Done, show just total, time and speed.
		clearEOL := "\033[K"
		if a.NoAnsi {
			clearEOL = strings.Repeat(" ", 40)
		}
		return fmt.Sprintf(" %s in %v, %s/s%s",
			HumanBytes(a.current), HumanDuration(elapsed), HumanBytes(speed), clearEOL)
	}
}

// AutoProgressReader is a reader proxy associated with a progress bar.
type AutoProgressReader struct {
	AutoProgress
	r io.Reader
}

func (r *AutoProgressReader) Read(p []byte) (n int, err error) {
	n, err = r.r.Read(p)
	if n > 0 {
		r.Update(n)
	}
	return n, err
}

// End the progress bar: writes a newline and last update if it was skipped
// earlier due to rate limits. This is called automatically upon Close() by
// the Auto* wrappers.
func (bar *Bar) End() {
	bar.out.Lock()
	// Potential unwritten/skipped last update (only if ending before 100%).
	if len(bar.out.buf) > 0 {
		_, _ = bar.out.out.Write(bar.out.buf)
		bar.out.buf = bar.out.buf[:0]
	}
	_, _ = bar.out.out.Write([]byte{'\n'})
	bar.out.needErase = false
	bar.out.Unlock()
}

func (r *AutoProgressReader) Close() error {
	r.End()
	if closer, ok := r.r.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// NewAutoReader returns a new io.Reader that will update the progress bar as it reads from the underlying reader
// up to the expected total (pass a negative total for just spinner updates for unknown end/total).
func NewAutoReader(bar *Bar, r io.Reader, total int64) *AutoProgressReader {
	res := &AutoProgressReader{}
	res.Bar = bar
	res.Bar.Extra = res.Extra
	res.r = r
	res.total = total
	res.Update(0)
	return res
}

// AutoProgressWriter is a writer proxy associated with a progress bar.
type AutoProgressWriter struct {
	AutoProgress
	w io.Writer
}

func (w *AutoProgressWriter) Write(p []byte) (n int, err error) {
	n, err = w.w.Write(p)
	w.Update(n)
	return n, err
}

func (w *AutoProgressWriter) Close() error {
	w.End()
	if closer, ok := w.w.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// NewAutoWriter returns a new io.Writer that will update the progress bar as it writes from the underlying writer
// up to the expected total (pass a negative total for just spinner updates for unknown end/total).
func NewAutoWriter(bar *Bar, w io.Writer, total int64) *AutoProgressWriter {
	res := &AutoProgressWriter{}
	res.Bar = bar
	res.Bar.Extra = res.Extra
	res.w = w
	res.total = total
	res.Update(0)
	return res
}

// DefaultConfig returns a default configuration for the progress bar with default values:
// DefaultWidth, color and spinner on, no extra nor prefix, a default update interval of 100ms.
func DefaultConfig() Config {
	return Config{
		Width:          DefaultWidth,
		UseColors:      true,
		Color:          DefaultColor, // GreenBar
		Spinner:        true,
		Prefix:         "",
		UpdateInterval: DefaultMaxUpdateInterval,
		NoAnsi:         false,
	}
}

// NewBar returns a new progress bar with default settings (DefaultWidth, color and spinner on, no extra nor prefix)
// and using the shared global ScreenWriter.
// Use cfg.NewBar() to create a new progress bar with a specific/different config.
func NewBar() *Bar {
	return &Bar{
		Config: DefaultConfig(),
		Extra:  nil,
		out:    screenWriter,
	}
}

// NewBar creates a new progress bar from the config.
func (cfg Config) NewBar() *Bar {
	// Default to DefaultWidth if 0.
	if cfg.Width <= 0 {
		cfg.Width = DefaultWidth
	}
	var out *writer
	// Default to share screenwriter if nil.
	if cfg.ScreenWriter == nil {
		out = screenWriter
	} else {
		out = &writer{out: cfg.ScreenWriter, buf: make([]byte, 0, ExpectedMaxLength), noAnsi: cfg.NoAnsi}
	}
	return &Bar{
		Config: cfg,
		Extra:  nil,
		out:    out,
	}
}

// Compile check time of interface implementations.
var (
	_ io.Writer = &AutoProgressWriter{}
	_ io.Closer = &AutoProgressWriter{}
	_ io.Reader = &AutoProgressReader{}
	_ io.Closer = &AutoProgressReader{}
)

// --- Multi bar ---

type MultiBar struct {
	// Common config used to create the bars.
	Config
	// The progress bars that are part of this multi bar set.
	Bars []*Bar
}

// End should be called at the end to move the cursor to the line after the last multi bar.
func (mb *MultiBar) End() {
	lastBar := mb.Bars[len(mb.Bars)-1]
	fmt.Fprintf(lastBar.out.out, "%s\n", lastBar.indexBasedMoveDown())
}

// PrefixesAlign iterates over the prefixes to left align them such as all the bars are also aligned.
// Adds 1 space to the longest prefix and updates the prefix of each bar and refreshes them.
// (Used in multi bar during initial setup as well as when Add()ing new bars whose prefix might be longer).
func (mb *MultiBar) PrefixesAlign() {
	// find the alignment of prefixes
	maxLen := 0
	for _, b := range mb.Bars {
		// find the longest prefix before current padding:
		p := strings.TrimSpace(b.Prefix)
		if len(p) > maxLen {
			maxLen = len(p)
		}
	}
	maxLen++ // extra space before spinner
	// update the prefixes
	for _, b := range mb.Bars {
		b.out.Lock()
		s := strings.TrimSpace(b.Prefix)
		b.Prefix = s + strings.Repeat(" ", maxLen-len(s))
		b.out.Unlock()
		b.Redraw()
	}
}

// NewMultiBarPrefixes creates an array of progress bars with the same settings and
// a prefix for each and with cfg.ExtraLines in between each.
// ANSI must be supported by the terminal as this relies on moving the cursor up/down for each bar.
func (cfg Config) NewMultiBarPrefixes(prefixes ...string) *MultiBar {
	res := &MultiBar{
		Config: cfg,
	}
	if cfg.ScreenWriter == nil {
		cfg.ScreenWriter = os.Stderr // to not get the shared screenwriter, with each their own update time/counter.
	}
	bars := make([]*Bar, len(prefixes))
	for i := range prefixes {
		bars[i] = cfg.NewBar()
		bars[i].Prefix = prefixes[i]
	}
	res.Add(bars...).init()
	res.PrefixesAlign()
	return res
}

// NewMultiBar sets up a multibar from already created progress bars (for instance AutoProgressReader/Writers).
// It will reserve space for the new bars and move the cursor up/down as needed as well as align the prefixes of each bar.
func (cfg Config) NewMultiBar(mbars ...*Bar) *MultiBar {
	res := &MultiBar{
		Config: cfg,
	}
	res.Add(mbars...).init()
	return res
}

// init makes enough space on the terminal for the number of bars and their ExtraLines.
// It will also clear the screen from the cursor to the end of the screen.
func (mb *MultiBar) init() {
	mb.reservespace(true)
}

func (mb *MultiBar) reservespace(initial bool) {
	clearAfter := ClearAfter
	n := len(mb.Bars)
	if n == 0 {
		panic("No bars to multi-bar init")
	}
	if !initial {
		clearAfter = "" // no screen clear after initial setup.
		// Initially we setup including the extra lines above the (first) bar, when doing later updates / adding more bars,
		// the cursor is on that first bar (so extralines below).
		if mb.ExtraLines > 0 {
			clearAfter = fmt.Sprintf("\033[%dA", mb.ExtraLines)
		}
	}
	mul := (1 + mb.ExtraLines)
	mb.Bars[0].out.Lock()
	// Clear from cursor/line to end of screen and make space for all the bars, then back up to the first bar.
	_, _ = mb.ScreenWriter.Write([]byte("\r" + clearAfter + strings.Repeat("\n", n*mul-1) + // add xxx to newline to see
		fmt.Sprintf("\033[%dA", (n-1)*mul)))
	mb.Bars[0].out.Unlock()
}

// Add adds 1 or more progress bars to an existing multibar.
// Add should be called before the added bars are started/updated.
// It will reserve space for the new bars and move the cursor up/down as needed
// for a bar that was already created with NewMultiBar() or NewMultiBarPrefixes().
func (mb *MultiBar) Add(mbars ...*Bar) *MultiBar {
	prev := len(mb.Bars)
	mb.Bars = append(mb.Bars, mbars...)
	for i := prev; i < len(mb.Bars); i++ {
		mb.Bars[i].out.Lock()
		mb.Bars[i].index = (1 + mb.ExtraLines) * i
		mb.Bars[i].out.Unlock()
	}
	if prev > 0 {
		mb.reservespace(false)
	}
	return mb
}
