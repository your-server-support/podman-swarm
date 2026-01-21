# Podman Swarm Agent

## Overview

The Podman Swarm agent is the core component that runs on each node in the cluster. It manages containers, handles service discovery, provides DNS resolution, routes traffic, and maintains cluster membership.

## Architecture

The agent consists of multiple components that work together:

```
┌─────────────────────────────────────────────────────────┐
│                  Podman Swarm Agent                      │
│                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │   Cluster    │  │   Discovery │  │     DNS      │  │
│  │  (Memberlist)│  │   Service   │  │   Server     │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
│                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │  Scheduler  │  │   Ingress    │  │     API      │  │
│  │             │  │  Controller  │  │   Server     │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
│                                                          │
│  ┌──────────────┐  ┌──────────────┐                    │
│  │   Podman     │  │    Parser    │                    │
│  │   Client     │  │              │                    │
│  └──────────────┘  └──────────────┘                    │
└─────────────────────────────────────────────────────────┘
```

## Components

### 1. Cluster Manager (`internal/cluster`)
- **Purpose**: Manages peer-to-peer cluster membership
- **Technology**: HashiCorp Memberlist
- **Functions**:
  - Node join/leave detection
  - Message broadcasting between nodes
  - Cluster state synchronization
  - Encrypted communication

### 2. Service Discovery (`internal/discovery`)
- **Purpose**: Tracks and synchronizes service endpoints across the cluster
- **Functions**:
  - Local service registration
  - Service endpoint health checking
  - Cross-node service synchronization
  - Service lookup and resolution

### 3. DNS Server (`internal/dns`)
- **Purpose**: Provides DNS resolution for cluster services and external domains
- **Functions**:
  - Resolves service names to IP addresses
  - Supports A and SRV records
  - Forwards external DNS queries to upstream servers
  - DNS whitelist management

### 4. Ingress Controller (`internal/ingress`)
- **Purpose**: Routes HTTP/HTTPS traffic to services
- **Functions**:
  - Ingress rule processing
  - Reverse proxy to services
  - Load balancing (round-robin)
  - Local optimization

### 5. Scheduler (`internal/scheduler`)
- **Purpose**: Distributes pods across cluster nodes
- **Functions**:
  - Node selection for pods
  - Pod distribution tracking
  - Node selector support

### 6. Podman Client (`internal/podman`)
- **Purpose**: Interfaces with Podman API
- **Functions**:
  - Container creation and management
  - Image pulling
  - Container lifecycle management
  - DNS configuration for containers

### 7. Parser (`internal/parser`)
- **Purpose**: Parses Kubernetes manifests
- **Functions**:
  - YAML to Kubernetes object conversion
  - Deployment, Service, Ingress parsing

### 8. API Server (`internal/api`)
- **Purpose**: REST API for cluster management
- **Functions**:
  - Manifest deployment
  - Resource management
  - Service discovery queries
  - DNS whitelist management

## Configuration

### Command Line Flags

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--node-name` | `NODE_NAME` | `node-1` | Unique node name in the cluster |
| `--bind-addr` | `BIND_ADDR` | `0.0.0.0:7946` | Cluster bind address (Memberlist) |
| `--api-addr` | `API_ADDR` | `0.0.0.0:8080` | API server address |
| `--podman-socket` | `PODMAN_SOCKET` | `unix:///run/podman/podman.sock` | Podman socket path |
| `--data-dir` | `DATA_DIR` | `/var/lib/podman-swarm` | Data directory for persistent files |
| `--join` | `JOIN` | - | Comma-separated list of addresses to join |
| `--join-token` | `JOIN_TOKEN` | - | Join token for cluster authentication |
| `--encryption-key` | `ENCRYPTION_KEY` | - | Encryption key for cluster communication |
| `--tls-cert` | `TLS_CERT` | - | TLS certificate file path |
| `--tls-key` | `TLS_KEY` | - | TLS key file path |
| `--tls-ca` | `TLS_CA` | - | TLS CA certificate file path |
| `--tls-skip-verify` | `TLS_SKIP_VERIFY` | `false` | Skip TLS certificate verification |
| `--ingress-port` | - | `80` | Ingress controller port |
| `--enable-ingress` | - | `true` | Enable ingress controller |
| `--dns-port` | `DNS_PORT` | `53` | DNS server port |
| `--cluster-domain` | `CLUSTER_DOMAIN` | `cluster.local` | Cluster domain for DNS |
| `--upstream-dns` | `UPSTREAM_DNS` | `8.8.8.8:53,8.8.4.4:53` | Upstream DNS servers (comma-separated) |

