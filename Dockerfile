# Use the official Ubuntu image as base
FROM ubuntu:22.04

# Install required packages
RUN apt-get update && apt-get install -y \
    curl \
    git \
    && rm -rf /var/lib/apt/lists/*

# Install Go
RUN curl -LO https://golang.org/dl/go1.21.5.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz \
    && rm go1.21.5.linux-amd64.tar.gz

# Set Go environment
ENV PATH=$PATH:/usr/local/go/bin
ENV GOPATH=/go
ENV PATH=$PATH:$GOPATH/bin

# Set the working directory
WORKDIR /app

# Copy Go module files
COPY go.mod ./
COPY go.sum ./

# Copy source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY manifests/ ./manifests/

# Build the Go application
RUN go build -o awx-deployer ./cmd/awx-deployer

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

# Run the deployment
exec ./awx-deployer
EOF

# Make entry script executable
RUN chmod +x /app/entrypoint.sh

# Set default kubeconfig path
ENV KUBECONFIG=/kubeconfig

# Set entrypoint
ENTRYPOINT ["/app/entrypoint.sh"]
