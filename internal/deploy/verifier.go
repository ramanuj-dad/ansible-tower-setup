package deploy

import (
	"context"
	"fmt"
	"log"
	"strings"

	"awx-deployer/internal/config"
	"awx-deployer/internal/k8s"
)

// DeploymentVerifier handles verification of AWX deployment
type DeploymentVerifier struct {
	k8sClient *k8s.KubernetesClient
	config    *config.Config
}

// NewDeploymentVerifier creates a new deployment verifier
func NewDeploymentVerifier(k8sClient *k8s.KubernetesClient, config *config.Config) *DeploymentVerifier {
	return &DeploymentVerifier{
		k8sClient: k8sClient,
		config:    config,
	}
}

// Verify verifies that the AWX deployment is working correctly
func (v *DeploymentVerifier) Verify(ctx context.Context) error {
	log.Println("Verifying AWX deployment...")

	// Verify AWX instance exists
	if err := v.verifyAWXInstance(ctx); err != nil {
		return fmt.Errorf("AWX instance verification failed: %v", err)
	}

	// Verify PostgreSQL is running
	if err := v.verifyPostgreSQL(ctx); err != nil {
		return fmt.Errorf("PostgreSQL verification failed: %v", err)
	}

	// Verify AWX web is running
	if err := v.verifyAWXWeb(ctx); err != nil {
		return fmt.Errorf("AWX web verification failed: %v", err)
	}

	// Verify AWX task manager is running
	if err := v.verifyAWXTask(ctx); err != nil {
		return fmt.Errorf("AWX task verification failed: %v", err)
	}

	// Verify services exist
	if err := v.verifyServices(ctx); err != nil {
		return fmt.Errorf("Services verification failed: %v", err)
	}

	// Verify ingress (if configured)
	if err := v.verifyIngress(ctx); err != nil {
		log.Printf("Warning: Ingress verification failed: %v", err)
		// Don't fail verification for ingress issues, just warn
	}

	log.Println("AWX deployment verification completed successfully!")
	return nil
}

// verifyAWXInstance verifies the AWX custom resource exists
func (v *DeploymentVerifier) verifyAWXInstance(ctx context.Context) error {
	exists, err := v.k8sClient.ResourceExists(ctx, "awx.ansible.com", "v1beta1", "awxs", v.config.AWXName, v.config.Namespace)
	if err != nil {
		return fmt.Errorf("failed to check AWX instance: %v", err)
	}

	if !exists {
		return fmt.Errorf("AWX instance %s does not exist", v.config.AWXName)
	}

	log.Printf("✓ AWX instance %s exists", v.config.AWXName)
	return nil
}

// verifyPostgreSQL verifies PostgreSQL deployment and pods
func (v *DeploymentVerifier) verifyPostgreSQL(ctx context.Context) error {
	// Check PostgreSQL deployment
	postgresDeployment := fmt.Sprintf("%s-postgres-15", v.config.AWXName)
	exists, err := v.k8sClient.ResourceExists(ctx, "apps", "v1", "deployments", postgresDeployment, v.config.Namespace)
	if err != nil {
		return fmt.Errorf("failed to check PostgreSQL deployment: %v", err)
	}

	if !exists {
		return fmt.Errorf("PostgreSQL deployment %s does not exist", postgresDeployment)
	}

	// Check PostgreSQL pod status
	labelSelector := fmt.Sprintf("app.kubernetes.io/name=postgres,app.kubernetes.io/instance=%s", v.config.AWXName)
	status, err := v.k8sClient.GetPodStatus(ctx, labelSelector, v.config.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get PostgreSQL pod status: %v", err)
	}

	if !strings.Contains(status, "Running") {
		return fmt.Errorf("PostgreSQL pod is not running, status: %s", status)
	}

	log.Printf("✓ PostgreSQL is running")
	return nil
}

// verifyAWXWeb verifies that the AWX web deployment is running
func (v *DeploymentVerifier) verifyAWXWeb(ctx context.Context) error {
	// Check AWX web deployment
	webDeployment := fmt.Sprintf("%s-web", v.config.AWXName)
	exists, err := v.k8sClient.ResourceExists(ctx, "apps", "v1", "deployments", webDeployment, v.config.Namespace)
	if err != nil {
		return fmt.Errorf("failed to check AWX web deployment: %v", err)
	}

	if !exists {
		return fmt.Errorf("AWX web deployment %s does not exist", webDeployment)
	}

	// Check AWX web pod status
	labelSelector := fmt.Sprintf("app.kubernetes.io/name=awx-web,app.kubernetes.io/instance=%s", v.config.AWXName)
	status, err := v.k8sClient.GetPodStatus(ctx, labelSelector, v.config.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get AWX web pod status: %v", err)
	}

	if !strings.Contains(status, "Running") {
		return fmt.Errorf("AWX web pod is not running, status: %s", status)
	}

	log.Printf("✓ AWX web deployment %s is running", webDeployment)
	return nil
}

// verifyAWXTask verifies that the AWX task deployment is running
func (v *DeploymentVerifier) verifyAWXTask(ctx context.Context) error {
	// Check AWX task deployment
	taskDeployment := fmt.Sprintf("%s-task", v.config.AWXName)
	exists, err := v.k8sClient.ResourceExists(ctx, "apps", "v1", "deployments", taskDeployment, v.config.Namespace)
	if err != nil {
		return fmt.Errorf("failed to check AWX task deployment: %v", err)
	}

	if !exists {
		return fmt.Errorf("AWX task deployment %s does not exist", taskDeployment)
	}

	// Check AWX task pod status
	labelSelector := fmt.Sprintf("app.kubernetes.io/name=awx-task,app.kubernetes.io/instance=%s", v.config.AWXName)
	status, err := v.k8sClient.GetPodStatus(ctx, labelSelector, v.config.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get AWX task pod status: %v", err)
	}

	if !strings.Contains(status, "Running") {
		return fmt.Errorf("AWX task pod is not running, status: %s", status)
	}

	log.Printf("✓ AWX task deployment %s is running", taskDeployment)
	return nil
}

// verifyServices verifies that the required services exist
func (v *DeploymentVerifier) verifyServices(ctx context.Context) error {
	services := []string{
		fmt.Sprintf("%s-service", v.config.AWXName),
		fmt.Sprintf("%s-postgres-15", v.config.AWXName),
	}

	for _, service := range services {
		exists, err := v.k8sClient.ResourceExists(ctx, "", "v1", "services", service, v.config.Namespace)
		if err != nil {
			return fmt.Errorf("failed to check service %s: %v", service, err)
		}

		if !exists {
			return fmt.Errorf("service %s does not exist", service)
		}
		log.Printf("✓ Service %s exists", service)
	}

	return nil
}

// verifyIngress verifies the ingress resource exists and gets its status
func (v *DeploymentVerifier) verifyIngress(ctx context.Context) error {
	ingressName := fmt.Sprintf("%s-ingress", v.config.AWXName)
	exists, err := v.k8sClient.ResourceExists(ctx, "networking.k8s.io", "v1", "ingresses", ingressName, v.config.Namespace)
	if err != nil {
		return fmt.Errorf("failed to check ingress: %v", err)
	}

	if !exists {
		log.Printf("Ingress %s not configured, skipping status check.", ingressName)
		return nil
	}

	status, err := v.k8sClient.GetIngressStatus(ctx, ingressName, v.config.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get ingress status: %v", err)
	}

	log.Printf("✓ Ingress status for %s: %s", ingressName, status)
	return nil
}
