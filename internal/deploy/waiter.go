package deploy

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"awx-deployer/internal/config"
	"awx-deployer/internal/k8s"
)

// DeploymentWaiter handles waiting for AWX deployment to be ready
type DeploymentWaiter struct {
	k8sClient *k8s.KubernetesClient
	config    *config.Config
}

// NewDeploymentWaiter creates a new deployment waiter
func NewDeploymentWaiter(k8sClient *k8s.KubernetesClient, config *config.Config) *DeploymentWaiter {
	return &DeploymentWaiter{
		k8sClient: k8sClient,
		config:    config,
	}
}

// WaitForReady waits for the AWX deployment to be fully ready
func (d *DeploymentWaiter) WaitForReady(ctx context.Context, timeout time.Duration) error {
	log.Printf("Waiting for AWX deployment to be ready (timeout: %v)...", timeout)

	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Wait for AWX instance to exist and be processed
	if err := d.waitForAWXInstance(ctxWithTimeout); err != nil {
		return fmt.Errorf("AWX instance not ready: %v", err)
	}

	// Wait for PostgreSQL to be ready
	if err := d.waitForPostgreSQL(ctxWithTimeout); err != nil {
		return fmt.Errorf("PostgreSQL not ready: %v", err)
	}

	// Wait for AWX web deployment to be ready
	if err := d.waitForAWXWeb(ctxWithTimeout); err != nil {
		return fmt.Errorf("AWX web not ready: %v", err)
	}

	// Wait for AWX task manager to be ready
	if err := d.waitForAWXTask(ctxWithTimeout); err != nil {
		return fmt.Errorf("AWX task manager not ready: %v", err)
	}

	log.Println("AWX deployment is ready!")
	return nil
}

// waitForAWXInstance waits for the AWX custom resource to be processed
func (d *DeploymentWaiter) waitForAWXInstance(ctx context.Context) error {
	log.Println("Waiting for AWX instance to be processed...")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for AWX instance")
		case <-ticker.C:
			exists, err := d.k8sClient.ResourceExists(ctx, "awx.ansible.com", "v1beta1", "awxs", d.config.AWXName, d.config.Namespace)
			if err != nil {
				log.Printf("Warning: Could not check AWX instance: %v", err)
				continue
			}

			if exists {
				log.Println("AWX instance exists and is being processed")
				return nil
			}

			log.Println("Waiting for AWX instance to be created...")
		}
	}
}

// waitForPostgreSQL waits for PostgreSQL to be ready
func (d *DeploymentWaiter) waitForPostgreSQL(ctx context.Context) error {
	log.Println("Waiting for PostgreSQL to be ready...")

	// Expected PostgreSQL deployment name based on AWX instance name
	postgresDeployment := fmt.Sprintf("%s-postgres-15", d.config.AWXName)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for PostgreSQL")
		case <-ticker.C:
			log.Printf("Checking for deployment %s...", postgresDeployment)
			exists, err := d.k8sClient.ResourceExists(ctx, "apps", "v1", "deployments", postgresDeployment, d.config.Namespace)
			if err != nil {
				log.Printf("Warning: Could not check for PostgreSQL deployment: %v", err)
				continue
			}

			if !exists {
				log.Printf("Waiting for PostgreSQL deployment %s to be created...", postgresDeployment)
				continue
			}

			// Check PostgreSQL pod status
			labelSelector := fmt.Sprintf("app.kubernetes.io/name=postgres,app.kubernetes.io/instance=%s", d.config.AWXName)
			status, err := d.k8sClient.GetPodStatus(ctx, labelSelector, d.config.Namespace)
			if err != nil {
				log.Printf("Warning: Could not get PostgreSQL pod status: %v", err)
				continue
			}

			if strings.Contains(status, "Running") {
				log.Println("PostgreSQL is running")
				return nil
			}

			log.Printf("PostgreSQL pod status: %s, waiting...", status)
		}
	}
}

// waitForAWXWeb waits for AWX web deployment to be ready
func (d *DeploymentWaiter) waitForAWXWeb(ctx context.Context) error {
	log.Println("Waiting for AWX web to be ready...")

	// Expected AWX web deployment name
	webDeployment := fmt.Sprintf("%s-web", d.config.AWXName)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for AWX web")
		case <-ticker.C:
			// Check if web deployment exists
			exists, err := d.k8sClient.ResourceExists(ctx, "apps", "v1", "deployments", webDeployment, d.config.Namespace)
			if err != nil {
				log.Printf("Warning: Could not check AWX web deployment: %v", err)
				continue
			}

			if !exists {
				log.Printf("Waiting for AWX web deployment %s to be created...", webDeployment)
				continue
			}

			// Check web pod status
			labelSelector := fmt.Sprintf("app.kubernetes.io/name=%s,app.kubernetes.io/component=web", d.config.AWXName)
			status, err := d.k8sClient.GetPodStatus(ctx, labelSelector, d.config.Namespace)
			if err != nil {
				log.Printf("Warning: Could not get AWX web pod status: %v", err)
				continue
			}

			if strings.Contains(status, "Running") {
				log.Println("AWX web is running")
				return nil
			}

			log.Printf("AWX web pod status: %s, waiting...", status)
		}
	}
}

// waitForAWXTask waits for the AWX task manager to be ready
func (d *DeploymentWaiter) waitForAWXTask(ctx context.Context) error {
	log.Println("Waiting for AWX task manager to be ready...")

	// Expected AWX task deployment name
	taskDeployment := fmt.Sprintf("%s-task", d.config.AWXName)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for AWX task manager")
		case <-ticker.C:
			// Check if task deployment exists
			exists, err := d.k8sClient.ResourceExists(ctx, "apps", "v1", "deployments", taskDeployment, d.config.Namespace)
			if err != nil {
				log.Printf("Warning: Could not check AWX task deployment: %v", err)
				continue
			}

			if !exists {
				log.Printf("Waiting for AWX task deployment %s to be created...", taskDeployment)
				continue
			}

			// Check task pod status
			labelSelector := fmt.Sprintf("app.kubernetes.io/name=%s,app.kubernetes.io/component=task", d.config.AWXName)
			status, err := d.k8sClient.GetPodStatus(ctx, labelSelector, d.config.Namespace)
			if err != nil {
				log.Printf("Warning: Could not get AWX task pod status: %v", err)
				continue
			}

			if strings.Contains(status, "Running") {
				log.Println("AWX task manager is running")
				return nil
			}

			log.Printf("AWX task pod status: %s, waiting...", status)
		}
	}
}
