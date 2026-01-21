# TODO - Podman Swarm Development Roadmap

> **Project Focus**: Podman Swarm is a Docker Swarm replacement for small to medium-sized clusters with Kubernetes API compatibility. It's not a Kubernetes clone, but a lightweight orchestrator that speaks Kubernetes manifests.
> 
> **Core Principles**:
> - **True peer-to-peer** - All nodes equal, no master nodes
> - **Security first** - Minimal privileges, rootless, secure by default
> - **Simplicity** - Easy to deploy and manage

## ✅ Already Implemented

### Core Infrastructure
- [x] **Peer-to-peer Cluster** - HashiCorp Memberlist-based cluster management
- [x] **Service Discovery** - Custom implementation with memberlist synchronization
- [x] **DNS Server** - Built-in DNS server for service resolution
  - [x] A and SRV record support
  - [x] Kubernetes-compatible DNS names (service.namespace.cluster.local)
  - [x] Upstream DNS forwarding
  - [x] Configurable cluster domain
- [x] **DNS Whitelist** - External domain resolution control
  - [x] Whitelist management via API
  - [x] Subdomain matching
  - [x] CNAME validation
- [x] **Ingress Controller** - HTTP/HTTPS traffic routing
  - [x] Round-robin load balancing
  - [x] Local optimization
  - [x] Health-aware routing

### Security
- [x] **Message Encryption** - AES-256-GCM encryption for cluster communication
- [x] **Join Token System** - Docker Swarm-like node authentication
- [x] **TLS Support** - Optional transport-level encryption
- [x] **API Authentication** - Bearer token authentication for API endpoints
  - [x] Token generation and management
  - [x] Token expiration support
  - [x] Automatic token cleanup

### Kubernetes Compatibility
- [x] **Manifest Parser** - Parse Kubernetes YAML manifests
- [x] **Deployment** - Deployment resource support
- [x] **Service** - Service resource support (ClusterIP)
- [x] **Ingress** - Ingress resource support
- [x] **Node Selector** - Basic node affinity support

### Scheduler
- [x] **Basic Scheduler** - Pod distribution across nodes
- [x] **Node Selection** - Random and node selector strategies

### API Server
- [x] **REST API** - Full REST API for cluster management
  - [x] Manifest application endpoint
  - [x] Pod/Deployment/Service/Node listing
  - [x] Service endpoint discovery
  - [x] DNS whitelist management
  - [x] Token management endpoints
- [x] **CORS Support** - Cross-origin resource sharing
- [x] **Health Check Endpoint** - Cluster health monitoring

### CLI Tool (psctl)
- [x] **kubectl-like Interface** - Familiar command structure
- [x] **apply** - Apply manifests from files or stdin
- [x] **get** - List and get resources (pods, deployments, services, nodes)
- [x] **delete** - Delete resources
- [x] **describe** - Show detailed resource information
- [x] **config** - Configuration management
- [x] **Output Formats** - JSON and table output
- [x] **Authentication** - Token-based authentication support

### Podman Integration
- [x] **Podman v4 API** - Full Podman v4 bindings support
- [x] **Container Management** - Create, start, stop, remove containers
- [x] **Image Management** - Pull and manage container images
- [x] **DNS Configuration** - Automatic DNS server configuration for containers
- [x] **Port Mapping** - Container port mapping support
- [x] **Volume Mounting** - Basic volume mounting support

### State Management
- [x] **Persistent Storage** - JSON-based file storage
  - [x] Deployments, Services, Ingresses persistence
  - [x] Pod state tracking
  - [x] Atomic file operations
  - [x] Unit tests (13 tests covering CRUD, concurrency, backups)
- [x] **State Recovery** - Automatic recovery on restart
  - [x] Deployment recreation
  - [x] Service re-registration
  - [x] Ingress restoration
- [x] **State Synchronization** - Peer-to-peer state sync
  - [x] Periodic state broadcasting (every 30s)
  - [x] State merge on receive
  - [x] Conflict resolution (newest wins)
