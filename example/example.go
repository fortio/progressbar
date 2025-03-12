package main

import (
	"flag"
	"fmt"
	"io"
	"time"

	"fortio.org/progressbar"
)

func PrintStuff(pb *progressbar.State, w io.Writer, every time.Duration) {
	ticker := time.NewTicker(every)
	i := 0
	for {
		i++
		nowStr := time.Now().Format("15:04:05 ")
		fmt.Fprintf(w, "[%d] Just an extra demo print every %v: %s\n", i, every, nowStr)
		pb.UpdatePrefix(nowStr)
		<-ticker.C
	}
}

func main() {
	colorFlag := flag.Bool("color", false, "Use color in the progress bar")
	delayFlag := flag.Duration("delay", 50*time.Millisecond, "Delay between progress bar updates")
	everyFlag := flag.Duration("every", 1*time.Second, "Print extra stuff every")
	flag.Parse()
	pb := progressbar.NewBar()
	pb.UseColors = *colorFlag
	w := pb.Writer()
	fmt.Fprintln(w, "Progress bar example")
	// demonstrate concurrency safety:
	go PrintStuff(pb, w, *everyFlag)
	// exact number of 'pixels', just to demo every smooth step:
	n := pb.Width * 8
	for i := 0; i <= n; i++ {
		pb.Progress(100. * float64(i) / float64(n))
		time.Sleep(*delayFlag)
	}
	fmt.Println() // When done, print a newline as the progress bar by default stays on same line.
}
