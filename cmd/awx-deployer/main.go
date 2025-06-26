package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"awx-deployer/internal/config"
	"awx-deployer/internal/deploy"
	"awx-deployer/internal/k8s"
	"awx-deployer/internal/operator"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Load configuration from environment
	cfg, err := config.NewConfigFromEnv()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize Kubernetes client
	k8sClient, err := k8s.NewKubernetesClient(cfg.KubeconfigPath)
	if err != nil {
		log.Fatalf("Failed to initialize Kubernetes client: %v", err)
	}

	ctx := context.Background()

	log.Println("Starting AWX deployment...")

	// Step 1: Install AWX Operator
	operatorInstaller := operator.NewOperatorInstaller(k8sClient, cfg)
	if err := operatorInstaller.Install(ctx); err != nil {
		log.Fatalf("Failed to install AWX operator: %v", err)
	}

	// Step 2: Apply manifests
	manifestApplier := deploy.NewManifestApplier(k8sClient, cfg)
	if err := manifestApplier.Apply(ctx); err != nil {
		log.Fatalf("Failed to apply manifests: %v", err)
	}

	// Step 3: Wait for deployment
	deploymentWaiter := deploy.NewDeploymentWaiter(k8sClient, cfg)
	if err := deploymentWaiter.WaitForReady(ctx, 15*time.Minute); err != nil {
		log.Fatalf("Deployment failed to become ready: %v", err)
	}

	// Step 4: Verify deployment
	verifier := deploy.NewDeploymentVerifier(k8sClient, cfg)
	if err := verifier.Verify(ctx); err != nil {
		log.Fatalf("Deployment verification failed: %v", err)
	}

	log.Println("AWX deployment completed successfully!")
	fmt.Printf("AWX should be accessible at: https://%s\n", cfg.AWXHostname)
	fmt.Printf("Admin username: %s\n", cfg.AdminUser)
	fmt.Printf("Admin password: %s\n", cfg.AdminPassword)
}