- [x] **Backup System** - Automated backups
  - [x] Periodic backups (hourly)
  - [x] Timestamped backup files

### Documentation
- [x] **Comprehensive Documentation** - Complete documentation suite
  - [x] README (EN/UK)
  - [x] ARCHITECTURE (EN/UK)
  - [x] SECURITY (EN/UK)
  - [x] ROUTING (EN/UK)
  - [x] SERVICE_COMMUNICATION (EN/UK)
  - [x] AGENTS.md - Agent documentation
  - [x] PSCTL (EN/UK) - CLI documentation
  - [x] TODO (EN/UK) - Development roadmap
  - [x] CONTRIBUTING.md - Contribution guidelines

### DevOps
- [x] **Makefile** - Build automation
- [x] **Dockerfile** - Container image for agent
- [x] **Docker Compose** - Multi-node testing setup

---

## 🔴 Critical / High Priority

### Core Functionality
- [ ] **Log Streaming Implementation**
  - [ ] Implement real-time log streaming in API
  - [ ] Complete `psctl logs` command with `-f` (follow) support
  - [ ] Add log filtering and search capabilities

- [ ] **Persistent Storage Support**
  - [ ] Implement PersistentVolume and PersistentVolumeClaim support
  - [ ] Add volume management in scheduler
  - [ ] Implement volume binding to specific nodes

- [ ] **Pod Lifecycle Management**
  - [ ] Implement pod restart policies (Always, OnFailure, Never)
  - [ ] Add pod health checks (liveness and readiness probes)
  - [ ] Implement graceful pod termination

- [x] **State Persistence** ✅
  - [x] Implement JSON-based file storage for cluster state
  - [x] Add automatic state recovery after node restart
  - [x] Implement periodic state backup (hourly)
  - [x] Add state synchronization between nodes
  - [x] Atomic file writes for data safety
  - [ ] Add state reconciliation after network partitions
  - [ ] Implement state versioning and migration

### Security Enhancements
- [ ] **Rootless Mode Improvements**
  - [ ] Full rootless operation documentation
  - [ ] Rootless port mapping below 1024 (using slirp4netns)
  - [ ] Rootless volume management best practices
  - [ ] SELinux/AppArmor integration for rootless

- [ ] **RBAC (Role-Based Access Control)**
  - [ ] Implement user and service account management
  - [ ] Add role and role binding support
  - [ ] Implement namespace isolation
  - [ ] Audit logging for security events

- [ ] **Network Policies**
  - [ ] Add network policy support
  - [ ] Implement pod-to-pod network restrictions
  - [ ] Add egress/ingress rules
  - [ ] DNS-based network policies

- [ ] **Principle of Least Privilege**
  - [ ] Document minimal required permissions for each component
  - [ ] Add capability dropping for containers
  - [ ] Implement read-only root filesystem support
  - [ ] Add security context constraints

### Testing
- [ ] **Unit Tests**
  - [ ] Add unit tests for core components (>80% coverage)
  - [ ] Add tests for security components
  - [ ] Add tests for DNS and networking

- [ ] **Integration Tests**
  - [ ] End-to-end deployment tests
  - [ ] Multi-node cluster tests
  - [ ] Failover and recovery tests

## 🟡 Medium Priority

### Features
- [ ] **ConfigMap and Secret Support**
  - [ ] Implement ConfigMap resource type
  - [ ] Implement Secret resource type
  - [ ] Add mounting to pods

- [ ] **StatefulSet Support**
  - [ ] Implement StatefulSet controller
  - [ ] Add ordered pod deployment
  - [ ] Implement stable network identities

- [ ] **DaemonSet Support**
  - [ ] Implement DaemonSet controller
  - [ ] Ensure one pod per node
  - [ ] Add node selector support

- [ ] **Horizontal Pod Autoscaler**
  - [ ] Implement metrics collection
  - [ ] Add auto-scaling based on CPU/memory
  - [ ] Support custom metrics

### Monitoring and Observability
- [ ] **Metrics Endpoint**
  - [ ] Add Prometheus-compatible metrics endpoint
  - [ ] Expose cluster, node, and pod metrics
  - [ ] Add custom metrics support

