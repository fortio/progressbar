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

func UpdateBar(b *progressbar.State, delay time.Duration) {
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
	mbar := progressbar.NewMultiBar(
		os.Stdout,
		*extraLinesFlag, // 1 extra line between bars by default.
		"b1",
		"longest prefix",
		"short",
		"b4",
	)
	wg := sync.WaitGroup{}
	for i, bar := range mbar {
		wg.Add(1)
		// Update at random speed so bars move differently:
		delay := time.Duration(5+rand.IntN(40)) * time.Millisecond //nolint:gosec // not crypto...
		if *extraLinesFlag > 0 {
			bar.WriteAbove(fmt.Sprintf("\t\t\tBar %d delay is %v", i+1, delay))
		}
		bar.UseColors = !*noColorFlag
		bar.UpdateInterval = 0 // Update immediately as we're simulating sleep and it's a demo.
		go func(b *progressbar.State) {
			UpdateBar(b, delay)
			wg.Done()
		}(bar)
	}
	wg.Wait()
	progressbar.MultiBarEnd(mbar)
}
