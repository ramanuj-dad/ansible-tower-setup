# AWX Kubernetes Deployment

This project automatically deploys AWX (Ansible Tower) on Kubernetes using GitHub Actions.

## Quick Start

1. **Set up your kubeconfig secret** (see below)
2. **Push to main branch** - the deployment will start automatically
3. **Access AWX** at `https://awx.sin.padminisys.com`

## Setting up Secrets

### KUBE_CONFIG Secret

The `KUBE_CONFIG` secret must contain your base64-encoded kubeconfig file.

#### Method 1: Using the provided script (Recommended)

```bash
# Run the encoding script
./encode-kubeconfig.sh ~/.kube/config

# Copy the output and set it as the KUBE_CONFIG secret
```

#### Method 2: Manual encoding

```bash
# Encode your kubeconfig
base64 -w 0 ~/.kube/config

# Copy the output and set it as the KUBE_CONFIG secret in GitHub
```

#### Setting the Secret in GitHub

1. Go to your repository: https://github.com/ramanuj-dad/ansible-tower-setup
2. Navigate to **Settings** > **Secrets and variables** > **Actions**
3. Click **New repository secret** or update existing `KUBE_CONFIG`
4. Name: `KUBE_CONFIG`
5. Value: The base64-encoded kubeconfig string
6. Click **Add secret**

### Important Notes

- ⚠️ **Do NOT use localhost/127.0.0.1** in your kubeconfig - it won't work in GitHub Actions
- ✅ Use the external cluster IP or hostname that's accessible from the internet
- ✅ Make sure your cluster allows connections from GitHub Actions runners
- ✅ The kubeconfig must be properly base64-encoded (no newlines or extra spaces)

## Troubleshooting

### "base64: invalid input" Error

This means your `KUBE_CONFIG` secret contains invalid base64 data:

1. Re-encode your kubeconfig using the script: `./encode-kubeconfig.sh ~/.kube/config`
2. Update the `KUBE_CONFIG` secret with the new value
3. Re-run the workflow

### "ERROR: kubeconfig contains localhost/127.0.0.1" Error

Your kubeconfig is using localhost, which won't work in CI:

1. Update your kubeconfig to use the external cluster IP/hostname
2. Re-encode and update the secret
3. Re-run the workflow

### Kubernetes Connection Failures

Check that:
- Your cluster is accessible from the internet
- The kubeconfig contains valid credentials
- The cluster certificates are valid
- Network policies allow connections from GitHub Actions

## Project Structure

```
.
├── deploy_awx.py           # Main deployment script
├── Dockerfile              # Container image for deployment
├── encode-kubeconfig.sh    # Helper script to encode kubeconfig
├── manifests/
│   └── awx-instance.yaml   # AWX instance configuration
└── .github/workflows/
    └── deploy-awx.yml      # GitHub Actions workflow
```

## How it Works

1. **Build**: Creates a Docker image with kubectl and the deployment script
2. **Validate**: Checks the kubeconfig secret and validates the cluster connection
3. **Deploy**: Runs the containerized deployment script with the kubeconfig mounted
   - Installs AWX Operator using Kustomize (latest stable version 2.19.1)
   - Creates necessary storage classes and persistent volumes
   - Deploys AWX instance with ingress configuration
4. **Verify**: Checks the deployment status and provides access information

## AWX Operator Installation

This project uses the **new recommended method** for installing AWX Operator:

- **Method**: Kustomize with stable release tags
- **Command**: `kubectl apply -k github.com/ansible/awx-operator/config/default?ref=2.19.1`
- **Namespace**: `awx` (operator and AWX instance in same namespace)

### Changes from Previous Versions

- ❌ **Old method** (deprecated): Raw YAML from `devel` branch - `https://raw.githubusercontent.com/ansible/awx-operator/devel/deploy/awx-operator.yaml`
- ✅ **New method**: Kustomize with stable tags - much more reliable and follows AWX best practices

## AWX Access

After successful deployment:

- **URL**: https://awx.sin.padminisys.com
- **Username**: admin
- **Password**: Retrieved from `awx-admin-password` secret or defaults to `admin123!@#`

## Manual Deployment

To run locally:

```bash
# Build the image
docker build -t awx-deployer .

# Run with your kubeconfig
docker run --rm -v ~/.kube/config:/kubeconfig:ro awx-deployer
```