- [ ] **Event System**
  - [ ] Implement event recording
  - [ ] Add event API endpoint
  - [ ] Implement event retention policy

### CLI Improvements
- [ ] **psctl Enhancements**
  - [ ] Add `psctl exec` command for executing commands in pods
  - [ ] Implement `psctl port-forward` for port forwarding
  - [ ] Add `psctl top` for resource usage
  - [ ] Implement `psctl scale` for scaling deployments
  - [ ] Add shell completion (bash, zsh, fish)

### Networking
- [ ] **CNI Plugin Support**
  - [ ] Add support for CNI plugins
  - [ ] Implement overlay networking options
  - [ ] Add network performance improvements

- [ ] **Load Balancer Integration**
  - [ ] Add LoadBalancer service type support
  - [ ] Implement external load balancer integration
  - [ ] Add MetalLB-like functionality

## 🟢 Low Priority / Nice to Have

### Documentation
- [ ] **User Guides**
  - [ ] Write getting started guide
  - [ ] Add troubleshooting guide
  - [ ] Create video tutorials
  - [ ] Add best practices guide

- [ ] **API Documentation**
  - [ ] Generate OpenAPI/Swagger documentation
  - [ ] Add API examples for all endpoints
  - [ ] Document authentication flows

### Developer Experience
- [ ] **Development Tools**
  - [ ] Add Makefile targets for common tasks
  - [ ] Create development environment setup script
  - [ ] Add hot-reload for development

- [ ] **CI/CD**
  - [ ] Set up GitHub Actions for automated testing
  - [ ] Add automated builds and releases
  - [ ] Implement semantic versioning

### Additional Features
- [ ] **Web UI**
  - [ ] Create web-based dashboard
  - [ ] Add cluster visualization
  - [ ] Implement resource management UI

- [ ] **Backup and Restore**
  - [ ] Implement cluster backup functionality
  - [ ] Add restore from backup
  - [ ] Implement disaster recovery procedures

- [ ] **Multi-Architecture Support**
  - [ ] Add ARM64 support
  - [ ] Test on different Linux distributions
  - [ ] Add Windows and macOS support (via Podman)

### Performance
- [ ] **Optimization**
  - [ ] Profile and optimize hot paths
  - [ ] Reduce memory footprint
  - [ ] Improve startup time
  - [ ] Optimize service discovery performance

- [ ] **Scalability for Target Range**
  - [ ] Test with 10-20 nodes (typical use case)
  - [ ] Test with 50+ nodes (upper limit)
  - [ ] Test with 500+ pods
  - [ ] Optimize for small to medium cluster sizes

## 🔵 Future / Research

- [ ] **Service Mesh Integration**
  - [ ] Research lightweight service mesh options
  - [ ] Add mTLS between services
  - [ ] Implement basic traffic management

- [ ] **Advanced Scheduling**
  - [ ] Add pod affinity/anti-affinity
  - [ ] Implement resource quotas and limits
  - [ ] Add priority classes for pods

## 📝 Known Issues

- [ ] CGO dependencies (btrfs, gpgme, devicemapper) require system libraries
- [ ] TLS transport for memberlist needs custom implementation
- [ ] Log streaming API not yet implemented
- [ ] No graceful handling of node failures
- [ ] Limited error recovery mechanisms

## 🎯 Milestones

### v0.2.0 (Q1 2026)
- Log streaming implementation
- Basic persistent storage support
- Comprehensive unit tests
- RBAC implementation

### v0.3.0 (Q2 2026)
- StatefulSet and DaemonSet support
- ConfigMap and Secret support
- Metrics and monitoring
- Network policies

### v1.0.0 (Q3-Q4 2026)
- Production-ready stability
- Complete Kubernetes API compatibility
- Full documentation
- Performance optimization
- Security audit

## 🤝 Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on how to contribute to these items.

Prioritization may change based on community feedback and requirements.

---

Last Updated: 2026-01-21
