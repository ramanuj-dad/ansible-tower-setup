# Ansible AWX Tower Setup Automation

This repository provides complete automation for deploying Ansible AWX on Kubernetes using the AWX Operator with GitHub Actions.

## 🚀 Features

- ✅ **Automated AWX Operator deployment**
- ✅ **AWX instance with PostgreSQL database**
- ✅ **Ingress configuration with SSL/TLS (cert-manager)**
- ✅ **HostPath persistent volumes for single-node clusters**
- ✅ **Python-based deployment script**
- ✅ **Dockerized deployment process**
- ✅ **GitHub Actions CI/CD pipeline**
- ✅ **Automatic credential extraction and logging**

## 📋 Prerequisites

- Kubernetes cluster with admin access
- cert-manager installed with valid cluster issuer (`letsencrypt-prod`)
- nginx ingress controller
- GitHub repository secrets configured:
  - `KUBECONFIG`: Base64 encoded kubeconfig file

## 🚀 Quick Start

### Option 1: Automated GitHub Actions Deployment

1. **Fork/clone this repository**
2. **Configure GitHub secrets** (see [GitHub Setup Guide](docs/github-setup.md))
3. **Push to trigger the deployment workflow**
4. **Access AWX at `https://awx.sin.padminisys.com`**

### Option 2: Manual Deployment

```bash
# Using the shell script
./deploy-manual.sh /path/to/your/kubeconfig

# Or using Docker
docker build -t awx-deployer .
docker run --rm -v /path/to/kubeconfig:/kubeconfig awx-deployer

# Or using kubectl directly
kubectl apply -f https://raw.githubusercontent.com/ansible/awx-operator/devel/deploy/awx-operator.yaml
kubectl apply -f manifests/awx-instance.yaml
```

## 🏗️ Architecture

```
GitHub Actions → Docker Image (GHCR) → Kubernetes Job → AWX Deployment
```

### Components

- **AWX Operator**: Manages AWX lifecycle on Kubernetes
- **PostgreSQL**: Database backend with persistent storage
- **Ingress**: External access with automatic SSL certificates
- **PersistentVolumes**: HostPath storage for single-node clusters
- **Python Script**: Orchestrates the entire deployment process

## 📁 Project Structure

```
.
├── deploy_awx.py              # Main Python deployment script
├── deploy-manual.sh           # Manual deployment script
├── Dockerfile                 # Container for deployment script
├── manifests/
│   └── awx-instance.yaml     # Kubernetes manifests
├── .github/workflows/
│   └── deploy-awx.yml        # GitHub Actions workflow
└── docs/                     # Documentation
    ├── github-setup.md       # GitHub configuration guide
    ├── architecture.md       # Architecture overview
    └── troubleshooting.md    # Troubleshooting guide
```

## 🔧 Configuration

### Default Settings

- **AWX URL**: `https://awx.sin.padminisys.com`
- **Admin Username**: `admin`
- **Admin Password**: `admin123!@#` (configurable)
- **PostgreSQL Storage**: 8Gi (hostPath: `/opt/awx/postgres`)
- **Projects Storage**: 8Gi (hostPath: `/opt/awx/projects`)

### Customization

To modify the configuration, edit the following files:
- `manifests/awx-instance.yaml` - AWX instance configuration
- `deploy_awx.py` - Python script settings

## 📚 Documentation

- **[GitHub Setup Guide](docs/github-setup.md)** - Configure repository secrets
- **[Architecture Overview](docs/architecture.md)** - System architecture and components
- **[Troubleshooting Guide](docs/troubleshooting.md)** - Common issues and solutions

## 🔍 Monitoring Deployment

### Check deployment status:
```bash
# Watch the deployment
kubectl get awx,pods,svc,ingress -n awx -w

# Check operator logs
kubectl logs -f deployment/awx-operator-controller-manager -n awx-system

# Check AWX instance logs
kubectl logs -f deployment/awx-instance -n awx
```

### Access Information
After successful deployment:
- **URL**: https://awx.sin.padminisys.com
- **Username**: admin
- **Password**: Check deployment logs or run:
  ```bash
  kubectl get secret awx-admin-password -n awx -o jsonpath='{.data.password}' | base64 -d
  ```

## 🐛 Troubleshooting

### Common Issues

1. **AWX not accessible**: Check ingress and SSL certificate status
2. **Storage issues**: Ensure `/opt/awx/` directories exist with proper permissions
3. **Database problems**: Check PostgreSQL pod logs and storage

See the detailed [Troubleshooting Guide](docs/troubleshooting.md) for more information.

### Quick Diagnostics
```bash
# Overall status
kubectl get awx,pods,svc,ingress,pvc -n awx

# Check for issues
kubectl get events -n awx --sort-by='.lastTimestamp'

# Reset if needed (WARNING: Data loss)
kubectl delete awx awx-instance -n awx
kubectl apply -f manifests/awx-instance.yaml
```

## 🔒 Security Notes

- AWX admin credentials are stored in Kubernetes secrets
- SSL/TLS certificates are automatically managed by cert-manager
- Ingress enforces HTTPS redirects
- Default passwords should be changed in production

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## 📄 License

This project is open source and available under the [MIT License](LICENSE).

## ⭐ Support

If you find this project helpful, please give it a star! For issues and questions, please use the GitHub issue tracker.
