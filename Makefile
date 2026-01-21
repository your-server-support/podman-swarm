.PHONY: build run test clean build-all install test-unit test-coverage

BUILD_TAGS = exclude_graphdriver_btrfs,exclude_graphdriver_devicemapper,containers_image_openpgp

build:
	CGO_ENABLED=0 go build -tags $(BUILD_TAGS) -o podman-swarm-agent ./cmd/agent

build-psctl:
	CGO_ENABLED=0 go build -o psctl ./cmd/psctl

build-all: build build-psctl

install:
	install -m 755 podman-swarm-agent /usr/local/bin/
	install -m 755 psctl /usr/local/bin/

run:
	./podman-swarm-agent

test-unit:
	CGO_ENABLED=0 go test -v -tags $(BUILD_TAGS) \
		./internal/storage \
		./internal/security \
		./internal/parser

test-coverage:
	CGO_ENABLED=0 go test -v -tags $(BUILD_TAGS) \
		-coverprofile=coverage.out \
		-covermode=atomic \
		./internal/storage \
		./internal/security \
		./internal/parser
	go tool cover -html=coverage.out -o coverage.html

test:
	@echo "Running unit tests..."
	@make test-unit

clean:
	rm -f podman-swarm-agent psctl coverage.out coverage.html
