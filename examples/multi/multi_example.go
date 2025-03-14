package main

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"sync"
	"time"

	"fortio.org/progressbar"
)

func UpdateBar(b *progressbar.Bar, delay time.Duration) {
	for i := 0; i <= 300; i++ {
		b.Progress(float64(i) / 3.)
		time.Sleep(delay)
	}
}

func main() {
	noColorFlag := flag.Bool("no-color", false, "Disable color in the progress bars")
	extraLinesFlag := flag.Int("extra", 1, "Extra lines between each progress bars")
	flag.Parse()
	fmt.Println("Multi progress bar example" + progressbar.ClearAfter)
	cfg := progressbar.DefaultConfig()
	cfg.ExtraLines = *extraLinesFlag
	cfg.UseColors = !*noColorFlag
	cfg.ScreenWriter = os.Stdout
	cfg.UpdateInterval = 0 // Update immediately as we're simulating sleep and it's a demo.
	mbar := cfg.NewMultiBarPrefixes(
		"b1",
		"longest prefix",
		"short",
		"b4",
	)
	wg := sync.WaitGroup{}
	for i, bar := range mbar.Bars {
		wg.Add(1)
		// Update at random speed so bars move differently:
		delay := time.Duration(5+rand.IntN(40)) * time.Millisecond //nolint:gosec // not crypto...
		if *extraLinesFlag > 0 {
			bar.WriteAbove(fmt.Sprintf("\t\t\tBar %d delay is %v", i+1, delay))
		}
		go func(b *progressbar.Bar) {
			UpdateBar(b, delay)
			wg.Done()
		}(bar)
	}
	// Add an extra bar later example:
	time.Sleep(3 * time.Second)
	cfg.Prefix = "A wild bar appears"
	extraBar := cfg.NewBar()
	mbar.Add(extraBar)
	mbar.PrefixesAlign()
	extraBar.WriteAbove("\t\t\tExtra bar added after 3 seconds")
	UpdateBar(extraBar, 15*time.Millisecond)
	// Wait for all the other bars to finish too:
	wg.Wait()
	mbar.End()
}
