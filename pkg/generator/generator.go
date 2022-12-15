package generator

import (
	"fmt"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/radekg/proxy-kubeconfig-generator/pkg/k8s"
)

const (
	DefaultTLSecretCAKey       = "ca"
	DefaultNamespace           = "default"
	DefaultKubeconfigSecretKey = "kubeconfig"
)

// Generates a Secret containing a Kubeconfig with the specified
// Service Account's token and server URL and CA certificate from
// the specified Secret.
func GenerateProxyKubeconfigFromSA(opArgs k8s.OperationArgs) (*clientcmdapi.Config, error) {
	/* serviceAccountName string, namespace string, server string, serverTLSSecretName string, serverTLSSecretCAKey string, serverTLSSecretNamespace string, kubeconfigSecretKey string*/
	// Get Tenant Service Account token
	saSecret, err := k8s.GetServiceAccountSecret(opArgs)
	if err != nil {
		return nil, err
	}
	if _, ok := saSecret.Data["token"]; !ok {
		return nil, fmt.Errorf("secret %s does not contain a token", saSecret.Name)
	}

	// Get Server Proxy CA certificate
	proxyCA, err := k8s.GetSecretField(opArgs)
	if err != nil {
		return nil, err
	}

	// Generate the client Config for the Tenant Owner
	tenantConfig, err := k8s.BuildKubeconfigFromToken(saSecret.Data["token"], proxyCA, opArgs)
	if err != nil {
		return nil, err
	}

	return tenantConfig, err
}
