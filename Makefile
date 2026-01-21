.PHONY: build run test clean build-all install

build:
	go build -o podman-swarm-agent ./cmd/agent

build-psctl:
	go build -o psctl ./cmd/psctl

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
