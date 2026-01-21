#!/bin/bash

# Example: Using psctl to manage Podman Swarm cluster

# Configuration
API_SERVER="http://localhost:8080"
API_TOKEN=""  # Set if authentication is enabled

# Configure psctl (one-time setup)
echo "=== Configuring psctl ==="
psctl config set-server "$API_SERVER"

if [ -n "$API_TOKEN" ]; then
    psctl config set-token "$API_TOKEN"
fi

psctl config view

# Example: Deploy application
echo -e "\n=== Deploying Application ==="

# Create a simple deployment manifest
cat > /tmp/nginx-deployment.yaml <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
  namespace: default
spec:
  replicas: 2
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
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: nginx-ingress
  namespace: default
spec:
  rules:
  - host: nginx.local
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: nginx-service
            port:
              number: 80
EOF

# Apply the manifest
psctl apply -f /tmp/nginx-deployment.yaml

# Wait for pods to start
echo -e "\n=== Waiting for pods to start ==="
sleep 5

# List resources
echo -e "\n=== Listing Resources ==="
echo "Deployments:"
psctl get deployments

echo -e "\nPods:"
psctl get pods

echo -e "\nServices:"
psctl get services

echo -e "\nNodes:"
psctl get nodes

# Describe deployment
echo -e "\n=== Describing Deployment ==="
psctl describe deployment nginx

# Get detailed information
echo -e "\n=== Detailed Pod Information ==="
psctl get pods -o json | jq '.pods[0]'

# Show labels
echo -e "\n=== Pods with Labels ==="
psctl get pods --show-labels

# Service endpoints
echo -e "\n=== Service Endpoints ==="
psctl get service nginx-service -o json | jq

# Get logs (placeholder)
echo -e "\n=== Pod Logs ==="
POD_NAME=$(psctl get pods -o json | jq -r '.pods[0].name // "nginx-0"')
psctl logs "$POD_NAME" 2>/dev/null || echo "Log streaming not yet implemented"

# Cleanup
echo -e "\n=== Cleanup ==="
read -p "Do you want to delete the deployment? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    psctl delete deployment nginx
    psctl delete service nginx-service
    psctl delete ingress nginx-ingress
    echo "Resources deleted"
fi

# Remove temporary file
rm -f /tmp/nginx-deployment.yaml

echo -e "\n=== Done! ==="
