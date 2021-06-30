GO ?= go

.PHONY: build
build:
	$(GO) build -v -o dist/magnet-watch ./main.go

.PHONY: clean
clean:
	$(GO) clean
	rm -rf dist