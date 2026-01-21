# Service Communication in Podman Swarm

## Communication Architecture

In Podman Swarm, services can communicate with each other in three ways:

1. **Via Ingress (HTTP/HTTPS)** - for external traffic
2. **Via DNS resolution** - for internal communication between services (recommended)
3. **Direct TCP communication via Service Discovery API** - for internal communication between services

## DNS Service Resolution (Recommended Method)

### How It Works

Podman Swarm provides a built-in DNS server that automatically resolves service names to IP addresses. This is the simplest and most convenient way for containers to find other services.

**DNS Resolution Advantages:**
- ✅ Standard approach (Kubernetes compatible)
- ✅ No need to know Service Discovery API
- ✅ Works with any programming language
- ✅ Automatic load balancing via multiple A records
- ✅ SRV record support for ports

### DNS Name Format

Services are available via the following DNS names:

```
<service-name>.<namespace>.cluster.local
<service-name>.<namespace>.svc.cluster.local  # Kubernetes compatibility
```

**Examples:**
- `postgres-service.default.cluster.local` → resolves to IP addresses of all endpoints
- `redis.cache.cluster.local` → resolves to Redis service IP addresses
- `api.production.cluster.local` → resolves to API service IP addresses

### Automatic Configuration

Each container is automatically configured to use the cluster DNS server:
- DNS server runs on each node
- Containers receive DNS server via Podman `--dns` option
- Queries to `cluster.local` are handled locally
- Other queries are forwarded to upstream DNS servers

### Usage Examples

#### Python Example

```python
import socket

# Simply use DNS name
host = "postgres-service.default.cluster.local"
port = 5432

# Standard DNS resolution
ip = socket.gethostbyname(host)
print(f"Resolved {host} to {ip}")

# PostgreSQL connection
import psycopg2
conn = psycopg2.connect(
    host=host,  # DNS name is automatically resolved
    port=port,
    database="mydb",
    user="postgres"
)
```

#### Go Example

```go
package main

import (
    "net"
    "fmt"
)

func main() {
    // DNS resolution
    host := "postgres-service.default.cluster.local"
    addrs, err := net.LookupHost(host)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Resolved %s to: %v\n", host, addrs)
    
    // Connect to first IP
    conn, err := net.Dial("tcp", fmt.Sprintf("%s:5432", addrs[0]))
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()
}
```

#### Node.js Example

```javascript
const dns = require('dns');
const net = require('net');

// DNS resolution
const host = 'postgres-service.default.cluster.local';
dns.lookup(host, (err, address) => {
    if (err) {
        console.error(err);
        return;
    }
    
    console.log(`Resolved ${host} to ${address}`);
    
    // Connection
    const socket = net.createConnection(5432, address);
});
```

#### Java Example

```java
import java.net.InetAddress;

String host = "postgres-service.default.cluster.local";
InetAddress address = InetAddress.getByName(host);
System.out.println("Resolved to: " + address.getHostAddress());

// JDBC connection string
String jdbcUrl = "jdbc:postgresql://" + host + ":5432/mydb";
```

### SRV Records for Ports

SRV records can be used to get port information:

```
_<port-name>._<protocol>.<service-name>.<namespace>.cluster.local
```

**Example:**
```
_http._tcp.api-service.default.cluster.local
```

### Load Balancing

The DNS server returns multiple A records for each service (one per healthy endpoint). Most DNS clients automatically select one of them (round-robin or random).

**Example:**
```bash
# DNS query returns multiple IP addresses
$ dig postgres-service.default.cluster.local

;; ANSWER SECTION:
postgres-service.default.cluster.local. 60 IN A 10.0.1.1
postgres-service.default.cluster.local. 60 IN A 10.0.1.2
postgres-service.default.cluster.local. 60 IN A 10.0.1.3
```

### Upstream DNS Configuration

The DNS server automatically forwards queries that don't belong to `cluster.local` to external DNS servers.

