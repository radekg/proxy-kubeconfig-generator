package e2e

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/hashicorp/go-hclog"
	"github.com/radekg/proxy-kubeconfig-generator/pkg/configuration"
	"github.com/radekg/proxy-kubeconfig-generator/pkg/k8s"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	ServiceAccountName  = "myapp"
	KubeconfigSecretKey = "kubeconfig"
	Namespace           = "default"
)

func BuildClientConfigFromSecret(opArgs k8s.OperationArgs) (*rest.Config, error) {
	o, err := k8s.GetSourceSecretField(opArgs)
	if err != nil {
		return nil, err
	}

	c, err := clientcmd.NewClientConfigFromBytes(o)
	if err != nil {
		return nil, err
	}

	config, err := c.ClientConfig()
	if err != nil {
		return nil, err
	}

	return config, nil
}

func main() {

	appConfig := &configuration.Config{
		ServiceAccountName:  ServiceAccountName,
		KubeConfigSecretKey: KubeconfigSecretKey,
		TargetNamespace:     Namespace,
	}

	logger := hclog.Default()

	config, err := k8s.BuildKubernetesClientConfig(logger)
	if err != nil {
		panic(err)
	}

	clientset := kubernetes.NewForConfigOrDie(config)

	opArgs := k8s.NewDefaultOperationArgs(appConfig, clientset, logger)

	// Retrieve the Kubeconfig secret and build a new client Config
	tenantClientConfig, err := BuildClientConfigFromSecret(opArgs)
	if err != nil {
		panic(err)
	}

	tenantClientset := kubernetes.NewForConfigOrDie(tenantClientConfig)

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "001"},
	}

	_, err = tenantClientset.CoreV1().Namespaces().Create(
		context.TODO(),
		namespace,
		metav1.CreateOptions{},
	)
	if err != nil {
		panic(err)
	}

	// Get Tenant's Namesapces through its ClientSet
	tenantNamespaces, err := tenantClientset.CoreV1().Namespaces().List(
		context.Background(),
		metav1.ListOptions{},
	)
	if err != nil {
		panic(err)
	}

	fmt.Printf("\nThe Tenant owner %s can list only these Namespaces through the proxy:\n", ServiceAccountName)
	for _, tenantNamespace := range tenantNamespaces.Items {
		fmt.Println(tenantNamespace.Name)
	}
}
