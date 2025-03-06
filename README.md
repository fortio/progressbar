# progressbar
[![Go Report Card](https://goreportcard.com/badge/fortio.org/progressbar)](https://goreportcard.com/report/fortio.org/progressbar)
[![GoDoc](https://godoc.org/fortio.org/progressbar?status.svg)](https://pkg.go.dev/fortio.org/progressbar)
[![Maintainability](https://api.codeclimate.com/v1/badges/bf83c496d49b169cd744/maintainability)](https://codeclimate.com/github/fortio/progressbar/maintainability)
[![CI Checks](https://github.com/fortio/progressbar/actions/workflows/include.yml/badge.svg)](https://github.com/fortio/progressbar/actions/workflows/include.yml)


Zero dependency cross platform (just needs basic ANSI codes and Unicode font support) golang progress bar for terminal/CLIs


## Example
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

```
go run fortio.org/progressbar/example@latest -color
```

Produces

![Example Screenshot](example.png)

Or without color:
```
◅███████████████████████████▊            ▻ 69.4%
```
