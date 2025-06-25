#!/bin/bash
set -e

echo "Manual AWX Deployment Script"
echo "============================="

# Check if kubeconfig is provided
if [ ! -f "$1" ]; then
    echo "Usage: $0 <path-to-kubeconfig>"
    echo "Example: $0 ~/.kube/config"
    exit 1
fi

export KUBECONFIG="$1"

echo "Using kubeconfig: $KUBECONFIG"

# Test cluster access
echo "Testing cluster access..."
kubectl cluster-info

# Install AWX Operator
echo "Installing AWX Operator..."
kubectl apply -f https://raw.githubusercontent.com/ansible/awx-operator/devel/deploy/awx-operator.yaml

# Wait for operator to be ready
echo "Waiting for AWX Operator to be ready..."
kubectl wait --for=condition=available --timeout=300s deployment/awx-operator-controller-manager -n awx-system

# Apply AWX instance and dependencies
echo "Applying AWX manifests..."
kubectl apply -f manifests/awx-instance.yaml

# Wait for AWX to be ready
echo "Waiting for AWX instance to be ready (this may take 10-15 minutes)..."
kubectl wait --for=condition=Running --timeout=1200s awx/awx-instance -n awx

# Get access information
echo ""
echo "============================================"
echo "AWX DEPLOYMENT COMPLETED!"
echo "============================================"
echo "URL: https://awx.sin.padminisys.com"
echo "Username: admin"

# Get admin password
if kubectl get secret awx-admin-password -n awx >/dev/null 2>&1; then
    PASSWORD=$(kubectl get secret awx-admin-password -n awx -o jsonpath='{.data.password}' | base64 -d)
    echo "Password: $PASSWORD"
else
    echo "Password: admin123!@# (default)"
fi

echo "============================================"
echo ""
echo "Status check:"
kubectl get awx,pods,svc,ingress -n awx
