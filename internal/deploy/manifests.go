package deploy

import (
	"context"
	"fmt"
	"log"

	"awx-deployer/internal/config"
	"awx-deployer/internal/k8s"
)

// ManifestApplier handles applying Kubernetes manifests
type ManifestApplier struct {
	k8sClient *k8s.KubernetesClient
	config    *config.Config
}

// NewManifestApplier creates a new manifest applier
func NewManifestApplier(k8sClient *k8s.KubernetesClient, config *config.Config) *ManifestApplier {
	return &ManifestApplier{
		k8sClient: k8sClient,
		config:    config,
	}
}

// Apply applies all AWX manifests
func (m *ManifestApplier) Apply(ctx context.Context) error {
	log.Println("Applying AWX manifests...")

	// Apply manifests in order
	if err := m.applyNamespace(ctx); err != nil {
		return fmt.Errorf("failed to apply namespace: %v", err)
	}

	if err := m.applyStorageClass(ctx); err != nil {
		return fmt.Errorf("failed to apply storage class: %v", err)
	}

	if err := m.applyPersistentVolumes(ctx); err != nil {
		return fmt.Errorf("failed to apply persistent volumes: %v", err)
	}

	if err := m.applySecrets(ctx); err != nil {
		return fmt.Errorf("failed to apply secrets: %v", err)
	}

	if err := m.applyAWXInstance(ctx); err != nil {
		return fmt.Errorf("failed to apply AWX instance: %v", err)
	}

	log.Println("All manifests applied successfully")
	return nil
}

// applyNamespace creates the AWX namespace
func (m *ManifestApplier) applyNamespace(ctx context.Context) error {
	exists, err := m.k8sClient.ResourceExists(ctx, "namespace", m.config.Namespace, "")
	if err != nil {
		return err
	}
	if exists {
		log.Printf("Namespace %s already exists", m.config.Namespace)
		return nil
	}

	manifest := fmt.Sprintf(`apiVersion: v1
kind: Namespace
metadata:
  name: %s
  labels:
    name: %s`, m.config.Namespace, m.config.Namespace)

	tempFile, err := m.k8sClient.CreateTempFile(manifest, "namespace.yaml")
	if err != nil {
		return err
	}

	if err := m.k8sClient.Apply(ctx, tempFile); err != nil {
		return err
	}

	log.Printf("Created namespace %s", m.config.Namespace)
	return nil
}

// applyStorageClass creates the hostpath storage class
func (m *ManifestApplier) applyStorageClass(ctx context.Context) error {
	exists, err := m.k8sClient.ResourceExists(ctx, "storageclass", m.config.StorageClass, "")
	if err != nil {
		return err
	}
	if exists {
		log.Printf("Storage class %s already exists", m.config.StorageClass)
		return nil
	}

	manifest := fmt.Sprintf(`apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: %s
provisioner: kubernetes.io/no-provisioner
volumeBindingMode: WaitForFirstConsumer
allowVolumeExpansion: true`, m.config.StorageClass)

	tempFile, err := m.k8sClient.CreateTempFile(manifest, "storage-class.yaml")
	if err != nil {
		return err
	}

	if err := m.k8sClient.Apply(ctx, tempFile); err != nil {
		return err
	}

	log.Printf("Created storage class %s", m.config.StorageClass)
	return nil
}

// applyPersistentVolumes creates the required persistent volumes
func (m *ManifestApplier) applyPersistentVolumes(ctx context.Context) error {
	// PostgreSQL PV
	postgresExists, err := m.k8sClient.ResourceExists(ctx, "pv", "awx-postgres-pv", "")
	if err != nil {
		return err
	}
	if !postgresExists {
		postgresManifest := fmt.Sprintf(`apiVersion: v1
kind: PersistentVolume
metadata:
  name: awx-postgres-pv
spec:
  capacity:
    storage: %s
  accessModes:
    - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  storageClassName: %s
  hostPath:
    path: /opt/awx/postgres
    type: DirectoryOrCreate`, m.config.PostgresStorage, m.config.StorageClass)

		tempFile, err := m.k8sClient.CreateTempFile(postgresManifest, "postgres-pv.yaml")
		if err != nil {
			return err
		}

		if err := m.k8sClient.Apply(ctx, tempFile); err != nil {
			return err
		}
		log.Println("Created PostgreSQL persistent volume")
	}

	// Projects PV
	projectsExists, err := m.k8sClient.ResourceExists(ctx, "pv", "awx-projects-pv", "")
	if err != nil {
		return err
	}
	if !projectsExists {
		projectsManifest := fmt.Sprintf(`apiVersion: v1
kind: PersistentVolume
metadata:
  name: awx-projects-pv
spec:
  capacity:
    storage: %s
  accessModes:
    - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  storageClassName: %s
  hostPath:
    path: /opt/awx/projects
    type: DirectoryOrCreate`, m.config.ProjectsStorage, m.config.StorageClass)

		tempFile, err := m.k8sClient.CreateTempFile(projectsManifest, "projects-pv.yaml")
		if err != nil {
			return err
		}

		if err := m.k8sClient.Apply(ctx, tempFile); err != nil {
			return err
		}
		log.Println("Created Projects persistent volume")
	}

	return nil
}