### Example Configuration

```bash
./podman-swarm-agent \
  --node-name=node1 \
  --bind-addr=0.0.0.0:7946 \
  --api-addr=0.0.0.0:8080 \
  --dns-port=53 \
  --cluster-domain=cluster.local \
  --upstream-dns=8.8.8.8:53,8.8.4.4:53 \
  --encryption-key=$(cat /etc/podman-swarm/encryption.key) \
  --join-token=$(cat /etc/podman-swarm/join.token)
```

## Startup Sequence

1. **Configuration Loading**: Loads configuration from flags and environment variables
2. **Logger Setup**: Initializes structured logging
3. **Token Manager**: Initializes token manager for join authentication
4. **Join Token Generation**: Generates join token if this is the first node
5. **TLS Configuration**: Loads TLS certificates if provided
6. **Encryption Key**: Prepares encryption key (from config or generated)
7. **Cluster Initialization**: Connects to or creates the cluster
8. **Podman Client**: Initializes Podman API client
9. **Service Discovery**: Initializes service discovery
10. **DNS Server**: Starts DNS server for service resolution
11. **Scheduler**: Initializes pod scheduler
12. **Parser**: Initializes Kubernetes manifest parser
13. **Ingress Controller**: Starts ingress controller (if enabled)
14. **API Server**: Starts REST API server
15. **Signal Handling**: Waits for shutdown signals

## Ports

| Port | Protocol | Component | Description |
|------|----------|-----------|-------------|
| 7946 | TCP/UDP | Cluster | Memberlist cluster communication |
| 8080 | TCP | API | REST API server |
| 80 | TCP | Ingress | Ingress controller (HTTP) |
| 443 | TCP | Ingress | Ingress controller (HTTPS) |
| 53 | UDP/TCP | DNS | DNS server |

## API Endpoints

### Manifest Management
- `POST /api/v1/manifests` - Apply Kubernetes manifest
- `DELETE /api/v1/manifests/:namespace/:name` - Delete resource

### Pod Management
- `GET /api/v1/pods` - List all pods
- `GET /api/v1/pods/:namespace/:name` - Get pod details

### Deployment Management
- `GET /api/v1/deployments` - List all deployments
- `GET /api/v1/deployments/:namespace/:name` - Get deployment details

### Service Management
- `GET /api/v1/services` - List all services
- `GET /api/v1/services/:namespace/:name/endpoints` - Get service endpoints
- `GET /api/v1/services/:namespace/:name/addresses` - Get service addresses

### Node Management
- `GET /api/v1/nodes` - List all cluster nodes

### DNS Whitelist
- `GET /api/v1/dns/whitelist` - Get DNS whitelist configuration
- `PUT /api/v1/dns/whitelist` - Set DNS whitelist configuration
- `POST /api/v1/dns/whitelist/hosts` - Add host to whitelist
- `DELETE /api/v1/dns/whitelist/hosts/:host` - Remove host from whitelist

### Health
- `GET /api/v1/health` - Health check endpoint

## Data Directory

The agent stores persistent data in the data directory (default: `/var/lib/podman-swarm`):

```
/var/lib/podman-swarm/
├── encryption.key    # Encryption key for cluster communication
└── (future: state files)
```

## Security

### Encryption
- **Message-level**: AES-256-GCM encryption for all cluster messages
- **Transport-level**: Optional TLS encryption

