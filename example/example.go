package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"fortio.org/progressbar"
)

func PrintStuff(w io.Writer, every time.Duration) {
	ticker := time.NewTicker(every)
	i := 0
	for range ticker.C {
		i++
		nowStr := time.Now().Format("15:04:05")
		fmt.Fprintf(w, "[%d] Just an extra demo print every %v (%s)\n", i, every, nowStr)
	}
}

func main() {
	colorFlag := flag.Bool("color", false, "Use color in the progress bar")
	delayFlag := flag.Duration("delay", 50*time.Millisecond, "Delay between progress bar updates")
	everyFlag := flag.Duration("every", 1*time.Second, "Print extra stuff every")
	flag.Parse()
	pb := progressbar.Config{Width: progressbar.DefaultWidth, UseColors: *colorFlag, Spinner: true}
	w := progressbar.ScreenWriter(os.Stdout)
	fmt.Fprintln(w, "Progress bar example")
	// demonstrate concurrency safety:
	go PrintStuff(w, *everyFlag)
	// exact number of 'pixels', just to demo every smooth step:
	n := pb.Width * 8
	for i := 0; i <= n; i++ {
		pb.ProgressBar(100. * float64(i) / float64(n))
		time.Sleep(*delayFlag)
	}
	fmt.Println() // When done, print a newline as the progress bar by default stays on same line.
}
