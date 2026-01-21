# State Persistence and Recovery

Podman Swarm implements a robust state persistence system that ensures cluster state survives node restarts and failures.

## Overview

The storage system provides:
- **Automatic persistence** of all cluster resources (Deployments, Services, Ingresses, Pods)
- **Automatic recovery** of deployments after node restart
- **Periodic backups** for disaster recovery
- **State synchronization** between nodes for consistency
- **Atomic operations** to prevent data corruption

## Architecture

### Storage Layer

The storage layer (`internal/storage`) uses JSON-based file storage:

```
/var/lib/podman-swarm/
├── state.json                           # Current cluster state
└── state-backup-20260121-120000.json    # Hourly backups
```

### State Structure

```json
{
  "deployments": {
    "default/nginx": {
      "name": "nginx",
      "namespace": "default",
      "desired_replicas": 3,
      "pods": [/* ... */]
    }
  },
  "services": {
    "default/nginx-service": {/* ... */}
  },
  "ingresses": {
    "default/nginx-ingress": {/* ... */}
  },
  "pods": {
    "default/nginx-0": {/* ... */}
  },
  "last_modified": "2026-01-21T12:00:00Z",
  "version": 1
}
```

## How It Works

### 1. Persistence on Changes

When a manifest is applied:
1. Resource is parsed and validated
2. Resource is stored in memory cache
3. **Immediately persisted to disk** (atomic write)
4. Pod creation/scheduling proceeds

```go
// Automatic persistence
a.deployments[key] = dep
a.storage.SaveDeployment(dep)  // Persisted to disk
```

### 2. Atomic File Operations

To prevent corruption:
1. State is written to temporary file (`state.json.tmp`)
2. Temporary file is atomically renamed to `state.json`
3. Old state is overwritten only if write succeeds

### 3. Automatic Recovery on Restart

When agent starts:
1. Storage loads existing `state.json`
2. State is loaded into memory cache
3. **Recovery process begins** (after 5s delay for cluster stabilization):
   - Deployments are recreated
   - Pods are rescheduled and started
   - Services are re-registered in discovery
   - Ingresses are restored

```bash
# On restart:
INFO Storage initialized successfully
INFO Loaded state: 2 deployments, 3 services, 1 ingresses
INFO Starting deployment recovery from persistent storage...
INFO Recovering deployment: default/nginx
INFO Recovered pod nginx-0 on node node-1
INFO Deployment recovery completed: 2 recovered, 0 failed
```

### 4. State Synchronization

For peer-to-peer consistency:

- **Periodic broadcast** (every 30 seconds): Node broadcasts its state to all peers
- **Merge strategy**: Incoming state is merged if it's newer (based on timestamp)
- **Conflict resolution**: Newest modification wins

This ensures all nodes eventually have consistent view of cluster state.

### 5. Periodic Backups

Automated backup system:
- **Frequency**: Every 1 hour
- **Format**: Timestamped JSON files
- **Location**: Same data directory
- **Retention**: Manual cleanup (automatic retention policy planned)

Example backup file:
```
/var/lib/podman-swarm/state-backup-20260121-120000.json
```

## State Recovery Process

### On Node Restart

1. **Load state** from `state.json`
2. **Wait for cluster** to stabilize (5 seconds)
3. **Recover deployments**:
   - For each deployment:
     - Clear old pod references
     - Reschedule pods according to desired replicas
     - Create and start pods on this node (if scheduled here)
     - Update deployment with new pod references
4. **Recover services**:
   - Re-register service endpoints for matching pods
5. **Recover ingresses**:
   - Restore ingress rules in ingress controller

### Example Recovery Log

```
Starting Podman Swarm agent: node-1
Storage initialized successfully
Loaded state: 2 deployments, 3 services, 1 ingresses, 4 pods
Cluster initialized with 3 nodes
Starting deployment recovery from persistent storage...
Recovering deployment: default/nginx
Recovering pod nginx-0 on node node-1
Created container nginx-0 (ID: abc123...)
Recovered pod nginx-0 on node node-1
Recovering deployment: default/api
Deployment recovery completed: 2 recovered, 0 failed
Recovering service: default/nginx-service
Recovering service: default/api-service
```

## Data Safety

### Preventing Data Loss

1. **Atomic writes**: State is never partially written
2. **Temporary files**: Changes go to `.tmp` file first
3. **Rename operation**: Only atomic rename on success
4. **Hourly backups**: Multiple recovery points
5. **Replication**: State synchronized across all nodes

### File Permissions

All state files have restricted permissions:
- **state.json**: `0640` (owner read/write, group read)
- **backups**: `0640`
- **encryption.key**: `0600` (owner read/write only)

## Configuration

State persistence is configured via:

```bash
./podman-swarm-agent \
  --data-dir=/var/lib/podman-swarm  # Where to store state
```

Default: `/var/lib/podman-swarm`

## Monitoring State

### Check Current State

```bash
# View state file directly
cat /var/lib/podman-swarm/state.json | jq

# Via API (after recovery)
curl http://localhost:8080/api/v1/deployments
curl http://localhost:8080/api/v1/services
```

### List Backups

```bash
ls -lh /var/lib/podman-swarm/state-backup-*.json
```

### Restore from Backup

```bash
# Stop agent
systemctl stop podman-swarm-agent

# Restore from backup
cp /var/lib/podman-swarm/state-backup-20260121-120000.json \
   /var/lib/podman-swarm/state.json

# Start agent (will recover from restored state)
systemctl start podman-swarm-agent
```

## Limitations

Current limitations of the state persistence system:

1. **No state versioning** - Schema changes require manual migration
2. **No incremental backups** - Full state written each time
3. **No automatic retention** - Backup cleanup is manual
4. **Simple conflict resolution** - Newest timestamp wins (no CRDT)
5. **No state encryption** - State file stored in plain JSON

## Planned Improvements

See [TODO.md](TODO.md) for planned enhancements:

- State versioning and schema migration
- State reconciliation after network partitions
- Encrypted state files
- Automatic backup retention policy
- Incremental state changes
- State compression

## Best Practices

1. **Regular backups**: Monitor backup creation
2. **Secure data directory**: Protect `/var/lib/podman-swarm`
3. **Monitor disk space**: State files can grow with cluster size
4. **Test recovery**: Periodically test recovery process
5. **Multiple nodes**: Run at least 3 nodes for redundancy

## Troubleshooting

### State not persisting

Check:
- Data directory is writable
- Sufficient disk space
- File permissions (0640 for state.json)
- Check agent logs for "Failed to persist" warnings

### Recovery fails

Check:
- state.json is valid JSON
- Podman is accessible
- Network connectivity to other nodes
- Check logs for specific pod creation errors

### State inconsistent across nodes

- Wait for next sync cycle (30 seconds)
- Check node connectivity
- Verify encryption keys match
- Check for network partitions

## See Also

- [AGENTS.md](AGENTS.md) - Agent architecture and components
- [ARCHITECTURE.md](ARCHITECTURE.md) - Overall system architecture
- [SECURITY.md](SECURITY.md) - Security features
