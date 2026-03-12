# Build configuration for speed and size
GO_BUILD=go build -ldflags="-s -w" -trimpath

all: build

build:
	@echo "Building optimized api binary..."
	@$(GO_BUILD) -o api cmd/api/main.go

clean:
	@echo "Cleaning up..."
	@rm -f api
	@rm -rf tmp/

optimize-git:
	@echo "Optimizing git repository size..."
	@git gc --aggressive --prune=now

.PHONY: all build clean optimize-git