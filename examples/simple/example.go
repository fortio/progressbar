package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"fortio.org/progressbar"
)

func PrintStuff(pb *progressbar.Bar, w io.Writer, every time.Duration) {
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
	noAnsiFlag := flag.Bool("no-ansi", false, "Disable ANSI escape codes (colors and cursor movement)")
	moveUpFlag := flag.Bool("moveup", false, "Demo in place move instead of writer")
	flag.Parse()
	pb := progressbar.NewBarWithWriter(os.Stdout) // For playground, defaults to stderr otherwise.
	pb.UseColors = *colorFlag
	pb.NoAnsi = *noAnsiFlag
	w := pb.Writer()
	fmt.Fprintln(w, "Single progress bar example")
	moveUpMode := *moveUpFlag
	if moveUpMode {
		fmt.Fprintln(w, "This line for space to demo MoveCursorUp mode")
	} else {
		// demonstrate concurrency safety:
		go PrintStuff(pb, w, *everyFlag)
	}
	// exact number of 'pixels', just to demo every smooth step:
	n := pb.Width * 8
	for i := 0; i <= n; i++ {
		pb.Progress(100. * float64(i) / float64(n))
		if moveUpMode && i%63 == 0 {
			pb.MoveCursorUp(1)
			fmt.Printf("Just an extra demo print for %d\n", i)
		}
		time.Sleep(*delayFlag)
	}
	pb.End() // When done, prints a newline as the progress bar otherwise updates on same line.
}
