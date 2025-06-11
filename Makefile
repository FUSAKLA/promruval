#!/usr/bin/make -f
SRC_DIR	:= $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
CACHE_FILE := .promruval_cache.json

PROMRUVAL_BIN := ./promruval
RELEASE_NOTES := release_notes.md

all: clean deps lint build test e2e-test test-release

$(RELEASE_NOTES):
	@cat CHANGELOG.md | grep -E -A 1000 -m 1 "## \[[0-9]" | grep -E -B 1000 -m 2 "## \[[0-9]" | head -n-1 | tail -n+2 > $(RELEASE_NOTES)

lint:
	golangci-lint run

lint-fix:
	golangci-lint run --fix

test:
	go test -race ./...

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(PROMRUVAL_BIN)

E2E_TESTS_VALIDATIONS_FILE := examples/validation.yaml
E2E_TESTS_ADDITIONAL_VALIDATIONS_FILE := examples/additional-validation.jsonnet
E2E_TESTS_RULES_FILES := examples/rules/*
E2E_TESTS_DOCS_FILE_TEXT := examples/human_readable.txt
E2E_TESTS_DOCS_FILE_MD := examples/human_readable.md
E2E_TESTS_DOCS_FILE_HTML := examples/human_readable.html

E2E_TESTS_LOKI_DIR := examples/loki/
E2E_TESTS_MIMIR_DIR := examples/mimir/
E2E_TESTS_THANOS_DIR := examples/thanos/
e2e-test: build
	$(PROMRUVAL_BIN) validate --config-file $(E2E_TESTS_VALIDATIONS_FILE) --config-file $(E2E_TESTS_ADDITIONAL_VALIDATIONS_FILE) $(E2E_TESTS_RULES_FILES)
	$(PROMRUVAL_BIN) validate --config-file $(E2E_TESTS_VALIDATIONS_FILE) --config-file $(E2E_TESTS_ADDITIONAL_VALIDATIONS_FILE) -o json $(E2E_TESTS_RULES_FILES)
	$(PROMRUVAL_BIN) validate --config-file $(E2E_TESTS_VALIDATIONS_FILE) --config-file $(E2E_TESTS_ADDITIONAL_VALIDATIONS_FILE) -o yaml $(E2E_TESTS_RULES_FILES)
	$(PROMRUVAL_BIN) validation-docs --config-file $(E2E_TESTS_VALIDATIONS_FILE) --config-file $(E2E_TESTS_ADDITIONAL_VALIDATIONS_FILE) --output=text > $(E2E_TESTS_DOCS_FILE_TEXT)
	$(PROMRUVAL_BIN) validation-docs --config-file $(E2E_TESTS_VALIDATIONS_FILE) --config-file $(E2E_TESTS_ADDITIONAL_VALIDATIONS_FILE) --output=markdown > $(E2E_TESTS_DOCS_FILE_MD)
	$(PROMRUVAL_BIN) validation-docs --config-file $(E2E_TESTS_VALIDATIONS_FILE) --config-file $(E2E_TESTS_ADDITIONAL_VALIDATIONS_FILE) --output=html > $(E2E_TESTS_DOCS_FILE_HTML)

	$(PROMRUVAL_BIN) validate --support-loki --config-file $(E2E_TESTS_LOKI_DIR)/validation.yaml $(E2E_TESTS_LOKI_DIR)/rules.yaml
	$(PROMRUVAL_BIN) validate --support-thanos --config-file $(E2E_TESTS_THANOS_DIR)/validation.yaml $(E2E_TESTS_THANOS_DIR)/rules.yaml
	$(PROMRUVAL_BIN) validate --support-mimir --config-file $(E2E_TESTS_MIMIR_DIR)/validation.yaml $(E2E_TESTS_MIMIR_DIR)/rules.yaml

docker: build
	docker build -t fusakla/promruval .

.PHONY: clean
clean:
	rm -rf dist $(RELEASE_NOTES) $(PROMRUVAL_BIN) $(CACHE_FILE)

.PHONY: deps
deps:
	go mod tidy && go mod verify

test-release: release_notes.md
	goreleaser release --snapshot --clean --release-notes release_notes.md
