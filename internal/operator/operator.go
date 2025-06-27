package operator

import (
	"context"
	"fmt"
	"log"
	"time"

	"awx-deployer/internal/config"
	"awx-deployer/internal/k8s"
)

// OperatorInstaller handles AWX operator installation
type OperatorInstaller struct {
	k8sClient *k8s.KubernetesClient
	config    *config.Config
}

// NewOperatorInstaller creates a new operator installer
func NewOperatorInstaller(k8sClient *k8s.KubernetesClient, config *config.Config) *OperatorInstaller {
	return &OperatorInstaller{
		k8sClient: k8sClient,
		config:    config,
	}
}

// Install installs the AWX operator using Kustomize
func (o *OperatorInstaller) Install(ctx context.Context) error {
	log.Println("Installing AWX Operator...")

	// Check if operator is already installed
	exists, err := o.k8sClient.ResourceExists(ctx, "apps", "v1", "deployments", "awx-operator-controller-manager", o.config.Namespace)
	if err != nil {
		return fmt.Errorf("failed to check if operator exists: %v", err)
	}

	if exists {
		log.Println("AWX Operator already installed, skipping installation")
		return nil
	}

	// Install operator using Kustomize
	kustomizeURL := fmt.Sprintf("github.com/ansible/awx-operator/config/default?ref=%s", o.config.OperatorVersion)
	log.Printf("Installing AWX Operator version %s...", o.config.OperatorVersion)

	if err := o.k8sClient.ApplyKustomize(ctx, kustomizeURL); err != nil {
		// Try fallback version if specific version fails
		log.Printf("Specific version failed, trying fallback version %s...", o.config.OperatorVersion)
		fallbackURL := fmt.Sprintf("github.com/ansible/awx-operator/config/default?ref=%s", o.config.OperatorVersion)
		if err := o.k8sClient.ApplyKustomize(ctx, fallbackURL); err != nil {
			return fmt.Errorf("failed to install AWX operator: %v", err)
		}
	}

	log.Println("Waiting for AWX Operator to be ready...")

	// Wait for operator deployment to be available
	if err := o.waitForOperatorReady(ctx); err != nil {
		return fmt.Errorf("operator failed to become ready: %v", err)
	}

	log.Println("AWX Operator installed successfully")
	return nil
}

// waitForOperatorReady waits for the operator deployment to be ready
func (o *OperatorInstaller) waitForOperatorReady(ctx context.Context) error {
	timeout := time.Duration(o.config.OperatorTimeout) * time.Minute
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Wait for the deployment to be ready
	if err := o.k8sClient.WaitForDeployment(ctxWithTimeout, "awx-operator-controller-manager", o.config.Namespace); err != nil {
		return fmt.Errorf("operator deployment not ready: %v", err)
	}

	// Additional check to ensure operator pods are running
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctxWithTimeout.Done():
			return fmt.Errorf("timeout waiting for operator pods to be ready")
		case <-ticker.C:
			status, err := o.k8sClient.GetPodStatus(ctxWithTimeout, "control-plane=controller-manager", o.config.Namespace)
			if err != nil {
				log.Printf("Warning: Could not get operator pod status: %v", err)
				continue
			}

			if status == "Running" {
				log.Println("Operator pods are running")
				return nil
			}

			log.Printf("Operator pod status: %s, waiting...", status)
		}
	}
}
