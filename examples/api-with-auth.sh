#!/bin/bash

# Example: Using Podman Swarm API with authentication

# Set your API token (get from agent logs or generate via API)
API_TOKEN="your-api-token-here"
API_URL="http://localhost:8080"

# Helper function for authenticated requests
api_get() {
    curl -s -H "Authorization: Bearer $API_TOKEN" "$API_URL$1"
}

api_post() {
    curl -s -H "Authorization: Bearer $API_TOKEN" \
         -H "Content-Type: application/json" \
         -X POST "$API_URL$1" \
         -d "$2"
}

api_put() {
    curl -s -H "Authorization: Bearer $API_TOKEN" \
         -H "Content-Type: application/json" \
         -X PUT "$API_URL$1" \
         -d "$2"
}

api_delete() {
    curl -s -H "Authorization: Bearer $API_TOKEN" \
         -X DELETE "$API_URL$1"
}

# Examples:

# 1. Check cluster health (no auth required)
echo "=== Health Check ==="
curl -s "$API_URL/api/v1/health" | jq

# 2. List nodes (requires auth)
echo -e "\n=== List Nodes ==="
api_get "/api/v1/nodes" | jq

# 3. List pods
echo -e "\n=== List Pods ==="
api_get "/api/v1/pods" | jq

# 4. Get service endpoints
echo -e "\n=== Service Endpoints ==="
api_get "/api/v1/services/default/nginx-service/endpoints" | jq

# 5. Deploy manifest
echo -e "\n=== Deploy Manifest ==="
curl -s -H "Authorization: Bearer $API_TOKEN" \
     -X POST "$API_URL/api/v1/manifests" \
     -H "Content-Type: application/yaml" \
     --data-binary @../examples/deployment.yaml | jq

# 6. Configure DNS whitelist
echo -e "\n=== Configure DNS Whitelist ==="
api_put "/api/v1/dns/whitelist" '{
  "enabled": true,
  "hosts": ["google.com", "github.com", "docker.io"]
}' | jq

# 7. Generate new API token
echo -e "\n=== Generate New Token ==="
api_post "/api/v1/tokens" '{
  "name": "ci-token",
  "expires_in": 86400
}' | jq

# 8. List API tokens
echo -e "\n=== List Tokens ==="
api_get "/api/v1/tokens" | jq

echo -e "\nDone!"
