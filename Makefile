VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test race lint check cross bench demo film test-nvidia test-fake-nvml completions clean

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/siltide .

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
	GOOS=linux  GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/siltide-linux-amd64 .
	GOOS=linux  GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/siltide-linux-arm64 .
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/siltide-darwin-arm64 .

bench:
	go test -run xxx -bench . -benchmem ./internal/collect/ ./internal/tui/ ./internal/history/

demo: build
	bin/siltide --demo

# Runs against real NVIDIA hardware when present: `siltide --once` must list
# the devices `nvidia-smi` sees.
test-nvidia: build
	@command -v nvidia-smi >/dev/null || { echo "no nvidia-smi, skipping"; exit 0; }
	bin/siltide --once --vendors nvidia --no-history

test-fake-nvml:
	scripts/test-fake-nvml.sh

# Shell completions and the man page, as the packages ship them.
completions: build
	mkdir -p build/extra
	bin/siltide --completion bash > build/extra/siltide.bash
	bin/siltide --completion zsh > build/extra/_siltide
	bin/siltide --completion fish > build/extra/siltide.fish
	bin/siltide --man > build/extra/siltide.1

# Screenshots for README.md: the fleet set from the simulated fleet, and
# the Apple silicon set captured on the Mac that runs `make shots-mac`.
shots:
	go run ./scripts/shots -out assets
	scripts/shots/png.sh

# The animated demo README.md and the site hero embed: a scripted run over
# the simulated fleet, rasterized frame by frame and assembled into a GIF.
film:
	scripts/shots/film.sh

shots-mac:
	go run ./scripts/shots -live 90s -host m4-pro -tabs overview,devices -prefix mac- -out assets
	scripts/shots/png.sh mac-overview mac-devices

clean:
	rm -rf bin dist build
