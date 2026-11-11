VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X github.com/skryfon-org/goclean/cmd.AppVersion=$(VERSION)

.PHONY: build install test smoke vet lint snapshot clean help

help: ## Show this help menu
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build ./bin/goclean with the version stamped in
	go build -ldflags "$(LDFLAGS)" -o bin/goclean .

install: ## Install goclean into GOPATH/bin
	go install -ldflags "$(LDFLAGS)" .

test: ## Run unit tests
	go test -race ./...

smoke: ## Scaffold a project, add a module and compile it (needs network)
	GOCLEAN_SMOKE=1 go test ./pkg/generator -run TestSmokeBuild -v

vet: ## Run go vet
	go vet ./...

lint: ## Check formatting
	@test -z "$$(gofmt -l .)" || (gofmt -l . && echo "files above need gofmt" && exit 1)

snapshot: ## Build a local multi-platform release (needs goreleaser)
	goreleaser release --snapshot --clean

clean: ## Remove build output
	rm -rf bin dist
