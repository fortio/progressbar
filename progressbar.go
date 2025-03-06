// Simple zero dependency cross platform (only need ANSI compatible terminal) progress bar
// for your golang terminal / command line interface (CLI) applications.
package progressbar

import (
	"fmt"
	"strings"
)

const (
	DefaultWidth = 40
	Space        = " "
	Full         = "█"
	// Green FG, Grey BG
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
// Use MoveCursorUp to move up to update other lines as needed.
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
	fmt.Printf("\r%s %.1f%%", bar, progressPercent)
}

// Move the cursor up n lines and clears that line.
func MoveCursorUp(n int) {
	// ANSI escape codes used:
	// xA = move up x lines
	// 2K = clear entire line
	// G = move to the beginning of the line
	fmt.Printf("\033[%dA\033[2K\033[G", n)
}
