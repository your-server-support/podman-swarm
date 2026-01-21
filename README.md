# Podman Swarm - Cluster Orchestrator for Podman

> ⚠️ **Project Status: Early Development**  
> This project is in active development and not yet production-ready. APIs and features may change. Use at your own risk in production environments.

**A lightweight Docker Swarm replacement for small to medium-sized clusters with Kubernetes API compatibility.**

Podman Swarm is designed to be a simple, easy-to-use container orchestrator that:
- **Speaks Kubernetes manifests** (Deployment, Service, Ingress)
- **Provides Docker Swarm-like simplicity** and ease of deployment
- **Works great for small to medium clusters** (5-50 nodes)
- **All nodes are equal** - true peer-to-peer architecture, no master nodes
- **Security by design** - minimal privileges, runs rootless, secure by default
- **Doesn't try to be a full Kubernetes replacement** - focused simplicity
- **Essential features only** - no unnecessary complexity

## Architecture

- **Peer-to-peer cluster**: All nodes are equal, uses HashiCorp Memberlist for cluster management
- **Kubernetes compatibility**: Support for standard Kubernetes manifests (Deployment, Service, Ingress)
- **Service Discovery**: Custom implementation based on memberlist for synchronization between nodes
- **DNS resolution**: Built-in DNS server for resolving services via DNS names (Kubernetes compatible)
- **DNS Whitelist**: Whitelist of external domains for DNS resolution control
- **Ingress**: Ingress controller on each node for request routing
- **Load Balancing**: Automatic load balancing between pods
- **State Persistence**: JSON-based storage with automatic recovery and periodic backups
- **State Synchronization**: Automatic state sync between nodes every 30 seconds
- **Encryption**: AES-256-GCM encryption for all messages between nodes
- **Join Token**: Token-based system for secure node joining (similar to Docker Swarm)
- **TLS support**: Optional TLS encryption at transport level

## Components

- `cmd/agent` - Agent that runs on each node
- `cmd/psctl` - CLI tool for cluster management
- `internal/api` - API server for receiving Kubernetes manifests
- `internal/cluster` - Peer-to-peer cluster
- `internal/scheduler` - Scheduler for pod distribution
- `internal/podman` - Podman integration
- `internal/parser` - Kubernetes manifest parser
- `internal/discovery` - Service discovery (custom implementation)
- `internal/dns` - DNS server for resolving services and external domains
- `internal/ingress` - Ingress controller
- `internal/storage` - Persistent state storage and recovery
- `internal/security` - Security (encryption, tokens, TLS)
- `internal/psctl` - CLI client library

## Installation

```bash
go mod download
go build -o podman-swarm-agent ./cmd/agent
```

## Running

### First node (creates cluster)

```bash
./podman-swarm-agent --node-name=node1 --bind-addr=0.0.0.0:7946
```

A join token will be generated at startup, which should be used to join other nodes.

### Joining other nodes

```bash
./podman-swarm-agent \
  --node-name=node2 \
  --bind-addr=0.0.0.0:7946 \
  --join=node1:7946 \
  --join-token=<TOKEN_FROM_NODE1>
```

### With encryption and TLS

```bash
# First node
./podman-swarm-agent \
  --node-name=node1 \
  --bind-addr=0.0.0.0:7946 \
  --encryption-key=<ENCRYPTION_KEY> \
  --tls-cert=/path/to/cert.pem \
  --tls-key=/path/to/key.pem \
  --tls-ca=/path/to/ca.pem

# Other nodes
./podman-swarm-agent \
  --node-name=node2 \
  --bind-addr=0.0.0.0:7946 \
  --join=node1:7946 \
  --join-token=<TOKEN> \
  --encryption-key=<ENCRYPTION_KEY> \
  --tls-cert=/path/to/cert.pem \
  --tls-key=/path/to/key.pem \
  --tls-ca=/path/to/ca.pem
```

### With DNS configuration

```bash
# DNS server configuration
./podman-swarm-agent \
  --node-name=node1 \
  --dns-port=53 \
  --cluster-domain=cluster.local \
  --upstream-dns=8.8.8.8:53,8.8.4.4:53
```

For more details on security, see [SECURITY.md](SECURITY.md)

## Usage

### API Authentication

Enable API authentication for production deployments:

```bash
./podman-swarm-agent --enable-api-auth=true
```

A token will be generated and displayed in logs. Use it in API requests:

```bash
# Store token in variable
export API_TOKEN="<token-from-logs>"

# Use in requests
curl -H "Authorization: Bearer $API_TOKEN" \
  http://localhost:8080/api/v1/pods
```

### Deploying a manifest

Send a Kubernetes manifest to the API:

```bash
# Without authentication
curl -X POST http://localhost:8080/api/v1/manifests \
  -H "Content-Type: application/yaml" \
  --data-binary @deployment.yaml

# With authentication
curl -H "Authorization: Bearer $API_TOKEN" \
  -X POST http://localhost:8080/api/v1/manifests \
  -H "Content-Type: application/yaml" \
  --data-binary @deployment.yaml
```

### DNS service resolution

Services are automatically available via DNS names (recommended approach):

```bash
# Services are automatically resolved via DNS
# Example: postgres-service.default.cluster.local
```

### TCP communication between services

Services can find each other via Service Discovery API or DNS:

```bash
# Via API
curl http://localhost:8080/api/v1/services/default/postgres-service/addresses
curl http://localhost:8080/api/v1/services/default/postgres-service/endpoints

# DNS whitelist management
curl http://localhost:8080/api/v1/dns/whitelist
curl -X PUT http://localhost:8080/api/v1/dns/whitelist \
  -H "Content-Type: application/json" \
  -d '{"enabled": true, "hosts": ["google.com", "github.com"]}'
```

For more details on service communication, see [SERVICE_COMMUNICATION.md](SERVICE_COMMUNICATION.md)

## Documentation

- [TODO.md](TODO.md) - Development roadmap and planned features
- [AGENTS.md](AGENTS.md) - Agent documentation
- [PSCTL.md](PSCTL.md) - CLI tool documentation
- [STORAGE.md](STORAGE.md) - State persistence and recovery
- [TESTING.md](TESTING.md) - Testing guide and coverage
- [ARCHITECTURE.md](ARCHITECTURE.md) - System architecture
- [ROUTING.md](ROUTING.md) - HTTP/HTTPS traffic routing
- [SERVICE_COMMUNICATION.md](SERVICE_COMMUNICATION.md) - Service communication (DNS and TCP)
- [SECURITY.md](SECURITY.md) - Security and encryption
