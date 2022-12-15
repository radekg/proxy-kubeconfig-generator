package generator

import (
	"fmt"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/radekg/proxy-kubeconfig-generator/pkg/k8s"
)

// GenerateProxyKubeConfigFromSA generates a Secret containing a Kubeconfig with the specified
// Service Account's token and server URL and CA certificate from the specified Secret.
func GenerateProxyKubeConfigFromSA(opArgs k8s.OperationArgs) (*clientcmdapi.Config, error) {
	/* serviceAccountName string, namespace string, server string, serverTLSSecretName string, serverTLSSecretCAKey string, serverTLSSecretNamespace string, kubeconfigSecretKey string*/
	// Get Tenant Service Account token
	saSecret, err := k8s.GetServiceAccountSecret(opArgs)
	if err != nil { // Logging taken care of.
		return nil, err
	}
	if _, ok := saSecret.Data["token"]; !ok {
		err := fmt.Errorf("secret '%s' does not contain a token", saSecret.Name)
		opArgs.Logger().Error("No required token field found in secret",
			"secret-name", saSecret.Name,
			"secret-namespace", saSecret.Namespace,
			"reason", err)
		return nil, err
	}

	// Get Server Proxy CA certificate
	proxyCA, err := k8s.GetSourceSecretField(opArgs)
	if err != nil { // Logging take care of.
		return nil, err
	}

	// Generate the client Config for the Tenant Owner
	tenantConfig, err := k8s.BuildKubeConfigFromToken(saSecret.Data["token"], proxyCA, opArgs)
	if err != nil { // Logging taken care of.
		return nil, err
	}

	return tenantConfig, err
}
