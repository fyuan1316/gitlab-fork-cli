package k8sutil

import (
	"context"
	"fmt"
	"log" // Using standard log for consistency as requested
	"net/http"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetKubeConfig 根据优先级获取 Kubernetes REST 配置：
// 1. 如果指定了 kubeconfig 文件路径
// 2. 尝试集群内配置 (in-cluster config)
// 3. 尝试默认 kubeconfig 路径 (如 ~/.kube/config)
func GetKubeConfig() (*rest.Config, error) { // Removed kubeconfigPath parameter
	var config *rest.Config
	var err error

	log.Println("ℹ️ 尝试获取 Kubernetes 配置...")

	// 1. Try in-cluster configuration
	config, err = rest.InClusterConfig()
	if err == nil {
		log.Println("✅ 成功加载集群内部配置。")
		return config, nil
	}
	log.Printf("ℹ️ 无法加载集群内部配置 (%v)，尝试从默认 kubeconfig 路径加载...\n", err)

	// 2. Fallback to default kubeconfig paths (e.g., ~/.kube/config)
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	config, err = kubeConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("❌ 无法加载 Kubernetes 配置 (既不在集群内，也未从默认 ~/.kube/config 路径加载): %w", err)
	}
	log.Println("✅ 成功从默认 kubeconfig 或默认路径加载配置。")
	return config, nil
}

// CheckK8sNamespaceExists 检查给定的 Kubernetes 命名空间是否存在。
// 它需要一个 Kubernetes REST 配置和一个命名空间名称。
func CheckK8sNamespaceExists(config *rest.Config, namespace string) (bool, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return false, fmt.Errorf("创建 Kubernetes 客户端失败: %w", err)
	}

	log.Printf("ℹ️ 正在检查 Kubernetes 命名空间 '%s' 是否存在...\n", namespace)

	_, err = clientset.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
	if err != nil {
		if statusError, isStatusError := err.(*errors.StatusError); isStatusError && statusError.ErrStatus.Code == http.StatusNotFound {
			log.Printf("ℹ️ Kubernetes 命名空间 '%s' 不存在。\n", namespace)
			return false, nil // Namespace doesn't exist, not an internal error
		}
		// Other types of errors, like connection issues, permission denied
		return false, fmt.Errorf("检查 Kubernetes 命名空间 '%s' 失败: %w", namespace, err)
	}

	log.Printf("✅ Kubernetes 命名空间 '%s' 已存在。\n", namespace)
	return true, nil // Namespace exists
}

// GetSecretValue 从 Kubernetes Secret 中获取指定 key 的值
func GetSecretValue(kubeConfig *rest.Config, namespace string, secretName string, key string) (string, error) {
	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return "", fmt.Errorf("创建 Kubernetes 客户端失败: %w", err)
	}

	log.Printf("ℹ️ 正在从 Kubernetes Secret 中获取令牌。命名空间: %s, Secret名称: %s, Key: %s\n",
		namespace, secretName, key)

	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("无法在命名空间 '%s' 中获取 Secret '%s': %w", namespace, secretName, err)
	}

	tokenBytes, ok := secret.Data[key]
	if !ok {
		return "", fmt.Errorf("Secret '%s' 中不存在 key '%s'", secretName, key)
	}

	log.Printf("✅ 成功从 Kubernetes Secret 获取令牌。命名空间: %s, Secret名称: %s\n",
		namespace, secretName)

	return string(tokenBytes), nil
}
