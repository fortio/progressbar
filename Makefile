DEMO_URL ?= https://go.dev/dl/go1.24.1.src.tar.gz

.PHONY: demo demo_auto demo_simple demo_no_ansi demo_moveup demo_multi lint

demo: demo_multi demo_simple demo_moveup demo_auto demo_no_ansi

demo_simple:
	go run -race ./examples/simple -color

demo_moveup:
	go run -race ./examples/simple -moveup # no color but ansi codes.

demo_auto:
	go run -race ./examples/auto $(DEMO_URL) | wc -c

demo_no_ansi:
	go run -race ./examples/auto -no-ansi $(DEMO_URL)  | wc -c

demo_multi:
	go run -race ./examples/multi

lint: .golangci.yml
	golangci-lint run

.golangci.yml: Makefile
	curl -fsS -o .golangci.yml https://raw.githubusercontent.com/fortio/workflows/main/golangci.yml

.PHONY: lint
