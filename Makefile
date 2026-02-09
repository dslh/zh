BINARY   := zh
MODULE   := github.com/dslh/zh
VERSION  ?= dev
COMMIT   := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE     := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  := -ldflags "-X $(MODULE)/cmd.Version=$(VERSION) -X $(MODULE)/cmd.Commit=$(COMMIT) -X $(MODULE)/cmd.Date=$(DATE)"

XDG_CONFIG_HOME := $(CURDIR)/test/config
XDG_CACHE_HOME  := $(CURDIR)/test/cache

.PHONY: build test lint run clean

build:
	go build $(LDFLAGS) -o $(BINARY) .

test:
	go test ./...

lint:
	golangci-lint run ./...

run: build
	XDG_CONFIG_HOME=$(XDG_CONFIG_HOME) XDG_CACHE_HOME=$(XDG_CACHE_HOME) ./$(BINARY) $(ARGS)

clean:
	rm -f $(BINARY)
