BINARY := bin/tether
MODULE := ./cmd/tether
GO     ?= go

.DEFAULT_GOAL := build

.PHONY: build run dev test vet tidy install clean help

## Build the tether binary into bin/
build:
	@mkdir -p $(dir $(BINARY))
	$(GO) build -o $(BINARY) $(MODULE)

## Build then run (pass args via ARGS=…)
run: build
	./$(BINARY) $(ARGS)

## Run without installing a binary
dev:
	$(GO) run $(MODULE) $(ARGS)

## Run all tests
test:
	$(GO) test ./...

## Static checks
vet:
	$(GO) vet ./...

## Sync go.mod / go.sum
tidy:
	$(GO) mod tidy

## Install into GOPATH/bin (or GOBIN)
install:
	$(GO) install $(MODULE)

## Remove build artifacts
clean:
	rm -rf bin/

help:
	@printf '%s\n' \
		'Usage: make [target] [ARGS="…"]' \
		'' \
		'  build    Build $(BINARY) (default)' \
		'  run      Build and run  (e.g. make run ARGS="search frontend")' \
		'  dev      go run        (e.g. make dev ARGS="ls")' \
		'  test     go test ./...' \
		'  vet      go vet ./...' \
		'  tidy     go mod tidy' \
		'  install  go install $(MODULE)' \
		'  clean    Remove bin/'
