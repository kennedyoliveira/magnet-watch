GO ?= go

.PHONY: build
build:
	$(GO) build -v -o dist/magnet-watch ./main.go

.PHONY: clean
clean:
	$(GO) clean
	rm -rf dist

.PHONY: download-dependencies
download-dependencies:
	echo "Downloading dependencies"
	$(GO) mod download .

.PHONY: cleanup-dependencies
cleanup-dependencies:
	echo "Cleaning up dependencies"
	$(GO) mod tidy .