**Configuration:**
```bash
# Use Google DNS (default)
--upstream-dns=8.8.8.8:53,8.8.4.4:53

# Use Cloudflare DNS
--upstream-dns=1.1.1.1:53,1.0.0.1:53

# Use system resolver
--upstream-dns=127.0.0.1:53
```

**Example:**
- `postgres-service.default.cluster.local` → handled locally
- `google.com` → forwarded to upstream DNS
- `github.com` → forwarded to upstream DNS

## TCP Communication via Service Discovery API

### How It Works

When service A wants to connect to service B:

1. **Service Discovery request**: Service A queries service B addresses via Service Discovery
2. **Get endpoints**: Service Discovery returns a list of all healthy endpoints of service B
3. **Select endpoint**: Service A selects an endpoint (round-robin or other algorithm)
4. **TCP connection**: Direct TCP connection to selected endpoint

### Flow Example

```
Service A (Pod on Node 1) wants to connect to Service B

1. Service A → Service Discovery API
   GET /api/v1/services/namespace/service-b/endpoints

2. Service Discovery returns:
   [
     {NodeName: "node1", Address: "10.0.1.1", Port: 5432},
     {NodeName: "node2", Address: "10.0.1.2", Port: 5432},
     {NodeName: "node3", Address: "10.0.1.3", Port: 5432}
   ]

3. Service A selects endpoint (e.g., node2:5432)

4. Service A establishes TCP connection:
   conn, err := net.Dial("tcp", "10.0.1.2:5432")
```

## Implementation

### Service Discovery API

Service Discovery provides API for getting service addresses:

```go
// Get all service addresses
addresses, err := discovery.GetServiceAddresses("postgres-service", "default")
// Returns: ["10.0.1.1:5432", "10.0.1.2:5432", "10.0.1.3:5432"]

// Get detailed endpoint information
endpoints, err := discovery.GetServiceEndpoints("postgres-service", "default")
// Returns: []*ServiceEndpoint with NodeName, Address, Port, Healthy
```

### Code Usage Examples

### Option 1: DNS Resolution (Recommended)

**Simplest way** - just use DNS names:

```go
// Simply use DNS name - everything else works automatically
host := "postgres-service.default.cluster.local"
port := 5432

conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
// DNS is automatically resolved to IP address
```

### Option 2: Direct Service Discovery Usage (within cluster)

```go
package main

import (
    "net"
    "fmt"
    "github.com/your-server-support/podman-swarm/internal/discovery"
)

func connectToService(discovery *discovery.Discovery, serviceName, namespace string) (net.Conn, error) {
    // Get service addresses
    addresses, err := discovery.GetServiceAddresses(serviceName, namespace)
    if err != nil {
        return nil, fmt.Errorf("service not found: %w", err)
    }

    // Round-robin: select first endpoint
    // In production, you can use a more complex algorithm
    target := addresses[0]

    // Establish TCP connection
    conn, err := net.Dial("tcp", target)
    if err != nil {
        return nil, fmt.Errorf("failed to connect: %w", err)
    }

    return conn, nil
}
```

### Option 3: Via API (for external clients or pods)

```go
// Use HTTP API to get addresses
resp, err := http.Get("http://node1:8080/api/v1/services/default/postgres-service/addresses")
// Get: {"addresses": ["10.0.1.1:5432", "10.0.1.2:5432"]}

// Then establish TCP connection
conn, err := net.Dial("tcp", "10.0.1.1:5432")
```

Full client example see in `examples/service-client.go`

## Implementation Details

### 1. Direct TCP Connection

Services establish **direct TCP connections** to pods:
- No intermediate proxies
- Minimal latency
- Support for any protocol over TCP (HTTP, gRPC, PostgreSQL, Redis, etc.)

### 2. Pod Location

```
Service A (Node 1) → Service B (Node 2)
   ↓                    ↓
TCP Connection: 10.0.1.1:xxxx → 10.0.1.2:5432
```

