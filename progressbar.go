// Simple zero dependency cross platform (only need ANSI compatible terminal) progress bar
// for your golang terminal / command line interface (CLI) applications.
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
	Color       = "\033[32;47m"
	Reset       = "\033[0m"
	DoneSpinner = "✓ "
)

var (
	// 1/8th of a full block to 7/8ths of a full block (ie fractional part of a block to
	// get 8x resolution per character).
	FractionalBlocks = [...]string{"", "▏", "▎", "▍", "▌", "▋", "▊", "▉"}
	SpinnerChars     = [...]string{"⣾ ", "⣽ ", "⣻ ", "⢿ ", "⡿ ", "⣟ ", "⣯ ", "⣷ "}
)

type Config struct {
	// Width of the progress bar in characters (0 will use DefaultWidth).
	Width int
	// UseColors to use colors in the progress bar.
	UseColors bool
	// Spinner to also show a spinner in front of the progress bar.
	Spinner bool
	// Extra string to show after the progress bar. Keep nil for no extra.
	Extra func(progressPercent float64) string
}

// Show a progress bar percentage (0-100%). On the same line as current line,
// so when call repeatedly it will overwrite/update itself.
// Use MoveCursorUp to move up to update other lines as needed or use Writer()
// to write output without mixing with a progress bar.
// This is thread safe / acquires a shared lock to avoid issues on the output.
// Of note it will work best if every output to the Writer() ends with a \n.
func (cfg *Config) ProgressBar(progressPercent float64) {
	bar := ""
	progressPercentStr := ""
	if progressPercent >= 0 && progressPercent <= 100 {
		width := float64(cfg.Width)
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
		color := Color
		reset := Reset
		if !cfg.UseColors {
			color = "◅" // "◢"
			reset = "▻" // "◣"
		}
		bar = color + strings.Repeat(Full, fullCount) + FractionalBlocks[remainder] + strings.Repeat(Space, spaceCount) + reset
		progressPercentStr = fmt.Sprintf(" %.1f%%", progressPercent)
	}
	spinner := ""
	screenWriter.Lock()
	if cfg.Spinner {
		spinner = SpinnerChars[screenWriter.count]
		screenWriter.count = (screenWriter.count + 1) % len(SpinnerChars)
		if IsDone(progressPercent) {
			spinner = DoneSpinner
		}
	}
	extra := ""
	if cfg.Extra != nil {
		extra = cfg.Extra(progressPercent)
	}
	screenWriter.buf = append(screenWriter.buf, "\r"...)
	screenWriter.buf = append(screenWriter.buf, spinner...)
	screenWriter.buf = append(screenWriter.buf, bar...)
	screenWriter.buf = append(screenWriter.buf, progressPercentStr...)
	screenWriter.buf = append(screenWriter.buf, extra...)
	// screenWriter.buf = append(screenWriter.buf, '\n') // Uncomment to debug/see all the incremental updates.
	_, _ = screenWriter.out.Write(screenWriter.buf)
	screenWriter.buf = screenWriter.buf[:0]
	screenWriter.needErase = true
	screenWriter.Unlock()
}

// Approximate check if the progress is done (percent > 99.99).
func IsDone(percent float64) bool {
	return percent > 99.999
}

// Standalone spinner when the total or progress toward 100% isn't known.
func Spinner() {
	screenWriter.Lock()
	fmt.Fprintf(screenWriter.out, "\r%s", SpinnerChars[screenWriter.count])
	screenWriter.count = (screenWriter.count + 1) % len(SpinnerChars)
	screenWriter.Unlock()
}

// Move the cursor up n lines and clears that line.
func MoveCursorUp(n int) {
	// ANSI escape codes used:
	// nA   = move up n lines
	// \r   = beginning of the line
	// (0)K = erase from current position to end of line
	fmt.Fprintf(screenWriter.out, "\033[%dA\r\033[K", n)
}

type writer struct {
	sync.Mutex
	out       io.Writer
	buf       []byte
	count     int
	needErase bool
}

func (w *writer) Write(buf []byte) (n int, err error) {
	w.Lock()
	if w.needErase {
		_, _ = w.out.Write([]byte("\r\033[K")) // erase current progress bar line
		w.needErase = false
	}
	n, err = w.out.Write(buf)
	w.Unlock()
	return
}

