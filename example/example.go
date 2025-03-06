package main

import (
	"flag"
	"fmt"
	"time"

	"fortio.org/progressbar"
)

func main() {
	colorFlag := flag.Bool("color", false, "Use color in the progress bar")
	flag.Parse()
	pb := progressbar.Config{Width: progressbar.DefaultWidth, UseColors: *colorFlag}
	fmt.Print("Progress bar example\n\n")
	n := pb.Width * 8 // just to demo every smooth step
	for i := 0; i <= n; i++ {
		pb.ProgressBar(100. * float64(i) / float64(n))
		if i%63 == 0 {
			progressbar.MoveCursorUp(1)
			fmt.Printf("Just an extra demo print for %d\n", i)
		}
		time.Sleep(20 * time.Millisecond)
	}
	fmt.Println()
}
