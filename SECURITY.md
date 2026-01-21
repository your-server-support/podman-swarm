# Podman Swarm Security

## Communication Encryption

Podman Swarm supports encryption of communication between nodes at two levels:

### 1. Message-level Encryption

All messages between nodes are encrypted using AES-256-GCM. The encryption key can be specified via the `--encryption-key` parameter or automatically generated for the first node.

**Usage:**
```bash
# First node (generates key automatically)
./podman-swarm-agent --node-name=node1

# Other nodes (use the same key)
./podman-swarm-agent --node-name=node2 \
  --join=node1:7946 \
  --join-token=<TOKEN> \
  --encryption-key=<KEY>
```

### 2. TLS Encryption (Transport-level)

For additional protection, TLS certificates can be used:

```bash
./podman-swarm-agent \
  --node-name=node1 \
  --tls-cert=/path/to/cert.pem \
  --tls-key=/path/to/key.pem \
  --tls-ca=/path/to/ca.pem
```

## Join Token

The token system works similarly to Docker Swarm:

### Token Generation

The first node automatically generates a join token at startup:

```bash
./podman-swarm-agent --node-name=node1
# Output: Generated join token: <TOKEN>
```

### Using Token to Join

```bash
./podman-swarm-agent \
  --node-name=node2 \
  --join=node1:7946 \
  --join-token=<TOKEN>
```

### Token Validation

- Token is validated when attempting to join the cluster
- Invalid token blocks node joining
- Tokens can be revoked via API

## DNS Whitelist

Podman Swarm supports a whitelist of external domains for DNS resolution control:

### Whitelist Configuration

```bash
# Via API
curl -X PUT http://localhost:8080/api/v1/dns/whitelist \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": true,
    "hosts": ["google.com", "github.com", "docker.io"]
  }'
```

### Features

- **By default**: Whitelist is disabled, all domains are allowed
- **Subdomain support**: If `example.com` is allowed, then `api.example.com` is also allowed
- **CNAME validation**: All CNAME targets in DNS responses are checked
- **Blocking**: Queries to disallowed domains return `RcodeRefused`

### Usage Example

```bash
# Get current configuration
curl http://localhost:8080/api/v1/dns/whitelist

# Add host
curl -X POST http://localhost:8080/api/v1/dns/whitelist/hosts \
  -H "Content-Type: application/json" \
  -d '{"host": "example.com"}'

# Remove host
curl -X DELETE http://localhost:8080/api/v1/dns/whitelist/hosts/example.com
```

## Security Recommendations

1. **Use strong encryption keys:**
   ```bash
   # Generate random key
   openssl rand -base64 32
   ```

2. **Store keys securely:**
   - Don't commit keys to git
   - Use secret managers (HashiCorp Vault, AWS Secrets Manager, etc.)
   - Restrict access to key files (chmod 600)

3. **Use TLS certificates:**
   - Use certificates from trusted CAs
   - Regularly update certificates
   - Use separate certificates for each node

4. **Restrict network access:**
   - Use firewall to restrict access to cluster port (7946)
   - Use VPN or private networks for inter-node communication

5. **Regularly rotate tokens:**
   - Generate new tokens periodically
   - Revoke old tokens via API

6. **Use DNS whitelist:**
   - Enable whitelist to restrict external DNS queries
   - Add only necessary domains
   - Regularly review the list of allowed domains

## Production Configuration Example

```bash
# node1 (first node)
./podman-swarm-agent \
  --node-name=node1 \
  --bind-addr=10.0.1.1:7946 \
  --encryption-key=$(cat /etc/podman-swarm/encryption.key) \
  --tls-cert=/etc/podman-swarm/certs/node1.crt \
  --tls-key=/etc/podman-swarm/certs/node1.key \
  --tls-ca=/etc/podman-swarm/certs/ca.crt

# node2 (joining)
./podman-swarm-agent \
  --node-name=node2 \
  --bind-addr=10.0.1.2:7946 \
  --join=node1:7946 \
  --join-token=$(cat /etc/podman-swarm/join.token) \
  --encryption-key=$(cat /etc/podman-swarm/encryption.key) \
  --tls-cert=/etc/podman-swarm/certs/node2.crt \
  --tls-key=/etc/podman-swarm/certs/node2.key \
  --tls-ca=/etc/podman-swarm/certs/ca.crt
```
