demo:
	go run ./example -color
	go run ./auto_example https://go.dev/dl/go1.24.1.src.tar.gz > /dev/null

lint: .golangci.yml
	golangci-lint run

.golangci.yml: Makefile
	curl -fsS -o .golangci.yml https://raw.githubusercontent.com/fortio/workflows/main/golangci.yml

.PHONY: lint
