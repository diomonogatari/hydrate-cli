BINARY  := hydrate
PREFIX  ?= $(HOME)/.local/bin
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X github.com/diomonogatari/hydrate-cli/internal/cli.version=$(VERSION)

.PHONY: build test vet fmt lint install clean run tidy

build: ## Build the binary with version stamped in
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/hydrate

test: ## Run all tests
	go test ./...

vet: ## go vet
	go vet ./...

fmt: ## Format the tree
	gofmt -w .

lint: vet ## Vet + fail on unformatted files
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then echo "gofmt needed:"; echo "$$unformatted"; exit 1; fi

install: build ## Install the binary to $(PREFIX)
	install -d $(PREFIX)
	install -m755 $(BINARY) $(PREFIX)/$(BINARY)

tidy: ## go mod tidy
	go mod tidy

run: build ## Build and show status
	./$(BINARY)

clean: ## Remove build artifacts
	rm -f $(BINARY)
