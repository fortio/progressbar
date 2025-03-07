# progressbar
[![Go Report Card](https://goreportcard.com/badge/fortio.org/progressbar)](https://goreportcard.com/report/fortio.org/progressbar)
[![GoDoc](https://godoc.org/fortio.org/progressbar?status.svg)](https://pkg.go.dev/fortio.org/progressbar)
[![Maintainability](https://api.codeclimate.com/v1/badges/bf83c496d49b169cd744/maintainability)](https://codeclimate.com/github/fortio/progressbar/maintainability)
[![CI Checks](https://github.com/fortio/progressbar/actions/workflows/include.yml/badge.svg)](https://github.com/fortio/progressbar/actions/workflows/include.yml)


Zero dependency cross platform (just needs basic ANSI codes and Unicode font support) golang concurrent safe progress bar for terminal/CLIs, with 8x the resolution of others (8 steps per character).


## Example

Manually handling a 2 lines output updates (1 misc line and the 1 line for the progress bar)
```go
	pb := progressbar.Config{Width: 60, UseColors: true}
	fmt.Print("Progress bar example\n\n") // 1 empty line before the progress bar, for the demo
	n := 1000
	for i := 0; i <= n; i++ {
		pb.ProgressBar(100. * float64(i) / float64(n))
		if i%63 == 0 {
			progressbar.MoveCursorUp(1)
			fmt.Printf("Just an extra demo print for %d\n", i)
		}
		time.Sleep(20 * time.Millisecond)
	}
```
Or using the concurrent safe writer:
```go
	pb := progressbar.Config{Width: 60, UseColors: true}
	w := progressbar.Writer(os.Stdout)
	fmt.Fprintln(w, "Progress bar example")
	// demonstrate concurrency safety:
	go PrintStuff(w, *everyFlag)
	// exact number of 'pixels', just to demo every smooth step:
	n := pb.Width * 8
	for i := 0; i <= n; i++ {
		pb.ProgressBar(100. * float64(i) / float64(n))
		time.Sleep(*delayFlag)
	}
```

```
go run fortio.org/progressbar/example@latest -color
```

Produces

![Example Screenshot](example.png)

Or without color:
```
◅███████████████████████████▊            ▻ 69.4%
```

## See also

If you have more advanced needs for TUI including raw mode input or readline, you can also see/use/have a look at

[github.com/fortio/terminal](https://github.com/fortio/terminal#terminal)

And still use this for a progress bar part.
