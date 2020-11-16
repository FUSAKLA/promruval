#!/usr/bin/make -f
SRC_DIR	:= $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
TMP_DIR ?= $(SRC_DIR)/tmp
TMP_BIN_DIR ?= $(TMP_DIR)/bin

PROMRUVAL_BIN := promruval

GORELEASER_VERSION ?= v0.147.2

$(TMP_DIR):
	mkdir -p $(TMP_DIR)

$(TMP_BIN_DIR):
	mkdir -p $(TMP_BIN_DIR)

GORELEASER ?= $(TMP_BIN_DIR)/goreleaser
$(GORELEASER): $(TMP_BIN_DIR)
	@echo "Downloading goreleaser version $(GORELEASER_VERSION) to $(TMP_BIN_DIR) ..."
	@curl -sNL "https://github.com/goreleaser/goreleaser/releases/download/$(GORELEASER_VERSION)/goreleaser_Linux_x86_64.tar.gz" | tar -xzf - -C $(TMP_BIN_DIR)

RELEASE_NOTES ?= $(TMP_DIR)/release_notes
$(RELEASE_NOTES): $(TMP_DIR)
	@echo "Generating release notes to $(RELEASE_NOTES) ..."
	@csplit -q -n1 --suppress-matched -f $(TMP_DIR)/release-notes-part CHANGELOG.md '/## \[\s*v.*\]/' {1}
	@mv $(TMP_DIR)/release-notes-part1 $(RELEASE_NOTES)
	@rm $(TMP_DIR)/release-notes-part*

test:
	go test -race ./...

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(PROMRUVAL_BIN) cmd/promruval.go

docker: build
	docker build -t fusakla/promruval .

.PHONY: release
release: $(RELEASE_NOTES) $(GORELEASER)
	@echo "Releasing new version do GitHub and DockerHub using goreleaser..."
	$(GORELEASER) release --rm-dist --release-notes $(RELEASE_NOTES)

.PHONY: clean
clean:
	rm -rf dist $(TMP_DIR) $(PROMRUVAL_BIN)
