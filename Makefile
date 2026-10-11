# FynChat desktop — build and run helpers.
#
#   make            build the binary
#   make run        build and launch the app
#   make test       run the HTTP client tests
#   make check      fmt + vet + test, what CI should run
#
BINARY := fynchat
GO     ?= go
PKG    := .

.PHONY: all build run test race cover fmt fmt-check vet tidy deps clean cross-windows cross-darwin shots help

all: build

## build: compile the desktop binary
build:
	$(GO) build -o $(BINARY) $(PKG)

## run: build and start the GUI
run: build
	./$(BINARY)

## test: run the test suite
test:
	$(GO) test ./...

## race: run the test suite with the race detector
race:
	$(GO) test -race ./...

## cover: run tests and report coverage per function
cover:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out | tail -1

## fmt: rewrite files with gofmt
fmt:
	gofmt -w .

## fmt-check: fail if anything is unformatted (no changes made)
fmt-check:
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "unformatted files:"; echo "$$out"; exit 1; fi

## vet: run go vet
vet:
	$(GO) vet ./...

## tidy: sync go.mod and go.sum
tidy:
	$(GO) mod tidy

## check: everything CI should verify
check: fmt-check vet test

## deps: system packages Fyne needs to compile (Debian/Ubuntu)
deps:
	sudo apt install build-essential pkg-config libgl1-mesa-dev libwayland-dev libxkbcommon-dev xorg-dev

## cross-windows: cross compile for Windows (needs fyne-cross)
cross-windows:
	fyne-cross $(BINARY).exe

## cross-darwin: cross compile for macOS (needs fyne-cross)
cross-darwin:
	fyne-cross -os darwin $(BINARY).app

## shots: render each view to a PNG for review
# Fyne's test driver runs fyne.Do inline, so the loaders shape text while the
# capture thread renders, and the shared font cache can be corrupted. It is a
# test-driver artefact — the app renders on one thread — so this retries.
shots:
	@for i in 1 2 3 4 5; do \
	  $(GO) test -tags screenshot -run TestRenderViews -count=1 ./... && break; \
	  echo "retry $$i/5"; sleep 1; \
	done; true

## clean: remove build output
clean:
	rm -f $(BINARY) coverage.out
	rm -rf fyne-cross/tmp

## help: list the targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | awk -F': ' '{printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'