.PHONY: all build generate test clean install-deps build-frontend build-go

# Default target
all: build

# Install frontend dependencies
install-deps:
	npm install

# Generate templ templates
generate:
	go tool templ generate

# Build frontend assets (CSS and JS)
build-frontend:
	npm run build

# Build Go binary
build-go: generate
	go build -o bin/generator ./cmd/generator

# Build everything
build: generate build-frontend build-go

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/ dist/
	find . -name "*_templ.go" -delete

# Watch mode for development
watch-frontend:
	npm run watch:css & npm run watch:js
