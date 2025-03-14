// Demonstrate using the AutoReader to show a progress bar while fetching a url.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"fortio.org/progressbar"
)

func usage() int {
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "Usage: %s [flags] url > file\n", os.Args[0])
	return 1
}

func main() {
	os.Exit(Main())
}

func Main() int {
	noAnsiFlag := flag.Bool("no-ansi", false, "Disable ANSI escape codes (colors and cursor movement)")
	delayFlag := flag.Duration("delay", 5*time.Millisecond, "Artificially slowdown writes with this delay")
	flag.Parse()
	if flag.NArg() != 1 {
		return usage()
	}
	url := flag.Arg(0)
	fmt.Fprintf(os.Stderr, "Fetching %s\n", url)
	cli := http.Client{}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		return 1
	}
	req.Header.Set("Accept-Encoding", "identity")
	resp, err := cli.Do(req) //nolint:bodyclose // closed by progressbar reader.Close() below.
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching %s: %v\n", url, err)
		return 1
	}
	cfg := progressbar.DefaultConfig()
	cfg.NoAnsi = *noAnsiFlag
	cfg.Prefix = "R "
	cfg.ScreenWriter = os.Stderr
	rBar := cfg.NewBar()
	// On purpose different buffer size than the writer to show the effect of different speeds.
	bufSize := 10 * 1024 * 1024 // 10MB
	reader := progressbar.NewAutoReader(rBar, resp.Body, resp.ContentLength)
	defer reader.Close()
	cfg.Prefix = "W "
	wBar := cfg.NewBar()
	writer := progressbar.NewAutoWriter(wBar, bufio.NewWriterSize(os.Stdout, 16*1024), resp.ContentLength)
	cfg.NewMultiBar(rBar, wBar)
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Error fetching %s: %s\n", url, resp.Status)
		return 1
	}
	err = AsyncCopy(writer, reader, bufSize, 100, *delayFlag)
	writer.End()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing: %v\n", err)
	}
	return 0
}

// AsyncCopy is a fairly overly complicated replacement for io.Copy that decouples
// reader and writer and optionally delays the writer.
func AsyncCopy(dst io.Writer, src io.Reader, bufSize, chanSize int, delay time.Duration) error {
	chunks := make(chan []byte, chanSize) // Channel for chunks of data
	errCh := make(chan error, 1)          // Channel for errors
	var wg sync.WaitGroup
	// Reader Goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(chunks) // Close channel when done reading
		buf := make([]byte, bufSize)
		for {
			n, err := src.Read(buf)
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])
				chunks <- data // Send chunk to channel
			}
			if err != nil {
				if err != io.EOF {
					errCh <- err
				}
				break
			}
		}
	}()
	// Writer
	for chunk := range chunks {
		if delay > 0 {
			time.Sleep(delay) // Simulate slow writing
		}
		if _, err := dst.Write(chunk); err != nil {
			return err
		}
	}
	// Ensure the reader goroutine finishes
	wg.Wait()
	// Return any error from reading
	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}
