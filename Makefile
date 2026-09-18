VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test race lint check cross bench demo film schema shots-light test-nvidia test-fake-nvml completions clean

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

# The JSON schema for config.yaml, generated from the struct that reads it.
schema:
	go run ./scripts/schema > schemas/config.schema.json

# The animated demo README.md and the site hero embed: a scripted run over
# the simulated fleet, rasterized frame by frame and assembled into a GIF.
# The second run records the same storyboard on paper, which is what the site
# plays in light mode; a capture made for a dark terminal is hard to read on
# a white page.
film:
	scripts/shots/film.sh
	go run ./scripts/shots -film build/film-light -player assets/demo-light.json -theme paper -width 150 -height 40
	rm -rf build/film-light

# The same captures on paper, for readers whose GitHub or browser is in light
# mode. The README picks between them with <picture>.
shots-light:
	go run ./scripts/shots -out assets -theme paper -prefix light-
	scripts/shots/png.sh light-overview light-devices light-history light-links light-health

shots-mac:
	go run ./scripts/shots -live 90s -host m4-pro -tabs overview,devices -prefix mac- -out assets
	scripts/shots/png.sh mac-overview mac-devices

clean:
	rm -rf bin dist build
