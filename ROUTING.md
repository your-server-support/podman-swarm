# Traffic Routing in Podman Swarm

> **Note**: This document describes HTTP/HTTPS routing via Ingress.
> For TCP communication between services, see [SERVICE_COMMUNICATION.md](SERVICE_COMMUNICATION.md)

## Routing Architecture

Traffic routing in Podman Swarm works at two levels:

### 1. Ingress Controller (Level 1 - Incoming Traffic)

Ingress Controller runs on **each node** in the cluster on port 80 (default). It handles incoming HTTP/HTTPS requests and routes them to the appropriate services.

**Request flow:**
```
User → Ingress (Node A:80) → Service Discovery → Pod (Node B:8080)
```

### 2. Service Discovery (Level 2 - Internal Routing)

Service Discovery stores information about all service pods and their locations in the cluster. Each node has a complete picture of all services thanks to synchronization via memberlist.

## Detailed Routing Process

### Step 1: Request arrives at Ingress

```
HTTP Request: GET http://example.com/api/users
Host: example.com
Path: /api/users
```

### Step 2: Ingress finds rule

Ingress Controller checks all registered Ingress rules and finds the matching one:
- Checks `Host` header
- Checks `Path` for match (Exact, Prefix, ImplementationSpecific)
- Finds corresponding Service

### Step 3: Service Discovery finds endpoints

```go
addresses, err := discovery.GetServiceAddresses("nginx-service", "default")
// Returns: ["node1:8080", "node2:8080", "node3:8080"]
```

Service Discovery returns a list of all healthy endpoints of the service from all cluster nodes.

### Step 4: Load Balancing

Ingress Controller selects an endpoint using round-robin algorithm:
- If pod is on the same node - direct access via localhost
- If pod is on another node - proxying via HTTP to remote node

### Step 5: Request Proxying

```go
// If pod is on local node
target = "localhost:8080"

// If pod is on remote node
target = "node2:8080"  // Proxying via HTTP
```

## Inter-node Routing Example

### Scenario: 3 nodes, 3 service pods

```
Node 1:
  - Ingress Controller (port 80)
  - Pod nginx-1 (port 8080)

Node 2:
  - Ingress Controller (port 80)
  - Pod nginx-2 (port 8080)

Node 3:
  - Ingress Controller (port 80)
  - Pod nginx-3 (port 8080)
```

**Request 1:** `GET http://example.com/` on Node 1
- Ingress on Node 1 finds rule for `example.com`
- Service Discovery returns: `["node1:8080", "node2:8080", "node3:8080"]`
- Round-robin selects `node1:8080` (local)
- Proxying to `localhost:8080` ✅

**Request 2:** `GET http://example.com/` on Node 2
- Ingress on Node 2 finds rule
- Service Discovery returns the same addresses
- Round-robin selects `node2:8080` (local)
- Proxying to `localhost:8080` ✅

**Request 3:** `GET http://example.com/` on Node 1
- Round-robin selects `node2:8080` (remote)
- Proxying via HTTP to `node2:8080` ✅

## Load Balancing Implementation

Current implementation uses **indexed round-robin**:

```go
// Round-robin selection
idx := ic.roundRobinIdx[proxyKey]
selectedEndpoint := endpoints[idx%len(endpoints)]
ic.roundRobinIdx[proxyKey] = (idx + 1) % len(endpoints)
```

**Features:**
- ✅ Real round-robin (each request selects next endpoint)
- ✅ Health-aware routing (only healthy endpoints)
- ✅ Local optimization (local pods have priority)

**Planned improvements:**
- Weighted round-robin
- Least connections
- Sticky sessions
- Circuit breaker for unhealthy endpoints

## Implementation Details

### 1. Local vs Remote Access

```go
if selectedEndpoint.NodeName == ic.localNodeName {
    // Local pod - direct access via localhost
    target = fmt.Sprintf("localhost:%d", selectedEndpoint.Port)
} else {
    // Remote pod - proxying via HTTP to node
    // Address contains real node IP from cluster
    target = fmt.Sprintf("%s:%d", selectedEndpoint.Address, selectedEndpoint.Port)
}
```

**Important:** 
- Local pods are accessible via `localhost:port`
- Remote pods are accessible via `node-ip:port`
- Node address is obtained from cluster when service is registered

### 2. Service Discovery Synchronization

Each node has a complete picture of services:
- When service is registered - broadcast via memberlist
- When update is received - synchronize local registry
- Health check every 10 seconds

### 3. Proxy Caching

Ingress Controller caches reverse proxy for each service:
```go
proxies[serviceKey] = httputil.NewSingleHostReverseProxy(targetURL)
```

## Limitations and Improvements

### Current Capabilities:
1. ✅ Real indexed round-robin
2. ✅ Health-aware routing (only healthy endpoints)
3. ✅ Local optimization (local pods)
4. ✅ Automatic synchronization between nodes

### Planned:
1. ⏳ Weighted balancing
2. ⏳ Least connections
3. ⏳ Sticky sessions (based on cookies)
4. ⏳ Circuit breaker for unhealthy endpoints
5. ⏳ Routing metrics and monitoring

## Routing Diagram

```
                    ┌─────────────────┐
                    │   User Request   │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │  Ingress (Node) │
                    │   Port 80/443   │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │ Service Discovery│
                    │  (Local Registry)│
                    └────────┬────────┘
                             │
            ┌────────────────┼────────────────┐
            │                │                │
    ┌───────▼──────┐  ┌──────▼──────┐  ┌──────▼──────┐
    │  Pod (Node1) │  │  Pod (Node2)│  │  Pod (Node3)│
    │   :8080      │  │   :8080     │  │   :8080     │
    └──────────────┘  └─────────────┘  └─────────────┘
```

## Configuration

### Disable Ingress on node

```bash
./podman-swarm-agent --enable-ingress=false
```

### Change Ingress port

```bash
./podman-swarm-agent --ingress-port=8080
```

### Example Kubernetes Ingress Manifest

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: example-ingress
spec:
  rules:
  - host: example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: nginx-service
            port:
              number: 80
```
