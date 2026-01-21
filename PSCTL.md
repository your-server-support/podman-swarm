# psctl - Podman Swarm CLI

`psctl` is a command-line interface tool for managing Podman Swarm clusters, inspired by `kubectl`.

## Installation

### Build from Source

```bash
# Build psctl
make build-psctl

# Or build both agent and psctl
make build-all

# Install to $GOPATH/bin
make install
```

### Binary Location

After building, the binary will be available at:
- `./psctl` (in project root)
- `$GOPATH/bin/psctl` (after `make install`)

## Configuration

`psctl` can be configured in three ways (in order of precedence):

1. **Command-line flags**: `--server`, `--token`, `--namespace`
2. **Environment variables**: `PSCTL_SERVER`, `PSCTL_TOKEN`
3. **Config file**: `~/.psctl/config`

### Setting up Configuration

```bash
# Set API server URL
psctl config set-server http://localhost:8080

# Set authentication token
psctl config set-token <your-token>

# View current configuration
psctl config view

# Get config file location
psctl config get-location
```

### Config File Format

`~/.psctl/config`:
```yaml
server: http://localhost:8080
token: your-api-token-here
```

## Usage

### Basic Commands

```bash
# Apply a manifest
psctl apply -f deployment.yaml

# Get resources
psctl get pods
psctl get deployments
psctl get services
psctl get nodes

# Get specific resource
psctl get pod nginx-0
psctl get deployment nginx

# Get resources in specific namespace
psctl get pods -n production

# Delete resources
psctl delete deployment nginx
psctl delete service nginx-service

# Describe resource
psctl describe pod nginx-0
psctl describe deployment nginx

# Get logs (placeholder)
psctl logs nginx-0
```

### Output Formats

```bash
# Default table output
psctl get pods

# JSON output
psctl get pods -o json

# Show labels
psctl get pods --show-labels
```

### Authentication

If API authentication is enabled on the server, provide a token:

```bash
# Via flag
psctl get pods --token <your-token>

# Via config
psctl config set-token <your-token>
psctl get pods

# Via environment variable
export PSCTL_TOKEN=<your-token>
psctl get pods
```

## Commands Reference

### apply

Apply a configuration to resources from a file.

```bash
psctl apply -f <filename>
psctl apply -f deployment.yaml
cat deployment.yaml | psctl apply -f -
```

**Flags:**
- `-f, --filename`: Filename or `-` for stdin (required)
- `-n, --namespace`: Namespace (default: "default")
- `--server`: API server URL
- `--token`: Authentication token

### get

Display one or many resources.

```bash
psctl get <resource> [name]
psctl get pods
psctl get pod nginx-0
psctl get deployments -n production
```

**Resources:**
- `pods`, `pod`, `po`: List or get pods
- `deployments`, `deployment`, `deploy`: List or get deployments
- `services`, `service`, `svc`: List or get services
- `nodes`, `node`: List nodes

**Flags:**
- `-o, --output`: Output format (json|yaml)
- `--show-labels`: Show resource labels
- `-n, --namespace`: Namespace (default: "default")
- `--server`: API server URL
- `--token`: Authentication token

### delete

Delete a resource by name.

```bash
psctl delete <resource> <name>
psctl delete deployment nginx
psctl delete service nginx-service -n production
```

**Flags:**
- `-n, --namespace`: Namespace (default: "default")
- `--server`: API server URL
- `--token`: Authentication token

### describe

Show detailed information about a specific resource.

```bash
psctl describe <resource> <name>
psctl describe pod nginx-0
psctl describe deployment nginx
psctl describe node node-1
```

**Flags:**
- `-n, --namespace`: Namespace (default: "default")
- `--server`: API server URL
- `--token`: Authentication token

### logs

Print the logs for a pod (placeholder - not fully implemented).

```bash
psctl logs <pod-name>
psctl logs nginx-0
```

