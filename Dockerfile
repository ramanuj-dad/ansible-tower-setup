FROM python:3.11-slim

# Install kubectl
RUN apt-get update && \
    apt-get install -y curl && \
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" && \
    chmod +x kubectl && \
    mv kubectl /usr/local/bin/ && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy deployment script
COPY deploy_awx.py /app/deploy_awx.py

# Make script executable
RUN chmod +x /app/deploy_awx.py

# Copy entry script
COPY <<EOF /app/entrypoint.sh
#!/bin/bash
set -e

# Check if kubeconfig exists
if [ ! -f /kubeconfig ]; then
  echo "ERROR: /kubeconfig file not found!"
  echo "Please mount a valid kubeconfig file to /kubeconfig"
  exit 1
fi

# Verify kubeconfig is not empty
if [ ! -s /kubeconfig ]; then
  echo "ERROR: /kubeconfig file is empty!"
  exit 1
fi

# Validate kubeconfig can connect
export KUBECONFIG=/kubeconfig
echo "Testing Kubernetes connection..."
kubectl cluster-info || {
  echo "ERROR: Failed to connect to Kubernetes cluster with provided kubeconfig"
  echo "Please check if the kubeconfig file is valid and the cluster is accessible"
  exit 1
}

# Run the deployment script
exec python3 /app/deploy_awx.py
EOF

# Make entry script executable
RUN chmod +x /app/entrypoint.sh

# Set entrypoint
ENTRYPOINT ["/app/entrypoint.sh"]
