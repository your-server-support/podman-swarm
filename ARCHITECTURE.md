# Podman Swarm Architecture

## Overview

Podman Swarm is a cluster orchestrator for Podman that provides Kubernetes manifest compatibility and peer-to-peer architecture.

## Project Philosophy

Podman Swarm is designed as a **Docker Swarm replacement** for small to medium-sized clusters (5-50 nodes) with Kubernetes API compatibility. Key principles:

### Core Design Goals

- **Simplicity over complexity** - Easy to deploy and manage, like Docker Swarm
- **Kubernetes API compatible** - Use standard Kubernetes manifests without full k8s complexity
- **Lightweight** - Minimal resource footprint, suitable for edge and small deployments
- **Essential features only** - Focus on core orchestration without enterprise-grade features

### Architecture Principles

- **True peer-to-peer** - All nodes are equal, no master/worker distinction
  - No single point of failure
  - No need for dedicated control plane nodes
  - Every node can handle API requests
  - Automatic leader election when needed (e.g., scheduling)

- **Security by design**
  - **Principle of least privilege** - Components run with minimal required permissions
  - **Rootless by default** - Leverages Podman's rootless capabilities
  - **Encrypted communication** - AES-256-GCM encryption for all inter-node messages
  - **Token-based authentication** - Secure node joining with join tokens
  - **API authentication** - Optional but encouraged API token authentication
  - **Network isolation** - DNS whitelist for external access control

- **No external dependencies**
  - No need for etcd or other distributed databases
  - Uses efficient peer-to-peer synchronization (memberlist)
  - Self-contained and easy to deploy

## Components

### 1. Cluster (internal/cluster)
- **Purpose**: Manages peer-to-peer cluster
- **Technology**: HashiCorp Memberlist
- **Functions**:
  - Node clustering
  - Node state tracking
  - Message broadcasting between nodes

### 2. Parser (internal/parser)
- **Purpose**: Parses Kubernetes manifests
- **Supported resources**:
  - Deployment
  - Service
  - Ingress
- **Functions**:
  - YAML to Kubernetes object conversion
  - Extracts information about pods, services, and ingress

### 3. Scheduler (internal/scheduler)
- **Purpose**: Distributes pods across nodes
- **Strategies**:
  - Random selection (default)
  - Node selector (for binding to specific nodes)
- **Functions**:
  - Node selection for pods
  - Pod distribution tracking

### 4. Podman Client (internal/podman)
- **Purpose**: Podman integration
- **Functions**:
  - Container creation
  - Container start/stop
  - Status retrieval
  - Image pulling

### 5. Service Discovery (internal/discovery)
- **Purpose**: Service discovery based on memberlist
- **Technology**: Custom implementation with synchronization via memberlist broadcast
- **Functions**:
  - Local service registration on each node
  - Synchronization via memberlist broadcast
  - Service lookup with automatic load balancing
  - Endpoint health checking
  - Service change tracking

### 6. DNS Server (internal/dns)
- **Purpose**: DNS resolution for services and external domains
- **Functions**:
  - Service resolution via DNS names (format: `service.namespace.cluster.local`)
  - A and SRV record support
  - Forwarding external DNS queries to upstream DNS servers
  - DNS whitelist for external domain control
  - CNAME record support in whitelist
- **Technology**: miekg/dns

### 7. Ingress Controller (internal/ingress)
- **Purpose**: HTTP/HTTPS traffic routing
- **Functions**:
  - Ingress rule processing
  - Reverse proxy to services
  - Load balancing

### 8. API Server (internal/api)
- **Purpose**: REST API for cluster management
- **Endpoints**:
  - `POST /api/v1/manifests` - Apply manifest
  - `DELETE /api/v1/manifests/:namespace/:name` - Delete resource
  - `GET /api/v1/pods` - List pods
  - `GET /api/v1/deployments` - List deployments
  - `GET /api/v1/services` - List services
  - `GET /api/v1/services/:namespace/:name/endpoints` - Service endpoints
  - `GET /api/v1/services/:namespace/:name/addresses` - Service addresses
  - `GET /api/v1/nodes` - List nodes
  - `GET /api/v1/dns/whitelist` - Get DNS whitelist
  - `PUT /api/v1/dns/whitelist` - Set DNS whitelist
  - `POST /api/v1/dns/whitelist/hosts` - Add host to whitelist
  - `DELETE /api/v1/dns/whitelist/hosts/:host` - Remove host from whitelist
  - `POST /api/v1/tokens` - Generate API token
  - `GET /api/v1/tokens` - List API tokens
  - `DELETE /api/v1/tokens/:token` - Revoke API token

## Workflow

### Deployment Deployment

1. User sends Kubernetes manifest to API
2. Parser parses manifest and extracts Deployment information
3. Scheduler determines which node to place each pod on
4. Podman Client creates container on the appropriate node
5. Container starts
6. Status is updated in Scheduler

### Service Deployment

1. Parser extracts Service information
2. Service is registered locally via Service Discovery
3. All pods matching the selector are registered as endpoints
4. Service information is synchronized between nodes via memberlist
5. DNS server automatically resolves service via DNS name (`service.namespace.cluster.local`)

### Ingress Deployment

1. Parser extracts Ingress information
2. Ingress Controller adds routing rules
3. Requests to the specified host/path are proxied to the corresponding service

## Load Balancing

- **DNS level**: Multiple A records for each service (one per endpoint)
- **Service level**: Round-robin between service pods via custom service discovery
- **Ingress level**: Round-robin between services via Ingress Controller with local optimization

## Persistence

- Pods can be bound to a specific node via `nodeSelector`
- This allows using local volumes for persistent data

## Scaling

- All nodes are equal (peer-to-peer)
- A new node can join the cluster via the `--join` parameter
- Scheduler automatically distributes new pods across all nodes
- DNS server automatically synchronizes service information between nodes

## DNS Resolution

- **Internal services**: Resolved via DNS names in format `service.namespace.cluster.local`
- **External domains**: Forwarding to upstream DNS servers (default: 8.8.8.8, 8.8.4.4)
- **Whitelist**: Ability to restrict external DNS queries with a domain whitelist
- **CNAME support**: CNAME record validation in whitelist for security
