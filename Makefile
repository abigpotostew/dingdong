.PHONY: build build-tracker clean run

# Build the tracker.min.js and then the Go binary
build: build-tracker
	go build -o dingdong .

# Minify the tracker JavaScript
build-tracker:
	@chmod +x scripts/build-tracker.sh
	@./scripts/build-tracker.sh

# Clean build artifacts
clean:
	rm -f dingdong
	rm -f internal/handlers/static/tracker.min.js

# Run the server
run: build
	./dingdong serve --http=0.0.0.0:8090

# Build for production (smaller binary)
build-prod: build-tracker
	go build -ldflags="-s -w" -o dingdong .