// Global write with lock and reused buffer.
// Outside of testing there is generally only 1 valid output for ansi progress bar:
// os.Stdout or os.Stderr.
var screenWriter = writer{out: os.Stderr, buf: make([]byte, 0, 256)}

// ScreenWriter returns an io.ScreenWriter which is safe to use concurrently with a progress bar.
// Any writes will clear the current line/progress bar and write the new content, and
// then rewrite the progress bar. Pass in os.Stdout or os.Stderr or any other io.ScreenWriter
// (that ends up outputting to ANSI aware terminal) to use this with your existing code.
func ScreenWriter(out io.Writer) io.Writer {
	screenWriter.Lock()
	screenWriter.out = out
	screenWriter.Unlock()
	return &screenWriter
}

// Can be used for speed (append "/s") or raw bytes.
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

func HumanDuration(d time.Duration) string {
	if d <= time.Second {
		return d.Round(time.Millisecond).String()
	}
	if d < time.Hour {
		return d.Round(100 * time.Millisecond).String()
	}
	return d.Round(time.Minute).String()
}

type AutoProgress struct {
	Config
	total   int64
	current int64
	start   time.Time
}

func (a *AutoProgress) Update(n int) {
	if a.current == 0 {
		a.start = time.Now()
	}
	a.current += int64(n)
	if a.current > 0 || a.total > 0 {
		a.ProgressBar(float64(a.current) * 100. / float64(a.total))
	}
}

func (a *AutoProgress) Extra(progressPercent float64) string {
	elapsed := time.Since(a.start)
	if a.current == 0 {
		return fmt.Sprintf(" %d/%d", a.current, a.total)
	}
	speed := float64(a.current) / elapsed.Seconds()
	switch {
	case a.total <= 0:
		// No total, show current, elapsed and speed.
		return fmt.Sprintf(" %s, %v elapsed, %s/s",
			HumanBytes(a.current), elapsed.Round(time.Millisecond), HumanBytes(speed))
	case !IsDone(progressPercent):
		bytesLeft := a.total - a.current
		timeLeft := time.Duration(float64(time.Second) * float64(bytesLeft) / speed)
		return fmt.Sprintf(" %s out of %s, %s elapsed, %s/s, %s remaining",
			HumanBytes(a.current), HumanBytes(a.total),
			HumanDuration(elapsed), HumanBytes(speed),
			HumanDuration(timeLeft))
	default:
		// Done, show just total, time and speed.
		return fmt.Sprintf(" %s in %v, %s/s\033[K",
			HumanBytes(a.current), HumanDuration(elapsed), HumanBytes(speed))
	}
}

// A reader proxy associated with a progress bar.
type AutoProgressReader struct {
	AutoProgress
	r io.Reader
}

func (r *AutoProgressReader) Read(p []byte) (n int, err error) {
	n, err = r.r.Read(p)
	if n > 0 {
		r.Update(n)
	}
	return
}

func (r *AutoProgressReader) Close() error {
	_, _ = screenWriter.out.Write([]byte("\n"))
	if closer, ok := r.r.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// NewAutoReader returns a new io.Reader that will show a progress bar as it reads from the underlying reader
// up to the expected total. You can adjust the Config of the resulting object.
func NewAutoReader(r io.Reader, total int64) *AutoProgressReader {
	res := &AutoProgressReader{}
	res.Config = DefaultConfig()
	res.Config.Extra = res.Extra
	res.r = r
	res.total = total
	res.Update(0)
	return res
}

// A writer proxy associated with a progress bar.
type AutoProgressWriter struct {
	AutoProgress
	w io.Writer
}

func (w *AutoProgressWriter) Write(p []byte) (n int, err error) {
	n, err = w.w.Write(p)
	w.Update(n)
	return
}

func (w *AutoProgressWriter) Close() error {
	_, _ = screenWriter.out.Write([]byte("\n"))
	if closer, ok := w.w.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// NewAutoWriter returns a new io.Writer that will show a progress bar as it writes to the underlying writer
// up to the expected total. You can adjust the Config of the resulting object.
func NewAutoWriter(w io.Writer, total int64) *AutoProgressWriter {
	res := &AutoProgressWriter{}
	res.Config = DefaultConfig()
	res.Config.Extra = res.Extra
	res.w = w
	res.total = total
	res.Update(0)
	return res
}

// DefaultConfig returns a default Config with DefaultWidth, colors, spinner and no extra string.
func DefaultConfig() Config {
	return Config{DefaultWidth, true, true, nil}
}
