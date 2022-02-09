#!/usr/bin/make -f
SRC_DIR	:= $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
TMP_DIR ?= $(SRC_DIR)/tmp
TMP_BIN_DIR ?= $(TMP_DIR)/bin

PROMRUVAL_BIN := promruval

$(TMP_DIR):
	mkdir -p $(TMP_DIR)

$(TMP_BIN_DIR):
	mkdir -p $(TMP_BIN_DIR)

RELEASE_NOTES ?= $(TMP_DIR)/release_notes
$(RELEASE_NOTES): $(TMP_DIR)
	@echo "Generating release notes to $(RELEASE_NOTES) ..."
	@csplit -q -n1 --suppress-matched -f $(TMP_DIR)/release-notes-part CHANGELOG.md '/## \[\s*v.*\]/' {1}
	@mv $(TMP_DIR)/release-notes-part1 $(RELEASE_NOTES)
	@rm $(TMP_DIR)/release-notes-part*

lint:
	golangci-lint run

test:
	go test -race ./...

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(PROMRUVAL_BIN)

docker: build
	docker build -t fusakla/promruval .

.PHONY: clean
clean:
	rm -rf dist $(TMP_DIR) $(PROMRUVAL_BIN)

.PHONY: deps
deps:
	go mod tidy && go mod verify
