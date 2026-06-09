BIN_NAME    ?= git-trend
GO          ?= go
INSTALL_DIR ?= $(HOME)/.local/bin

SRC_DIR   := .
BUILD_DIR := ./bin
TMP_DIR   := ./tmp

VERSION      := $(shell git describe --abbrev=0 --tags --always)
LDFLAGS      := -X main.Version=$(VERSION)
PROD_LDFLAGS := -s -w $(LDFLAGS)
BIN          := $(BUILD_DIR)/$(BIN_NAME)

## help: print this help message
.PHONY: help
help:
	@echo "Usage:"
	@sed -n "s/^##//p" ${MAKEFILE_LIST} | column -t -s ":" | sed -e "s/^/ /"

## audit: run quality control checks
.PHONY: audit
audit: test
	$(GO) mod tidy -diff
	$(GO) mod verify
	@test -z "$$(gofmt -l .)"
	$(GO) vet ./...

## test: run all tests
.PHONY: test
test:
	$(GO) test -v -race -buildvcs ./...

## test/cover: run tests and open the HTML coverage report
.PHONY: test/cover
test/cover:
	@mkdir -p $(TMP_DIR)
	$(GO) test -v -race -buildvcs -coverprofile=$(TMP_DIR)/coverage.out ./...
	$(GO) tool cover -html=$(TMP_DIR)/coverage.out

## tidy: tidy modfiles and modernize and format .go files
.PHONY: tidy
tidy:
	$(GO) mod tidy -v
	$(GO) fix ./...
	$(GO) fmt ./...

## build: build the application
.PHONY: build
build:
	@$(GO) build -v -ldflags "$(LDFLAGS)" -o=$(BIN) $(SRC_DIR)

## build/prod: build an optimized, stripped, static production binary
.PHONY: build/prod
build/prod:
	@CGO_ENABLED=0 $(GO) build -trimpath -ldflags "$(PROD_LDFLAGS)" -o=$(BIN) $(SRC_DIR)

## run: run the bin
.PHONY: run
run: build
	@$(BIN)

## install: build and install to INSTALL_DIR (default ~/.local/bin)
.PHONY: install
install: build/prod
	@mkdir -p $(INSTALL_DIR)
	@cp $(BIN) $(INSTALL_DIR)/$(BIN_NAME)
	@echo "Installed $(BIN_NAME) to $(INSTALL_DIR)/$(BIN_NAME)"

## uninstall: remove the installed binary from INSTALL_DIR
.PHONY: uninstall
uninstall:
	@rm -f $(INSTALL_DIR)/$(BIN_NAME)
	@echo "Removed $(INSTALL_DIR)/$(BIN_NAME)"

## clean: remove build artifacts
.PHONY: clean
clean:
	rm -rvI $(BUILD_DIR) $(TMP_DIR)

# vim: set tabstop=4 shiftwidth=4 noexpandtab
