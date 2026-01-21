.PHONY: build run test clean

build:
	go build -o podman-swarm-agent ./cmd/agent

run:
	./podman-swarm-agent

test:
	go test ./...

clean:
	rm -f podman-swarm-agent

docker-build:
	docker build -t podman-swarm:latest .

docker-compose-up:
	docker-compose up -d

docker-compose-down:
	docker-compose down
