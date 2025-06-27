package k8s

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// KubernetesClient handles all Kubernetes operations using client-go
type KubernetesClient struct {
	clientset       kubernetes.Interface
	dynamicClient   dynamic.Interface
	discoveryClient *discovery.DiscoveryClient
}

// NewKubernetesClient creates a new Kubernetes client using client-go
func NewKubernetesClient(kubeconfigPath string) (*KubernetesClient, error) {
	var config *rest.Config
	var err error

	if kubeconfigPath != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to build config from kubeconfig: %v", err)
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get in-cluster config: %v", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %v", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %v", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client: %v", err)
	}

	return &KubernetesClient{
		clientset:       clientset,
		dynamicClient:   dynamicClient,
		discoveryClient: discoveryClient,
	}, nil
}

// Apply applies a YAML manifest file
func (k *KubernetesClient) Apply(ctx context.Context, manifestPath string) error {
	manifestData, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest file %s: %v", manifestPath, err)
	}

	decoder := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	obj := &unstructured.Unstructured{}
	_, gvk, err := decoder.Decode(manifestData, nil, obj)
	if err != nil {
		return fmt.Errorf("failed to decode manifest %s: %v", manifestPath, err)
	}

	gvr, err := k.gvrForGVK(gvk)
	if err != nil {
		return fmt.Errorf("failed to get GVR for GVK %s: %v", gvk.String(), err)
	}

	namespace := obj.GetNamespace()
	if namespace == "" {
		// some resources are cluster-wide and don't have a namespace
		if gvr.Resource != "namespaces" && gvr.Resource != "persistentvolumes" {
			namespace = "default"
		}
	}

	var resource dynamic.ResourceInterface
	if namespace != "" {
		resource = k.dynamicClient.Resource(gvr).Namespace(namespace)
	} else {
		resource = k.dynamicClient.Resource(gvr)
	}

	_, createErr := resource.Create(ctx, obj, metav1.CreateOptions{})
	if createErr != nil {
		if errors.IsAlreadyExists(createErr) {
			existingObj, getErr := resource.Get(ctx, obj.GetName(), metav1.GetOptions{})
			if getErr != nil {
				return fmt.Errorf("failed to get existing resource %s: %v", obj.GetName(), getErr)
			}
			obj.SetResourceVersion(existingObj.GetResourceVersion())
			_, updateErr := resource.Update(ctx, obj, metav1.UpdateOptions{})
			if updateErr != nil {
				return fmt.Errorf("failed to update resource %s: %v", obj.GetName(), updateErr)
			}
			return nil
		}
		return fmt.Errorf("failed to create resource %s: %v", obj.GetName(), createErr)
	}

	return nil
}

func (k *KubernetesClient) gvrForGVK(gvk *schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	apiResourceList, err := k.discoveryClient.ServerResourcesForGroupVersion(gvk.GroupVersion().String())
	if err != nil {
		return schema.GroupVersionResource{}, err
	}

	for _, apiResource := range apiResourceList.APIResources {
		if apiResource.Kind == gvk.Kind {
			return schema.GroupVersionResource{
				Group:    gvk.Group,
				Version:  gvk.Version,
				Resource: apiResource.Name,
			}, nil
		}
	}

	return schema.GroupVersionResource{}, fmt.Errorf("resource not found for GVK %s", gvk.String())
}

// ApplyKustomize is deprecated and will be removed.
func (k *KubernetesClient) ApplyKustomize(ctx context.Context, kustomizeURL string) error {
	return fmt.Errorf("ApplyKustomize is deprecated")
}

// ResourceExists checks if a Kubernetes resource exists
func (k *KubernetesClient) ResourceExists(ctx context.Context, group, version, resource, name, namespace string) (bool, error) {
	gvr := schema.GroupVersionResource{Group: group, Version: version, Resource: resource}
	var err error
	if namespace != "" {
		_, err = k.dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	} else {
		_, err = k.dynamicClient.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
	}

	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to get resource %s/%s: %v", resource, name, err)
	}
	return true, nil
}

// WaitForDeployment waits for a deployment to be ready
func (k *KubernetesClient) WaitForDeployment(ctx context.Context, deploymentName, namespace string) error {
	watcher, err := k.clientset.AppsV1().Deployments(namespace).Watch(ctx, metav1.ListOptions{FieldSelector: "metadata.name=" + deploymentName})
	if err != nil {
		return fmt.Errorf("failed to watch deployment: %v", err)
	}
	defer watcher.Stop()

	ch := watcher.ResultChan()
	timeout := time.After(15 * time.Minute) // 15 minute timeout, configurable?

	for {
		select {
		case event, ok := <-ch:
			if !ok {
				// Channel closed, something went wrong.
				return fmt.Errorf("watcher channel closed for deployment %s", deploymentName)
			}
			deployment, ok := event.Object.(*appsv1.Deployment)
			if !ok {
				continue
			}

			for _, cond := range deployment.Status.Conditions {
				if cond.Type == appsv1.DeploymentAvailable && cond.Status == "True" {
					return nil
				}
			}
		case <-timeout:
			return fmt.Errorf("timeout waiting for deployment %s to be ready", deploymentName)
		case <-ctx.Done():
			return fmt.Errorf("context cancelled waiting for deployment to be ready")
		}
	}
}

// GetPodStatus gets the status of pods with a given label selector
func (k *KubernetesClient) GetPodStatus(ctx context.Context, labelSelector, namespace string) (string, error) {
	pods, err := k.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return "", fmt.Errorf("failed to list pods: %v", err)
	}

	if len(pods.Items) == 0 {
		return "No pods found", nil
	}

	// For simplicity, returning the phase of the first pod.
	return string(pods.Items[0].Status.Phase), nil
}
