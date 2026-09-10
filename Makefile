.PHONY: run build test integration check init web-install web-build web-dev backend

export GOCACHE := $(CURDIR)/.cache/go-build
NPM ?= npm

init:
	python3 scripts/init_config.py

web-install:
	$(NPM) --prefix web ci

web-build:
	$(NPM) --prefix web run build

web-dev:
	$(NPM) --prefix web run dev

run: init web-build
	go run . -f etc/config.yaml

# For Vite development, set BaseURL to http://127.0.0.1:5173 in a private config.
CONFIG ?= etc/config.yaml
backend: init web-build
	go run . -f $(CONFIG)

build: web-build
	go build -o bin/adn-report .

test: web-build
	go test ./...

integration:
	ADN_TEST_POSTGRES=1 go test ./internal/server -run TestPostgreSQLWorkflowAndPermissions -v

check: test
	go vet ./...
