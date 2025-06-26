package k8s

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// KubernetesClient handles all kubectl operations
type KubernetesClient struct {
	kubeconfigPath string
}

// NewKubernetesClient creates a new Kubernetes client
func NewKubernetesClient(kubeconfigPath string) (*KubernetesClient, error) {
	// Verify kubeconfig file exists and is valid
	if err := validateKubeconfig(kubeconfigPath); err != nil {
		return nil, fmt.Errorf("invalid kubeconfig: %v", err)
	}

	return &KubernetesClient{
		kubeconfigPath: kubeconfigPath,
	}, nil
}

// validateKubeconfig verifies the kubeconfig file is valid and accessible
func validateKubeconfig(kubeconfigPath string) error {
	// Check if file exists
	if kubeconfigPath == "" {
		return fmt.Errorf("kubeconfig path is empty")
	}

	// Test kubectl connectivity
	cmd := exec.Command("kubectl", "cluster-info", "--kubeconfig", kubeconfigPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cannot connect to cluster: %v", err)
	}

	return nil
}

// Apply applies a YAML manifest file
func (k *KubernetesClient) Apply(ctx context.Context, manifestPath string) error {
	cmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", manifestPath, "--kubeconfig", k.kubeconfigPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kubectl apply failed: %v, output: %s", err, output)
	}
	return nil
}

// ApplyKustomize applies a kustomize URL
func (k *KubernetesClient) ApplyKustomize(ctx context.Context, kustomizeURL string) error {
	cmd := exec.CommandContext(ctx, "kubectl", "apply", "-k", kustomizeURL, "--kubeconfig", k.kubeconfigPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kubectl apply -k failed: %v, output: %s", err, output)
	}
	return nil
}

// ResourceExists checks if a Kubernetes resource exists
func (k *KubernetesClient) ResourceExists(ctx context.Context, resourceType, name, namespace string) (bool, error) {
	args := []string{"get", resourceType, name, "--kubeconfig", k.kubeconfigPath, "--ignore-not-found"}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("kubectl get failed: %v", err)
	}

	return strings.TrimSpace(string(output)) != "", nil
}

// WaitForDeployment waits for a deployment to be ready
func (k *KubernetesClient) WaitForDeployment(ctx context.Context, deploymentName, namespace string) error {
	args := []string{
		"wait", "deployment", deploymentName,
		"--for=condition=Available",
		"--timeout=300s",
		"--kubeconfig", k.kubeconfigPath,
	}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("waiting for deployment failed: %v, output: %s", err, output)
	}
	return nil
}

// GetPodStatus gets the status of pods with a given label selector
func (k *KubernetesClient) GetPodStatus(ctx context.Context, labelSelector, namespace string) (string, error) {
	args := []string{
		"get", "pods",
		"-l", labelSelector,
		"-o", "jsonpath={.items[*].status.phase}",
		"--kubeconfig", k.kubeconfigPath,
	}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("kubectl get pods failed: %v", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// GetIngressStatus gets the status of an ingress
func (k *KubernetesClient) GetIngressStatus(ctx context.Context, ingressName, namespace string) (string, error) {
	args := []string{
		"get", "ingress", ingressName,
		"-o", "jsonpath={.status.loadBalancer.ingress[0].ip}",
		"--kubeconfig", k.kubeconfigPath,
	}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("kubectl get ingress failed: %v", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// CreateTempFile creates a temporary file with the given content
func (k *KubernetesClient) CreateTempFile(content, filename string) (string, error) {
	tempDir := "/tmp"
	tempFile := filepath.Join(tempDir, filename)

	cmd := exec.Command("sh", "-c", fmt.Sprintf("cat > %s", tempFile))
	cmd.Stdin = strings.NewReader(content)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}

	return tempFile, nil
}
