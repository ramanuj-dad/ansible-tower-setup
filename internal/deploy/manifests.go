package deploy

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"awx-deployer/internal/config"
	"awx-deployer/internal/k8s"
)

// ManifestApplier handles applying Kubernetes manifests
type ManifestApplier struct {
	k8sClient     *k8s.KubernetesClient
	config        *config.Config
	manifestsPath string
}

// NewManifestApplier creates a new manifest applier
func NewManifestApplier(k8sClient *k8s.KubernetesClient, config *config.Config) *ManifestApplier {
	return &ManifestApplier{
		k8sClient:     k8sClient,
		config:        config,
		manifestsPath: "./manifests",
	}
}

// Apply applies all AWX manifests from the manifests directory
func (m *ManifestApplier) Apply(ctx context.Context) error {
	log.Println("Applying AWX manifests from static YAML files...")

	// Check if manifests directory exists
	if _, err := os.Stat(m.manifestsPath); os.IsNotExist(err) {
		return fmt.Errorf("manifests directory %s does not exist", m.manifestsPath)
	}

	// Read all YAML files from manifests directory
	files, err := filepath.Glob(filepath.Join(m.manifestsPath, "*.yaml"))
	if err != nil {
		return fmt.Errorf("failed to read manifest files: %v", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no YAML manifest files found in %s", m.manifestsPath)
	}

	// Sort files to ensure they are applied in order
	sort.Strings(files)

	log.Printf("Found %d manifest files to apply", len(files))

	// Apply each manifest file
	for _, file := range files {
		log.Printf("Applying manifest: %s", filepath.Base(file))
		if err := m.k8sClient.Apply(ctx, file); err != nil {
			return fmt.Errorf("failed to apply manifest %s: %v", file, err)
		}
	}

	log.Println("All manifests applied successfully")
	return nil
}
