.PHONY: all build generate test clean install-deps build-frontend build-go generate-html html-build

# Default target
all: build

# Install frontend dependencies
install-deps:
	npm install

# Generate templ templates
generate:
	go tool templ generate

# Build Go binary (depends on generate for templ files)
build-go: generate
	mkdir -p bin
	go build -o bin/generator ./cmd/generator

# Build frontend assets (CSS and JS) - runs after build-go for consistent ordering
build-frontend: build-go
	npm run build

# Generate HTML pages from content (requires build-go and build-frontend)
generate-html: build-frontend
	./bin/generator -content content -dist dist

# Full build orchestration: templ generate → Go build → frontend assets → HTML + RSS generation
# Single entry point that runs the complete pipeline without duplication
html-build: generate-html
	@echo "Full build completed successfully"
	@echo "  - templ templates generated"
	@echo "  - Go generator binary built"
	@echo "  - UnoCSS styles extracted to dist/styles.css"
	@echo "  - esbuild bundled components to dist/components.js"
	@echo "  - HTML pages generated in dist/"
	@echo "  - RSS feed generated (dist/feed.xml)"

# Build everything (alias for html-build)
build: html-build

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
