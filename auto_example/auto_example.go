// Demonstrate using the AutoReader to show a progress bar while fetching a url.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

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
	bar := progressbar.NewBarWithWriter(os.Stderr)
	bar.Prefix = "URL "
	reader := progressbar.NewAutoReader(bar, resp.Body, resp.ContentLength)
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Error fetching %s: %s\n", url, resp.Status)
		return 1
	}
	_, err = io.Copy(os.Stdout, reader)
	reader.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading/writing: %v\n", err)
	}
	return 0
}
