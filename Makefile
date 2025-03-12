DEMO_URL ?= https://go.dev/dl/go1.24.1.src.tar.gz

.PHONY: demo demo_auto demo_simple

demo: demo_simple demo_auto

demo_simple:
	go run ./example -color

demo_auto:
	go run ./auto_example $(DEMO_URL) | wc -c

lint: .golangci.yml
	golangci-lint run

.golangci.yml: Makefile
	curl -fsS -o .golangci.yml https://raw.githubusercontent.com/fortio/workflows/main/golangci.yml

.PHONY: lint
