BINARY := pixel
BIN_DIR := $(HOME)/.local/bin

DIST_DIR := dist
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64

.PHONY: all build web-build install dist clean

all: install

web-build:
	cd web && bun install && bun run build

build: web-build
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) .

install: build
	mkdir -p $(BIN_DIR)
	rm -f $(BIN_DIR)/$(BINARY)
	cp bin/$(BINARY) $(BIN_DIR)/$(BINARY)

# Cross-compile release archives for every entry in PLATFORMS into dist/,
# one tar.gz + checksums.txt. The web UI is embedded via go:embed and is
# platform-independent, so it only needs building once (via the web-build
# dependency), not per target.
dist: web-build
	rm -rf $(DIST_DIR)
	mkdir -p $(DIST_DIR)
	for platform in $(PLATFORMS); do \
		os=$${platform%/*}; arch=$${platform#*/}; \
		name=$(BINARY)_$${os}_$${arch}; \
		outdir=$(DIST_DIR)/$$name; \
		mkdir -p $$outdir; \
		echo "==> building $$name"; \
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $$outdir/$(BINARY) . || exit 1; \
		cp README.md LICENSE $$outdir/ 2>/dev/null || cp README.md $$outdir/; \
		tar -C $(DIST_DIR) -czf $(DIST_DIR)/$$name.tar.gz $$name; \
		rm -rf $$outdir; \
	done
	cd $(DIST_DIR) && ( sha256sum *.tar.gz > checksums.txt 2>/dev/null || shasum -a 256 *.tar.gz > checksums.txt )

clean:
	rm -rf bin $(DIST_DIR)