// applySecrets creates the required secrets
func (m *ManifestApplier) applySecrets(ctx context.Context) error {
	// PostgreSQL configuration secret
	postgresExists, err := m.k8sClient.ResourceExists(ctx, "secret", "awx-postgres-configuration", m.config.Namespace)
	if err != nil {
		return err
	}
	if !postgresExists {
		postgresSecret := fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: awx-postgres-configuration
  namespace: %s
type: Opaque
stringData:
  host: %s
  port: "%d"
  database: %s
  username: %s
  password: %s
  type: managed`, m.config.Namespace, m.config.PostgresHost, m.config.PostgresPort,
			m.config.PostgresDatabase, m.config.PostgresUsername, m.config.PostgresPassword)

		tempFile, err := m.k8sClient.CreateTempFile(postgresSecret, "postgres-secret.yaml")
		if err != nil {
			return err
		}

		if err := m.k8sClient.Apply(ctx, tempFile); err != nil {
			return err
		}
		log.Println("Created PostgreSQL configuration secret")
	}

	// Admin password secret
	adminExists, err := m.k8sClient.ResourceExists(ctx, "secret", "awx-admin-password", m.config.Namespace)
	if err != nil {
		return err
	}
	if !adminExists {
		adminSecret := fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: awx-admin-password
  namespace: %s
type: Opaque
stringData:
  password: %s`, m.config.Namespace, m.config.AdminPassword)

		tempFile, err := m.k8sClient.CreateTempFile(adminSecret, "admin-secret.yaml")
		if err != nil {
			return err
		}

		if err := m.k8sClient.Apply(ctx, tempFile); err != nil {
			return err
		}
		log.Println("Created admin password secret")
	}

	return nil
}

// applyAWXInstance creates the AWX instance
func (m *ManifestApplier) applyAWXInstance(ctx context.Context) error {
	exists, err := m.k8sClient.ResourceExists(ctx, "awx", m.config.AWXName, m.config.Namespace)
	if err != nil {
		return err
	}
	if exists {
		log.Printf("AWX instance %s already exists", m.config.AWXName)
		return nil
	}

	awxManifest := fmt.Sprintf(`apiVersion: awx.ansible.com/v1beta1
kind: AWX
metadata:
  name: %s
  namespace: %s
spec:
  service_type: ClusterIP
  hostname: %s
  ingress_type: ingress
  ingress_class_name: %s
  ingress_annotations: |
    cert-manager.io/cluster-issuer: "%s"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
  ingress_tls_secret: %s
  postgres_storage_class: %s
  postgres_storage_requirements:
    requests:
      storage: %s
  projects_persistence: true
  projects_storage_class: %s
  projects_storage_size: %s
  postgres_configuration_secret: awx-postgres-configuration
  admin_user: %s
  admin_password_secret: awx-admin-password`,
		m.config.AWXName, m.config.Namespace, m.config.AWXHostname,
		m.config.IngressClassName, m.config.CertIssuer, m.config.TLSSecretName,
		m.config.StorageClass, m.config.PostgresStorage, m.config.StorageClass,
		m.config.ProjectsStorage, m.config.AdminUser)

	tempFile, err := m.k8sClient.CreateTempFile(awxManifest, "awx-instance.yaml")
	if err != nil {
		return err
	}

	if err := m.k8sClient.Apply(ctx, tempFile); err != nil {
		return err
	}

	log.Printf("Created AWX instance %s", m.config.AWXName)
	return nil
}
