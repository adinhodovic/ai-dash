AI_DASHBOARD := $(shell git describe --tags --always --dirty 2>/dev/null || printf 'dev')
LDFLAGS += -X "main.buildTimestamp=$(shell date -u '+%Y-%m-%d %H:%M:%S')"
LDFLAGS += -X "main.aiDashVersion=$(AI_DASHBOARD)"
LDFLAGS += -X "main.goVersion=$(shell go version | sed -E 's/go version go([^ ]+) .*/\1/')"

GO := GO111MODULE=on CGO_ENABLED=0 go

.PHONY: tidy
tidy:
	$(GO) mod tidy

.PHONY: fmt
fmt:
	$(GO) fmt ./...

.PHONY: build
build:
	$(GO) build -ldflags '$(LDFLAGS)' -o ai-dash ./cmd/ai-dash

.PHONY: build-all
build-all:
	@echo "Building linux/amd64..."
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags '$(LDFLAGS)' -o ai-dash-linux-amd64 ./cmd/ai-dash
	@echo "Building linux/arm64..."
	GOOS=linux GOARCH=arm64 $(GO) build -ldflags '$(LDFLAGS)' -o ai-dash-linux-arm64 ./cmd/ai-dash

.PHONY: test
test:
	$(GO) test ./...
