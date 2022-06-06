package main

import (
	"flag"
	"fmt"

	"k8s.io/client-go/kubernetes"

	"github.com/maxgio92/proxy-kubeconfig-generator/pkg/generator"
	"github.com/maxgio92/proxy-kubeconfig-generator/pkg/utils"
)

const (
	DefaultTLSecretCAKey       = "ca"
	DefaultNamespace           = "default"
	DefaultKubeconfigSecretKey = "kubeconfig"
)

func main() {
	serviceAccountName := flag.String("serviceaccount", "", "The name of the service account for which to create the kubeconfig")
	namespace := flag.String("namespace", DefaultNamespace, "(optional) The namespace of the service account and where the kubeconfig secret will be created.")
	server := flag.String("server", "", "The server url of the kubeconfig where API requests will be sent")
	serverTLSSecretNamespace := flag.String("server-tls-secret-namespace", DefaultNamespace, "(optional) The namespace of the server TLS secret.")
	serverTLSSecretName := flag.String("server-tls-secret-name", "", "The server TLS secret name")
	serverTLSSecretCAKey := flag.String("server-tls-secret-ca-key", DefaultTLSecretCAKey, "(optional) The CA key in the server TLS secret.")
	kubeconfigSecretKey := flag.String("kubeconfig-secret-key", DefaultKubeconfigSecretKey, "(optional) The key of the kubeconfig in the secret that will be created")

	flag.Parse()

	if *serviceAccountName == "" {
		flag.Usage()
		panic(fmt.Errorf("missing service account name"))
	}

	if *server == "" {
		flag.Usage()
		panic(fmt.Errorf("missing server url"))
	}

	if *serverTLSSecretName == "" {
		flag.Usage()
		panic(fmt.Errorf("missing server TLS secret name"))
	}

	config, err := utils.BuildClientConfig()
	if err != nil {
		panic(err)
	}

	clientset := kubernetes.NewForConfigOrDie(config)

	tenantConfig, err := generator.GenerateProxyKubeconfigFromSA(clientset, *serviceAccountName, *namespace, *server, *serverTLSSecretName, *serverTLSSecretCAKey, *serverTLSSecretNamespace, *kubeconfigSecretKey)
	if err != nil {
		panic(err)
	}

	err = utils.CreateKubeconfigSecret(clientset, tenantConfig, *namespace, *serviceAccountName+"-kubeconfig", *kubeconfigSecretKey)
	if err != nil {
		panic(err)
	}

	fmt.Println("Proxy kubeconfig Secret created")
}