### 3. Load Balancing

Client service itself selects endpoint:
- Round-robin
- Random selection
- Health-aware (only healthy endpoints)
- Weighted selection (if implemented)

### 4. Local Optimization

If pod is on the same node:
```go
if endpoint.NodeName == localNodeName {
    // Use localhost for lower latency
    target = fmt.Sprintf("localhost:%d", endpoint.Port)
} else {
    // Use node IP address
    target = fmt.Sprintf("%s:%d", endpoint.Address, endpoint.Port)
}
```

## Usage Examples

### Example 1: PostgreSQL Connection (via DNS)

```go
// Simplest way - use DNS name
dsn := "host=postgres-service.default.cluster.local port=5432 user=postgres dbname=mydb sslmode=disable"
db, err := sql.Open("postgres", dsn)
// PostgreSQL driver automatically resolves DNS name
```

**Alternative via Service Discovery:**
```go
// Get PostgreSQL service addresses
addresses, err := discovery.GetServiceAddresses("postgres", "default")
if err != nil {
    log.Fatal(err)
}

// Connect to first endpoint
dsn := fmt.Sprintf("host=%s port=%s user=postgres dbname=mydb sslmode=disable",
    strings.Split(addresses[0], ":")[0],
    strings.Split(addresses[0], ":")[1])

db, err := sql.Open("postgres", dsn)
```

### Example 2: Redis Connection (via DNS)

```go
// Simplest way - use DNS name
client := redis.NewClient(&redis.Options{
    Addr: "redis-service.default.cluster.local:6379",
})
// Redis client automatically resolves DNS name
```

**Alternative via Service Discovery:**
```go
// Get Redis service addresses
endpoints, err := discovery.GetServiceEndpoints("redis", "default")
if err != nil {
    log.Fatal(err)
}

// Select healthy endpoint
var selectedEndpoint *discovery.ServiceEndpoint
for _, ep := range endpoints {
    if ep.Healthy {
        selectedEndpoint = ep
        break
    }
}

// Connect
client := redis.NewClient(&redis.Options{
    Addr: fmt.Sprintf("%s:%d", selectedEndpoint.Address, selectedEndpoint.Port),
})
```

### Example 3: gRPC Connection (via DNS)

```go
// Simplest way - use DNS name
conn, err := grpc.Dial("grpc-service.default.cluster.local:50051", grpc.WithInsecure())
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

client := pb.NewMyServiceClient(conn)
// gRPC automatically resolves DNS name
```

**Alternative via Service Discovery:**
```go
// Get gRPC service addresses
addresses, err := discovery.GetServiceAddresses("grpc-service", "default")
if err != nil {
    log.Fatal(err)
}

// Connect to service
conn, err := grpc.Dial(addresses[0], grpc.WithInsecure())
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

client := pb.NewMyServiceClient(conn)
```

## Service Discovery Synchronization

### Automatic Synchronization

Each node has a complete picture of all services:

1. **Registration**: When Service is created, all pods are registered in Service Discovery
2. **Broadcast**: Service information is broadcast via memberlist
3. **Synchronization**: All nodes receive updates and synchronize local registry
4. **Health Check**: Endpoint health is checked every 10 seconds

### Real-time Updates

- When new pod is added → automatically added to Service Discovery
- When pod is deleted → automatically removed from Service Discovery
- When pod fails → marked as unhealthy after 30 seconds

## Communication Method Comparison

### DNS Resolution (Recommended)

```
Service A → DNS query → DNS server → Service Discovery → Real Node IP:Port → Direct TCP → Pod
```

**Advantages:**
- ✅ Standard approach (Kubernetes compatible)
- ✅ Works with any programming language
- ✅ No need to know Service Discovery API
- ✅ Automatic load balancing via multiple A records
- ✅ SRV record support

**Disadvantages:**
- ❌ Dependency on DNS server
- ❌ Possible delays due to DNS cache

