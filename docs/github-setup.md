# GitHub Repository Setup Guide

## Required Secrets

Configure the following secrets in your GitHub repository:

### 1. KUBECONFIG
- **Description**: Base64 encoded kubeconfig file for your Kubernetes cluster
- **How to get**: 
  ```bash
  # Encode your kubeconfig file
  cat ~/.kube/config | base64 -w 0
  ```
- **Value**: The base64 encoded string output

### 2. GITHUB_TOKEN (Optional)
- **Description**: GitHub Personal Access Token for GHCR access
- **Note**: Usually not needed as GitHub Actions provides `GITHUB_TOKEN` automatically
- **Scope**: `write:packages` if creating manually

## Setting Up Secrets

1. Go to your repository on GitHub
2. Click on **Settings** tab
3. Click on **Secrets and variables** → **Actions**
4. Click **New repository secret**
5. Add each secret with the name and value

## Verifying Setup

1. Push code to your repository
2. Check the **Actions** tab
3. Monitor the deployment workflow
4. Access AWX at `https://awx.sin.padminisys.com`

## Troubleshooting

### Common Issues

1. **Kubeconfig Access Issues**
   - Ensure the kubeconfig is properly base64 encoded
   - Verify cluster admin permissions
   - Check cluster connectivity

2. **Ingress Not Working**
   - Verify cert-manager is installed
   - Check if `letsencrypt-prod` cluster issuer exists
   - Ensure nginx ingress controller is running

3. **Storage Issues**
   - Verify hostPath directories exist: `/opt/awx/postgres`, `/opt/awx/projects`
   - Check node permissions for directory creation

### Debug Commands

```bash
# Check AWX status
kubectl get awx -n awx

# Check pods
kubectl get pods -n awx

# Check operator logs
kubectl logs -f deployment/awx-operator-controller-manager -n awx-system

# Check AWX instance logs
kubectl logs -f deployment/awx-instance -n awx
```
