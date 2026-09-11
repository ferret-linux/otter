GOOS ?= $(shell go env GOOS)
GO_BUILD_ENV := CGO_ENABLED=0 GOOS=$(GOOS)

COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X github.com/ferret-linux/otter/internal/cli.commit=$(COMMIT) -X github.com/ferret-linux/otter/internal/cli.buildTime=$(BUILD_TIME)

.PHONY: version
version:
	sed -i 's/Version: "[^"]*"/Version: "$(word 2, $(MAKECMDGOALS))"/' internal/cli/root.go
	for f in internal/insidecontainer/assets/*; do \
		sed -i 's/^\(VERSION[[:space:]]*=[[:space:]]*\)"[^"]*"/\1"$(word 2, $(MAKECMDGOALS))"/; s/^\(version=\)"[^"]*"/\1"$(word 2, $(MAKECMDGOALS))"/' "$$f"; \
	done

%:
	@:

.PHONY: build
build:
	$(GO_BUILD_ENV) go build -ldflags "$(LDFLAGS)" -o ./bin/otter ./cmd/otter

.PHONY: completions
completions: build
	./bin/otter completion bash > completions/otter.bash
	./bin/otter completion fish > completions/otter.fish
	./bin/otter completion zsh > completions/otter.zsh

.PHONY: test
test: vet
	$(GO_BUILD_ENV) go test -v ./...

.PHONY: vet
vet:
	$(GO_BUILD_ENV) go vet ./...

.PHONY: fmt
fmt:
	$(GO_BUILD_ENV) go fmt ./...

PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin
BASHCOMPLETION_DIR ?= $(PREFIX)/share/bash-completion/completions
FISHCOMPLETION_DIR ?= $(HOME)/.config/fish/completions
ZSHCOMPLETION_DIR ?= $(PREFIX)/share/zsh/site-functions

.PHONY: install
install: build
	install -d $(DESTDIR)$(BINDIR)
	install -m 0755 ./bin/otter $(DESTDIR)$(BINDIR)/otter
	install -d $(DESTDIR)$(BASHCOMPLETION_DIR)
	install -m 0644 ./completions/otter.bash $(DESTDIR)$(BASHCOMPLETION_DIR)/otter
	install -d $(DESTDIR)$(FISHCOMPLETION_DIR)
	install -m 0644 ./completions/otter.fish $(DESTDIR)$(FISHCOMPLETION_DIR)/otter.fish
	install -d $(DESTDIR)$(ZSHCOMPLETION_DIR)
	install -m 0644 ./completions/otter.zsh $(DESTDIR)$(ZSHCOMPLETION_DIR)/_otter

.PHONY: uninstall
uninstall:
	rm -f $(DESTDIR)$(BINDIR)/otter
	rm -f $(DESTDIR)$(BASHCOMPLETION_DIR)/otter
	rm -f $(DESTDIR)$(FISHCOMPLETION_DIR)/otter.fish
	rm -f $(DESTDIR)$(ZSHCOMPLETION_DIR)/_otter

.PHONY: clean
clean:
	rm -f ./bin/otter

.PHONY: lint
lint:
	$(GO_BUILD_ENV) golangci-lint run --verbose

.PHONY: lint-fix
lint-fix:
	$(GO_BUILD_ENV) golangci-lint run --fix

.PHONY: newline-fix
newline-fix:
	@files=$$(git ls-files '*.sh' '*.go' '*.help' '*.conf' '*.fish'; git ls-files | grep -E '(^|/)(sudoers|pam-su)$$'); \
	files=$$(printf '%s\n' "$$files" | sort -u); \
	for f in $$files; do \
		[ -s "$$f" ] || continue; \
		perl -i -0pe 's/\n*\z/\n/' "$$f"; \
	done