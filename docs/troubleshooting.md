# Troubleshooting Guide

## Common Issues and Solutions

### 1. AWX Operator Installation Issues

#### Symptoms
- Operator pods not starting
- CRD installation failures
- Permission errors

#### Solutions
```bash
# Check operator status
kubectl get pods -n awx-system

# Check operator logs
kubectl logs -f deployment/awx-operator-controller-manager -n awx-system

# Reinstall operator if needed
kubectl delete -f https://raw.githubusercontent.com/ansible/awx-operator/devel/deploy/awx-operator.yaml
kubectl apply -f https://raw.githubusercontent.com/ansible/awx-operator/devel/deploy/awx-operator.yaml
```

### 2. AWX Instance Deployment Issues

#### Symptoms
- AWX instance stuck in pending state
- Database connection issues
- Storage mounting problems

#### Solutions
```bash
# Check AWX instance status
kubectl describe awx awx-instance -n awx

# Check all resources
kubectl get all -n awx

# Check events
kubectl get events -n awx --sort-by='.lastTimestamp'

# Check storage
kubectl get pv,pvc -A
```

### 3. Storage Issues

#### Symptoms
- PVC stuck in pending state
- Database initialization failures
- Permission denied errors

#### Solutions
```bash
# Check if hostPath directories exist
sudo ls -la /opt/awx/

# Create directories if missing
sudo mkdir -p /opt/awx/postgres /opt/awx/projects
sudo chmod 755 /opt/awx/postgres /opt/awx/projects

# Check PV status
kubectl get pv
kubectl describe pv awx-postgres-pv
kubectl describe pv awx-projects-pv
```

### 4. Ingress and SSL Issues

#### Symptoms
- AWX not accessible via domain
- SSL certificate issues
- Redirect loops

#### Solutions
```bash
# Check ingress status
kubectl get ingress -n awx
kubectl describe ingress -n awx

# Check cert-manager
kubectl get clusterissuer
kubectl get certificate -n awx

# Check nginx ingress controller
kubectl get pods -n ingress-nginx

# Force certificate renewal
kubectl delete certificate awx-tls -n awx
```

### 5. Database Issues

#### Symptoms
- Database connection timeouts
- Migration failures
- Data persistence issues

#### Solutions
```bash
# Check PostgreSQL pod
kubectl get pods -n awx | grep postgres
kubectl logs awx-instance-postgres-13-0 -n awx

# Check PostgreSQL service
kubectl get svc -n awx | grep postgres

# Reset database (WARNING: Data loss)
kubectl delete pvc awx-instance-postgres-13 -n awx
kubectl delete awx awx-instance -n awx
kubectl apply -f manifests/awx-instance.yaml
```

### 6. Authentication Issues

#### Symptoms
- Cannot login with admin credentials
- Password not working
- User creation failures

#### Solutions
```bash
# Get admin password
kubectl get secret awx-admin-password -n awx -o jsonpath='{.data.password}' | base64 -d

# Reset admin password
kubectl delete secret awx-admin-password -n awx
kubectl create secret generic awx-admin-password -n awx --from-literal=password=newpassword123

# Restart AWX
kubectl rollout restart deployment awx-instance -n awx
```

## Debug Commands

### General Status Check
```bash
# Overall cluster health
kubectl get nodes
kubectl get pods --all-namespaces

# AWX specific
kubectl get awx,pods,svc,ingress,pvc -n awx
kubectl get pods -n awx-system
```

### Detailed Diagnostics
```bash
# AWX instance details
kubectl describe awx awx-instance -n awx

# Pod logs
kubectl logs -f deployment/awx-instance -n awx
kubectl logs -f deployment/awx-instance-task -n awx

# Events timeline
kubectl get events -n awx --sort-by='.lastTimestamp'
```

### Network Diagnostics
```bash
# Test internal connectivity
kubectl run test-pod --image=busybox --rm -it --restart=Never -- sh

# Inside the test pod:
nslookup awx-instance-service.awx.svc.cluster.local
wget -O- http://awx-instance-service.awx.svc.cluster.local
```

## Recovery Procedures

### Complete Reset
```bash
# WARNING: This will delete all AWX data
kubectl delete awx awx-instance -n awx
kubectl delete pvc --all -n awx
kubectl delete secret awx-admin-password awx-postgres-configuration -n awx
kubectl delete namespace awx

# Wait a moment, then redeploy
kubectl apply -f manifests/awx-instance.yaml
```

### Backup Important Data
```bash
# Before major changes, backup important configs
kubectl get secret awx-admin-password -n awx -o yaml > awx-admin-backup.yaml
kubectl get awx awx-instance -n awx -o yaml > awx-instance-backup.yaml
```

## Performance Tuning

### Resource Limits
```yaml
# Add to AWX spec for production
spec:
  web_resource_requirements:
    requests:
      cpu: 500m
      memory: 1Gi
    limits:
      cpu: 2000m
      memory: 2Gi
  task_resource_requirements:
    requests:
      cpu: 500m
      memory: 1Gi
    limits:
      cpu: 2000m
      memory: 2Gi
```

### Database Tuning
```yaml
# PostgreSQL configuration
spec:
  postgres_resource_requirements:
    requests:
      cpu: 500m
      memory: 2Gi
    limits:
      cpu: 1000m
      memory: 4Gi
```

## Contact and Support

- **AWX Documentation**: https://ansible.readthedocs.io/projects/awx/
- **AWX Operator GitHub**: https://github.com/ansible/awx-operator
- **Kubernetes Documentation**: https://kubernetes.io/docs/
