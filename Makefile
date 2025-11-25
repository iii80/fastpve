GO ?= go
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
BUILD_FLAGS ?= -trimpath -a -ldflags "-s -w -extldflags '-static'"
BIN_DIR ?= bin
BINARY ?= $(BIN_DIR)/FastPVE
CMD ?= ./cmd/fast
WORKFILE ?= go.work
WORKFILE_ABS := $(abspath $(WORKFILE))

.PHONY: all build build-remote clean

all: build

# Build without HAS_REMOTE_URL; ignore go.work so the default go.mod is used.
build: $(BIN_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) GOWORK=off $(GO) build $(BUILD_FLAGS) -o $(BINARY) $(CMD)

# Build with HAS_REMOTE_URL; use go.work so the replace for remote cache is applied.
build-remote: $(BIN_DIR) $(WORKFILE)
	GOOS=$(GOOS) GOARCH=$(GOARCH) GOWORK=$(WORKFILE_ABS) $(GO) build $(BUILD_FLAGS) -tags HAS_REMOTE_URL -o $(BINARY) $(CMD)

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

clean:
	rm -f $(BINARY)