### Service Discovery API

```
Service A → Service Discovery API → Real Node IP:Port → Direct TCP → Pod
```

**Advantages:**
- ✅ Direct access to endpoint information
- ✅ Ability to select specific endpoint
- ✅ Detailed information (health, node name, etc.)

**Disadvantages:**
- ❌ Need to know Service Discovery API
- ❌ Additional code for integration

## Comparison with Kubernetes

### Kubernetes
```
Service A → DNS → ClusterIP (Virtual IP) → kube-proxy → iptables/ipvs → Pod
```

### Podman Swarm
```
Service A → DNS → DNS server → Service Discovery → Real Node IP:Port → Direct TCP → Pod
```

**Podman Swarm Approach Advantages:**
- ✅ Simpler architecture (no virtual IPs)
- ✅ Direct connection (lower latency)
- ✅ Support for any protocol
- ✅ Easier debugging (real addresses visible)
- ✅ Kubernetes DNS format compatibility

**Disadvantages:**
- ❌ Client service must select endpoint itself (via DNS round-robin)
- ❌ No automatic network-level load balancing (like in Kubernetes)

## Network Configuration

### Requirements

For inter-service communication you need:

1. **DNS server**: Automatically configured on each node
2. **Network accessibility**: All nodes must be accessible to each other
3. **Firewall**: Pod ports must be open between nodes
4. **Upstream DNS**: External DNS server configuration (optional)

### DNS Configuration

DNS server is automatically configured for containers:
- Containers receive DNS server via Podman `--dns` option
- DNS server listens on port 53 (default)
- Queries to `cluster.local` are handled locally
- Other queries are forwarded to upstream DNS

**Upstream DNS Configuration:**
```bash
# Via environment variable
export UPSTREAM_DNS="8.8.8.8:53,8.8.4.4:53"

# Via command line parameter
./agent --upstream-dns=1.1.1.1:53,1.0.0.1:53
```

### Firewall Configuration Example

```bash
# Allow communication between nodes
iptables -A INPUT -s 10.0.1.0/24 -j ACCEPT
iptables -A OUTPUT -d 10.0.1.0/24 -j ACCEPT
```

## TCP Communication Diagram

```
┌─────────────────────────────────────────────────────────┐
│                    Service A (Node 1)                   │
│  ┌──────────────────────────────────────────────────┐   │
│  │  1. Request to Service Discovery                 │   │
│  │     GetServiceAddresses("service-b", "default")  │   │
│  └──────────────────┬───────────────────────────────┘   │
│                     │                                   │
│                     ▼                                   │
│  ┌──────────────────────────────────────────────────┐   │
│  │  2. Gets addresses:                              │   │
│  │     ["10.0.1.2:5432", "10.0.1.3:5432"]           │   │
│  └──────────────────┬───────────────────────────────┘   │
│                     │                                   │
│                     ▼                                   │
│  ┌──────────────────────────────────────────────────┐   │
│  │  3. Establishes TCP connection                   │   │
│  │     net.Dial("tcp", "10.0.1.2:5432")             │   │
│  └──────────────────┬───────────────────────────────┘   │
└──────────────────────┼──────────────────────────────────┘
                       │
                       │ TCP Connection
                       │
┌──────────────────────▼──────────────────────────────────┐
│                    Service B (Node 2)                   │
│  ┌──────────────────────────────────────────────────┐   │
│  │  Pod listens on port 5432                       │   │
│  │  net.Listen("tcp", ":5432")                      │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

## Recommendations

1. **Use DNS resolution** as the primary method for inter-service communication
2. **Use connection pooling** for efficiency
3. **Implement retry logic** for handling temporary failures
4. **Use health checks** before connecting (DNS automatically filters unhealthy endpoints)
5. **Cache DNS queries** - most DNS clients automatically cache results
6. **Monitor latency** between nodes for optimization
7. **Configure upstream DNS** for faster external domain resolution
