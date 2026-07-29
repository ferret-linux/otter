GOOS ?= $(shell go env GOOS)
GO_BUILD_ENV := CGO_ENABLED=0 GOOS=$(GOOS)

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
	$(GO_BUILD_ENV) go build -o ./bin/otter ./cmd/otter

.PHONY: test
test: vet
	$(GO_BUILD_ENV) go test -v ./...

.PHONY: vet
vet:
	$(GO_BUILD_ENV) go vet ./...

.PHONY: fmt
fmt:
	$(GO_BUILD_ENV) go fmt ./...

PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin

.PHONY: install
install: build
	install -d $(DESTDIR)$(BINDIR)
	install -m 0755 ./bin/otter $(DESTDIR)$(BINDIR)/otter

.PHONY: uninstall
uninstall:
	rm -f $(DESTDIR)$(BINDIR)/otter

.PHONY: clean
clean:
	rm -f ./bin/otter

.PHONY: lint
lint:
	$(GO_BUILD_ENV) golangci-lint run --verbose

.PHONY: lint-fix
lint-fix:
	$(GO_BUILD_ENV) golangci-lint run --fix