### Authentication
- **Join Tokens**: Token-based authentication for node joining (similar to Docker Swarm)
- **Token Validation**: Tokens are validated when nodes join the cluster

See [SECURITY.md](SECURITY.md) for detailed security information.

## Logging

The agent uses structured logging with logrus:

```go
logger.Infof("Starting Podman Swarm agent: %s", cfg.NodeName)
logger.Errorf("Failed to start DNS server: %v", err)
```

Log levels:
- **Info**: Normal operation messages
- **Error**: Error conditions
- **Fatal**: Critical errors that cause shutdown

## Graceful Shutdown

The agent handles shutdown signals (SIGINT, SIGTERM):

1. Receives shutdown signal
2. Stops accepting new requests
3. Shuts down cluster connection
4. Stops DNS server
5. Stops ingress controller
6. Stops API server
7. Cleans up resources

## Troubleshooting

### Agent fails to start

**Check:**
- Podman socket is accessible
- Required ports are not in use
- Data directory is writable
- Network connectivity to cluster nodes

### Agent fails to join cluster

**Check:**
- Join token is correct
- Encryption key matches cluster
- Network connectivity to join addresses
- Firewall rules allow cluster communication

### DNS not resolving

**Check:**
- DNS server is running (check logs)
- Containers are configured with correct DNS server
- Cluster domain matches configuration
- Service is registered in service discovery

### Services not accessible

**Check:**
- Service is registered in service discovery
- Pods are running and healthy
- Ingress rules are configured correctly
- Network connectivity between nodes

## Monitoring

### Health Check

```bash
curl http://localhost:8080/api/v1/health
```

### Cluster Status

```bash
curl http://localhost:8080/api/v1/nodes
```

### Service Status

```bash
curl http://localhost:8080/api/v1/services
```

## Performance Considerations

1. **Cluster Size**: Memberlist works best with < 100 nodes
2. **Service Count**: Large number of services may impact synchronization
3. **DNS Queries**: High DNS query rate may require DNS caching
4. **Network Latency**: High latency between nodes affects cluster operations

## Best Practices

1. **Use unique node names**: Each node must have a unique name
2. **Secure encryption keys**: Store encryption keys securely
3. **Use TLS in production**: Enable TLS for production deployments
4. **Monitor cluster health**: Regularly check cluster status
5. **Backup data directory**: Backup encryption keys and state
6. **Use DNS whitelist**: Restrict external DNS queries in production
7. **Configure firewall**: Restrict access to cluster ports
8. **Regular updates**: Keep agent updated with latest version

## Example Deployment

### Docker Compose

```yaml
version: '3.8'
services:
  agent:
    image: podman-swarm-agent:latest
    command:
      - --node-name=node1
      - --bind-addr=0.0.0.0:7946
      - --api-addr=0.0.0.0:8080
      - --dns-port=53
      - --cluster-domain=cluster.local
    environment:
      - ENCRYPTION_KEY=${ENCRYPTION_KEY}
      - JOIN_TOKEN=${JOIN_TOKEN}
    volumes:
      - /var/run/podman/podman.sock:/var/run/podman/podman.sock
      - /var/lib/podman-swarm:/var/lib/podman-swarm
    ports:
      - "7946:7946"
      - "8080:8080"
      - "53:53/udp"
      - "53:53/tcp"
      - "80:80"
    network_mode: host
```

### Systemd Service

```ini
[Unit]
Description=Podman Swarm Agent
After=network.target podman.service

[Service]
Type=simple
ExecStart=/usr/local/bin/podman-swarm-agent \
  --node-name=node1 \
  --bind-addr=0.0.0.0:7946 \
  --api-addr=0.0.0.0:8080
EnvironmentFile=/etc/podman-swarm/agent.conf
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

## See Also

- [ARCHITECTURE.md](ARCHITECTURE.md) - System architecture
- [SECURITY.md](SECURITY.md) - Security features
- [ROUTING.md](ROUTING.md) - Traffic routing
- [SERVICE_COMMUNICATION.md](SERVICE_COMMUNICATION.md) - Service communication
