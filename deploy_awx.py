#!/usr/bin/env python3
"""
AWX Deployment Script
Automates the deployment of Ansible AWX on Kubernetes using the AWX Operator
"""

import os
import sys
import time
import json
import base64
import logging
import subprocess
import tempfile
import shutil
from typing import Optional

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(),
        logging.FileHandler("awx_deployment.log")
    ]
)
logger = logging.getLogger(__name__)


class AWXDeployer:
    """Handles the deployment of AWX on Kubernetes using the AWX Operator"""
    
    def __init__(self, kubeconfig_path: str = "/kubeconfig"):
        """
        Initialize AWX deployer
        
        Args:
            kubeconfig_path: Path to the kubeconfig file
        """
        self.kubeconfig_path = kubeconfig_path
        self.namespace = "awx"
        self.awx_name = "awx-instance"
        self.domain = "awx.sin.padminisys.com"
        self.temp_dir = tempfile.mkdtemp(prefix="awx-deployer-")
        
        # Set KUBECONFIG environment variable
        os.environ['KUBECONFIG'] = self.kubeconfig_path
        logger.info("Initialized AWX deployer with kubeconfig: %s",
                    kubeconfig_path)
        
    def run_kubectl(
        self, command: str, capture_output: bool = True
    ) -> subprocess.CompletedProcess:
        """
        Execute kubectl command
        
        Args:
            command: The kubectl command to run
            capture_output: Whether to capture command output
            
        Returns:
            The completed process
        """
        full_command = f"kubectl {command}"
        logger.debug(f"Executing: {full_command}")
        
        result = subprocess.run(
            full_command,
            shell=True,
            capture_output=capture_output,
            text=True
        )
        
        if result.returncode != 0 and capture_output:
            logger.error(f"Command failed: {result.stderr}")
        else:
            logger.debug(f"Command succeeded: {result.stdout[:100]}...")
            
        return result
    
    def resource_exists(
        self, resource_type: str, name: str, namespace: str = None
    ) -> bool:
        """
        Check if a Kubernetes resource exists
        
        Args:
            resource_type: Type of resource (e.g., 'deployment', 'service')
            name: Name of the resource
            namespace: Namespace of the resource
            
        Returns:
            True if resource exists, False otherwise
        """
        ns = namespace or self.namespace
        ns_arg = f"-n {ns}" if namespace or self.namespace else ""
        
        result = self.run_kubectl(
            f"get {resource_type} {name} {ns_arg} --ignore-not-found"
        )
        
        return result.returncode == 0 and result.stdout.strip() != ""
    
    def wait_for_deployment(
        self, deployment_name: str, namespace: str = None, timeout: int = 600
    ) -> bool:
        """
        Wait for deployment to be ready
        
        Args:
            deployment_name: Name of the deployment
            namespace: Namespace of the deployment
            timeout: Timeout in seconds
            
        Returns:
            True if deployment is ready, False otherwise
        """
        ns = namespace or self.namespace
        logger.info(f"Waiting for deployment {deployment_name} in {ns}")
        
        start_time = time.time()
        while time.time() - start_time < timeout:
            result = self.run_kubectl(
                f"get deployment {deployment_name} -n {ns} -o json"
            )
            
            if result.returncode == 0:
                try:
                    deployment = json.loads(result.stdout)
                    status = deployment.get('status', {})
                    ready_replicas = status.get('readyReplicas', 0)
                    replicas = status.get('replicas', 1)
                    
                    elapsed = int(time.time() - start_time)
                    if ready_replicas == replicas:
                        logger.info(
                            "Deployment %s ready after %ds",
                            deployment_name, elapsed
                        )
                        return True
                        
                    if elapsed % 30 == 0:  # Log every 30 seconds
                        logger.info(
                            "Waiting: %d/%d replicas ready",
                            ready_replicas, replicas
                        )
                        
                except json.JSONDecodeError:
                    logger.warning("Failed to parse deployment status")
            
            time.sleep(10)
        
        logger.error(f"Timeout waiting for deployment {deployment_name}")
        return False
    
    def wait_for_awx_instance(self, timeout: int = 1200) -> bool:
        """
        Wait for AWX instance to be ready
        
        Args:
            timeout: Timeout in seconds
            
        Returns:
            True if AWX instance is ready, False otherwise
        """
        logger.info(
            "Waiting for AWX instance %s (timeout: %ds)",
            self.awx_name, timeout
        )
        
        start_time = time.time()
        while time.time() - start_time < timeout:
            elapsed = int(time.time() - start_time)
            
            cmd = f"get awx {self.awx_name} -n {self.namespace} -o json"
            result = self.run_kubectl(cmd)
            
            if result.returncode == 0:
                try:
                    awx = json.loads(result.stdout)
                    conditions = awx.get('status', {}).get('conditions', [])
                    status_msg = awx.get('status', {}).get('message', '')
                    
                    # Log the current status
                    if elapsed % 60 == 0:  # Log every minute
                        logger.info(f"Current status: {status_msg}")
                    
                    # Check if instance is running
                    for condition in conditions:
                        is_running = (condition.get('type') == 'Running' and
                                      condition.get('status') == 'True')
                        if is_running:
                            logger.info(
                                "AWX instance %s is running after %ds",
                                self.awx_name, elapsed
                            )
                            return True
                except json.JSONDecodeError as e:
                    logger.warning(f"Failed to parse AWX status: {e}")
            
            # Log progress periodically
            if elapsed % 60 == 0:  # Log every minute
                logger.info(
                    f"Still waiting for AWX instance ({elapsed}s elapsed)..."
                )
            
            time.sleep(30)
        
        logger.error(f"Timeout waiting for AWX instance {self.awx_name}")
        return False
    
    def create_namespace(self):
        """
        Create the AWX namespace if it doesn't exist
        """
        logger.info(f"Ensuring namespace {self.namespace} exists")
        
        # Check if namespace exists
        if self.resource_exists("namespace", self.namespace):
            logger.info(f"Namespace {self.namespace} already exists")
            return
            
        # Create namespace
        result = self.run_kubectl(f"create namespace {self.namespace}")
        if result.returncode != 0 and "already exists" not in result.stderr:
            raise Exception(f"Failed to create namespace: {result.stderr}")
        
        logger.info(f"Successfully created namespace {self.namespace}")
    
    def install_awx_operator(self):
        """
        Install AWX Operator if not already installed
        """
        logger.info("Ensuring AWX Operator is installed")
        
        # Check if operator already deployed
        if self.resource_exists(
            "deployment",
            "awx-operator-controller-manager",
            "awx"
        ):
            logger.info("AWX Operator already installed, skipping")
            return
            
        # Apply AWX Operator using the new Kustomize method
        # This uses the stable tag instead of the deprecated raw YAML URL
        logger.info("Installing AWX Operator using Kustomize...")
        operator_kustomize_url = (
            "github.com/ansible/awx-operator/config/default"
            "?ref=2.19.1"
        )
        result = self.run_kubectl(f"apply -k {operator_kustomize_url}")
        if result.returncode != 0:
            # Fallback to latest stable release if specific version fails
            logger.warning("Specific version failed, trying latest stable...")
            fallback_url = (
                "github.com/ansible/awx-operator/config/default?ref=2.19.1"
            )
            result = self.run_kubectl(f"apply -k {fallback_url}")
            if result.returncode != 0:
                raise Exception(
                    f"Failed to install AWX Operator: {result.stderr}"
                )
        
        # Wait for operator to be ready
        logger.info("Waiting for AWX Operator to be ready")
        if not self.wait_for_deployment(
            "awx-operator-controller-manager",
            "awx"
        ):
            raise Exception("AWX Operator deployment failed")
            
        logger.info("AWX Operator installed successfully")
    
    def create_storage_class(self):
        """
        Create storage class for hostPath if it doesn't exist
        """
        logger.info("Ensuring hostPath storage class exists")
        
        # Check if storage class exists
        if self.resource_exists("storageclass", "hostpath"):
            logger.info("Storage class 'hostpath' already exists")
            return
        
        storage_class = """
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: hostpath
provisioner: kubernetes.io/no-provisioner
volumeBindingMode: WaitForFirstConsumer
allowVolumeExpansion: true
"""
        storage_class_file = os.path.join(self.temp_dir, "storage-class.yaml")
        with open(storage_class_file, "w") as f:
            f.write(storage_class)
        
        result = self.run_kubectl(f"apply -f {storage_class_file}")
        if result.returncode != 0:
            logger.warning(f"Storage class creation warning: {result.stderr}")
        else:
            logger.info("Storage class 'hostpath' created successfully")
    
    def create_persistent_volumes(self):
        """
        Create persistent volumes for AWX if they don't exist
        """
        logger.info("Ensuring persistent volumes exist")
        
        # Check if PVs exist
        postgres_exists = self.resource_exists("pv", "awx-postgres-pv")
        projects_exists = self.resource_exists("pv", "awx-projects-pv")
        
        if postgres_exists and projects_exists:
            logger.info("All required persistent volumes already exist")
            return
        
        # PostgreSQL PV
        postgres_pv = """
apiVersion: v1
kind: PersistentVolume
metadata:
  name: awx-postgres-pv
spec:
  capacity:
    storage: 8Gi
  accessModes:
    - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  storageClassName: hostpath
  hostPath:
    path: /opt/awx/postgres
    type: DirectoryOrCreate
"""
        
        # AWX Projects PV
        projects_pv = """
apiVersion: v1
kind: PersistentVolume
metadata:
  name: awx-projects-pv
spec:
  capacity:
    storage: 8Gi
  accessModes:
    - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  storageClassName: hostpath
  hostPath:
    path: /opt/awx/projects
    type: DirectoryOrCreate
"""
        
        postgres_file = os.path.join(self.temp_dir, "postgres-pv.yaml")
        projects_file = os.path.join(self.temp_dir, "projects-pv.yaml")
        
        # Create PostgreSQL PV if needed
        if not postgres_exists:
            with open(postgres_file, "w") as f:
                f.write(postgres_pv)
            
            result = self.run_kubectl(f"apply -f {postgres_file}")
            if result.returncode == 0:
                logger.info("Created PostgreSQL persistent volume")
            else:
                logger.error(
                    "Failed to create PostgreSQL PV: %s",
                    result.stderr
                )
        
        # Create Projects PV if needed
        if not projects_exists:
            with open(projects_file, "w") as f:
                f.write(projects_pv)
            
            result = self.run_kubectl(f"apply -f {projects_file}")
            if result.returncode == 0:
                logger.info("Created Projects persistent volume")
            else:
                logger.error(f"Failed to create Projects PV: {result.stderr}")
    
    def create_awx_instance(self):
        """
        Create AWX instance if it doesn't exist
        """
        logger.info(f"Ensuring AWX instance {self.awx_name} exists")
        
        # Check if AWX instance exists
        if self.resource_exists("awx", self.awx_name, self.namespace):
            logger.info(f"AWX instance {self.awx_name} already exists")
            return
        
        # Create AWX instance
        awx_spec = f"""
apiVersion: awx.ansible.com/v1beta1
kind: AWX
metadata:
  name: {self.awx_name}
  namespace: {self.namespace}
spec:
  service_type: ClusterIP
  hostname: {self.domain}
  ingress_type: ingress
  ingress_class_name: nginx
  ingress_annotations: |
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
  ingress_tls_secret: awx-tls
  postgres_storage_class: hostpath
  postgres_storage_requirements:
    requests:
      storage: 8Gi
  projects_persistence: true
  projects_storage_class: hostpath
  projects_storage_size: 8Gi
  postgres_configuration_secret: awx-postgres-configuration
  admin_user: admin
  admin_password_secret: awx-admin-password
"""
        
        awx_file = os.path.join(self.temp_dir, "awx-instance.yaml")
        with open(awx_file, "w") as f:
            f.write(awx_spec)
        
        logger.info(f"Creating AWX instance {self.awx_name}")
        result = self.run_kubectl(f"apply -f {awx_file}")
        
        if result.returncode != 0:
            raise Exception(f"Failed to create AWX instance: {result.stderr}")
        else:
            logger.info(f"AWX instance {self.awx_name} created/updated")
    
    def create_secrets(self):
        """
        Create required secrets if they don't exist
        """
        logger.info("Ensuring required secrets exist")
        
        # Check if secrets exist
        postgres_exists = self.resource_exists(
            "secret", "awx-postgres-configuration", self.namespace
        )
        admin_exists = self.resource_exists(
            "secret", "awx-admin-password", self.namespace
        )
        
        if postgres_exists and admin_exists:
            logger.info("All required secrets already exist")
            return
        
        # PostgreSQL configuration secret
        postgres_secret = """
apiVersion: v1
kind: Secret
metadata:
  name: awx-postgres-configuration
  namespace: awx
type: Opaque
stringData:
  host: awx-instance-postgres-13
  port: "5432"
  database: awx
  username: awx
  password: awxpassword
  type: managed
"""
        
        # Admin password secret
        admin_secret = """
apiVersion: v1
kind: Secret
metadata:
  name: awx-admin-password
  namespace: awx
type: Opaque
stringData:
  password: admin123!@#
"""
        
        # Create PostgreSQL secret if needed
        if not postgres_exists:
            postgres_file = os.path.join(self.temp_dir, "postgres-secret.yaml")
            with open(postgres_file, "w") as f:
                f.write(postgres_secret)
            
            result = self.run_kubectl(f"apply -f {postgres_file}")
            if result.returncode == 0:
                logger.info("Created PostgreSQL configuration secret")
            else:
                logger.error(
                    f"Failed to create PostgreSQL secret: {result.stderr}"
                )
        
        # Create Admin secret if needed
        if not admin_exists:
            admin_file = os.path.join(self.temp_dir, "admin-secret.yaml")
            with open(admin_file, "w") as f:
                f.write(admin_secret)
            
            result = self.run_kubectl(f"apply -f {admin_file}")
            if result.returncode == 0:
                logger.info("Created admin password secret")
            else:
                logger.error(
                    f"Failed to create admin secret: {result.stderr}"
                )
    
    def get_admin_password(self) -> Optional[str]:
        """
        Get AWX admin password from secret
        
        Returns:
            The admin password or None if not found
        """
        try:
            cmd = f"get secret awx-admin-password -n {self.namespace} -o json"
            result = self.run_kubectl(cmd)
            
            if result.returncode == 0:
                secret = json.loads(result.stdout)
                password_b64 = secret.get('data', {}).get('password', '')
                if password_b64:
                    return base64.b64decode(password_b64).decode('utf-8')
        except Exception as e:
            logger.error(f"Failed to get admin password: {e}")
        
        return None
    
    def print_access_info(self):
        """
        Print AWX access information
        """
        password = self.get_admin_password()
        
        print("\n" + "="*60)
        print("AWX DEPLOYMENT COMPLETED SUCCESSFULLY!")
        print("="*60)
        print(f"AWX URL: https://{self.domain}")
        print("Username: admin")
        print(f"Password: {password if password else 'admin123!@#'}")
        print("="*60)
        print("Please allow a few minutes for the ingress and SSL certificate")
        print("to be ready.")
        print("="*60 + "\n")
    
    def cleanup(self):
        """
        Clean up temporary files
        """
        try:
            if os.path.exists(self.temp_dir):
                shutil.rmtree(self.temp_dir)
                logger.debug(f"Cleaned up temporary directory {self.temp_dir}")
        except Exception as e:
            logger.warning(f"Failed to clean up temporary files: {e}")
    
    def deploy(self):
        """
        Main deployment function - runs all steps in sequence
        """
        start_time = time.time()
        
        try:
            logger.info("Starting AWX deployment")
            
            # Check cluster access
            logger.info("Checking Kubernetes cluster access")
            result = self.run_kubectl("cluster-info")
            if result.returncode != 0:
                raise Exception("Cannot access Kubernetes cluster")
            
            # Create namespace
            self.create_namespace()
            
            # Create storage resources
            self.create_storage_class()
            self.create_persistent_volumes()
            
            # Create secrets
            self.create_secrets()
            
            # Install AWX Operator
            self.install_awx_operator()
            
            # Create AWX instance
            self.create_awx_instance()
            
            # Wait for AWX to be ready
            if self.wait_for_awx_instance():
                elapsed = int(time.time() - start_time)
                logger.info(f"Deployment completed in {elapsed} seconds")
                self.print_access_info()
            else:
                logger.error("AWX deployment timed out")
                sys.exit(1)
                
        except Exception as e:
            logger.error(f"Deployment failed: {e}")
            sys.exit(1)
        finally:
            self.cleanup()


def main():
    """
    Main entry point for the script
    """
    # Add a console handler with better formatting for direct output
    console = logging.StreamHandler()
    console.setFormatter(logging.Formatter(
        '%(asctime)s [%(levelname)s] %(message)s',
        datefmt='%H:%M:%S'
    ))
    logger.addHandler(console)
    
    # Get kubeconfig path from environment or use default
    kubeconfig_path = os.getenv('KUBECONFIG', '/kubeconfig')
    
    if not os.path.exists(kubeconfig_path):
        logger.error(f"Kubeconfig file not found at {kubeconfig_path}")
        logger.error("Please provide a valid kubeconfig file")
        sys.exit(1)
    
    logger.info(f"Using kubeconfig from {kubeconfig_path}")
    deployer = AWXDeployer(kubeconfig_path)
    
    try:
        deployer.deploy()
    except KeyboardInterrupt:
        logger.info("Deployment interrupted by user")
        sys.exit(130)


if __name__ == "__main__":
    main()
