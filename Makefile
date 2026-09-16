VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test race lint check cross bench demo test-nvidia test-fake-nvml completions clean

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/accel .

test:
	go test ./...

race:
	go test -race ./...

lint:
	test -z "$$(gofmt -l .)"
	go vet ./...
	GOOS=linux go vet ./...
	golangci-lint run ./...
	GOOS=linux golangci-lint run ./...

check: lint test cross

cross:
	GOOS=linux  GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/accel-linux-amd64 .
	GOOS=linux  GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/accel-linux-arm64 .
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/accel-darwin-arm64 .

bench:
	go test -run xxx -bench . -benchmem ./internal/collect/ ./internal/tui/ ./internal/history/

demo: build
	bin/accel --demo

# Runs against real NVIDIA hardware when present: `accel --once` must list
# the devices `nvidia-smi` sees.
test-nvidia: build
	@command -v nvidia-smi >/dev/null || { echo "no nvidia-smi, skipping"; exit 0; }
	bin/accel --once --vendors nvidia --no-history

test-fake-nvml:
	scripts/test-fake-nvml.sh

# Shell completions and the man page, as the packages ship them.
completions: build
	mkdir -p build/extra
	bin/accel --completion bash > build/extra/accel.bash
	bin/accel --completion zsh > build/extra/_accel
	bin/accel --completion fish > build/extra/accel.fish
	bin/accel --man > build/extra/accel.1

clean:
	rm -rf bin dist build