**Flags:**
- `-f, --follow`: Follow log output (not implemented)
- `--tail`: Number of lines to show (not implemented)
- `-n, --namespace`: Namespace (default: "default")
- `--server`: API server URL
- `--token`: Authentication token

### config

Manage psctl configuration.

```bash
# Set server URL
psctl config set-server http://localhost:8080

# Set authentication token
psctl config set-token <token>

# View configuration
psctl config view

# Get config file location
psctl config get-location
```

**Subcommands:**
- `set-server <url>`: Set API server URL
- `set-token <token>`: Set authentication token
- `view`: View current configuration
- `get-location`: Show config file path

## Examples

### Deploy an Application

```bash
# Create deployment.yaml
cat > deployment.yaml <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
  namespace: default
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:latest
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: nginx-service
  namespace: default
spec:
  selector:
    app: nginx
  ports:
  - port: 80
    targetPort: 80
EOF

# Apply manifest
psctl apply -f deployment.yaml

# Check deployment
psctl get deployments
psctl get pods
psctl get services

# Describe resources
psctl describe deployment nginx
psctl describe service nginx-service
```

### Manage Resources Across Namespaces

```bash
# List pods in default namespace
psctl get pods

# List pods in production namespace
psctl get pods -n production

# Delete deployment in staging
psctl delete deployment myapp -n staging
```

### Use with Authentication

```bash
# Configure once
psctl config set-server http://192.168.1.100:8080
psctl config set-token abc123xyz...

# Use without specifying credentials
psctl get pods
psctl apply -f app.yaml
psctl get nodes
```

### JSON Output and Processing

```bash
# Get pods as JSON
psctl get pods -o json

# Process with jq
psctl get pods -o json | jq '.pods[] | select(.status=="running")'

# Count running pods
psctl get pods -o json | jq '.pods | length'
```

## Comparison with kubectl

`psctl` is designed to be similar to `kubectl` for Kubernetes:

| kubectl | psctl | Description |
|---------|-------|-------------|
| `kubectl apply -f file.yaml` | `psctl apply -f file.yaml` | Apply manifest |
| `kubectl get pods` | `psctl get pods` | List pods |
| `kubectl get pod nginx-0` | `psctl get pod nginx-0` | Get specific pod |
| `kubectl delete deployment nginx` | `psctl delete deployment nginx` | Delete deployment |
| `kubectl describe pod nginx-0` | `psctl describe pod nginx-0` | Describe pod |
| `kubectl logs nginx-0` | `psctl logs nginx-0` | Get logs |
| `kubectl get pods -n prod` | `psctl get pods -n prod` | Namespace scoped |
| `kubectl get pods -o json` | `psctl get pods -o json` | JSON output |

## Troubleshooting

### Connection Refused

```bash
# Check server URL
psctl config view

# Test connection
curl http://localhost:8080/api/v1/health

# Update server URL
psctl config set-server http://correct-url:8080
```

### Authentication Error (401)

```bash
# Check if token is set
psctl config view

# Get new token from agent logs or API
psctl config set-token <new-token>
```

### Resource Not Found

```bash
# Check namespace
psctl get pods -n default
psctl get pods -n production

# List all resources
psctl get pods
psctl get deployments
psctl get services
```

## Development

### Adding New Commands

1. Create a new file in `internal/psctl/` (e.g., `mycommand.go`)
2. Implement `NewMyCommand()` function returning `*cobra.Command`
3. Add the command in `cmd/psctl/main.go`: `rootCmd.AddCommand(psctl.NewMyCommand(...))`

### Testing

```bash
# Build
make build-psctl

# Run
./psctl --help
./psctl apply --help
./psctl get pods
```

## See Also

- [README.md](README.md) - Main project documentation
- [AGENTS.md](AGENTS.md) - Agent documentation
- [ARCHITECTURE.md](ARCHITECTURE.md) - Architecture overview
- [SECURITY.md](SECURITY.md) - Security features
