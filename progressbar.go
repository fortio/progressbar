// Simple zero dependency cross platform (only need ANSI compatible terminal) progress bar
// for your golang terminal / command line interface (CLI) applications.
package progressbar

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

const (
	DefaultWidth = 40
	Space        = " "
	Full         = "█"
	// Green FG, Grey BG.
	Color = "\033[32;47m"
	Reset = "\033[0m"
)

// 1/8th of a full block to 7/8ths of a full block (ie fractional part of a block to
// get 8x resolution per character).
var FractionalBlocks = [...]string{"", "▏", "▎", "▍", "▌", "▋", "▊", "▉"}

type Config struct {
	Width     int
	UseColors bool
}

// Show a progress bar percentage (0-100%). On the same line as current line,
// so when call repeatedly it will overwrite/update itself.
// Use MoveCursorUp to move up to update other lines as needed or use Writer()
// to write output without mixing with a progress bar.
// This is thread safe / acquires a shared lock to avoid issues on the output.
// Of note it will work best if every output to the Writer() ends with a \n.
func (cfg *Config) ProgressBar(progressPercent float64) {
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
	bar := color + strings.Repeat(Full, fullCount) + FractionalBlocks[remainder] + strings.Repeat(Space, spaceCount) + reset
	w.Lock()
	fmt.Printf("\r%s %.1f%%", bar, progressPercent)
	w.Unlock()
}

// Move the cursor up n lines and clears that line.
func MoveCursorUp(n int) {
	// ANSI escape codes used:
	// nA   = move up n lines
	// \r   = beginning of the line
	// (0)K = erase from current position to end of line
	fmt.Printf("\033[%dA\r\033[K", n)
}

type writer struct {
	sync.Mutex
	out io.Writer
}

func (w *writer) Write(buf []byte) (n int, err error) {
	w.Lock()
	defer w.Unlock()
	_, _ = w.out.Write([]byte("\r\033[K")) // erase current line
	n, err = w.out.Write(buf)
	return
}

var w = writer{out: os.Stdout}

// Writer returns an io.Writer which is safe to use concurrently with a progress bar.
// Any writes will clear the current line/progress bar and write the new content, and
// then rewrite the progress bar. Pass in os.Stdout or os.Stderr or any other io.Writer
// (that ends up outputting to ANSI aware terminal) to use this with your existing code.
func Writer(out io.Writer) io.Writer {
	w.Lock()
	w.out = out
	w.Unlock()
	return &w
}
