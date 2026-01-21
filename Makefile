.PHONY: build run test clean build-all install

BUILD_TAGS = exclude_graphdriver_btrfs,exclude_graphdriver_devicemapper,containers_image_openpgp

build:
	CGO_ENABLED=0 go build -tags $(BUILD_TAGS) -o podman-swarm-agent ./cmd/agent

build-psctl:
	CGO_ENABLED=0 go build -o psctl ./cmd/psctl

build-all: build build-psctl

install:
	go install ./cmd/agent
	go install ./cmd/psctl

run:
	./podman-swarm-agent

test:
	go test ./...

clean:
	rm -f podman-swarm-agent psctl

docker-build:
	docker build -t podman-swarm:latest .

docker-compose-up:
	docker-compose up -d

docker-compose-down:
	docker-compose down
