package k8s

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-hclog"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// BuildKubeConfigFromToken builds a kubeconfig.
func BuildKubeConfigFromToken(token []byte, CACertificate []byte, opArgs OperationArgs) (*clientcmdapi.Config, error) {
	clusters := make(map[string]*clientcmdapi.Cluster)
	clusters["default"] = &clientcmdapi.Cluster{
		Server:                   opArgs.AppConfig().Server,
		CertificateAuthorityData: CACertificate,
	}

	contexts := make(map[string]*clientcmdapi.Context)
	contexts["default"] = &clientcmdapi.Context{
		Cluster:   "default",
		Namespace: opArgs.AppConfig().ServerTLSSecretNamespace,
		AuthInfo:  opArgs.AppConfig().ServerTLSSecretNamespace,
	}

	authinfos := make(map[string]*clientcmdapi.AuthInfo)
	authinfos[opArgs.AppConfig().ServerTLSSecretNamespace] = &clientcmdapi.AuthInfo{
		Token: string(token),
	}

	config := clientcmdapi.Config{
		Kind:           "Config",
		APIVersion:     "v1",
		Clusters:       clusters,
		Contexts:       contexts,
		CurrentContext: "default",
		AuthInfos:      authinfos,
	}

	err := clientcmd.Validate(config)
	if err != nil {
		opArgs.Logger().Error("kubeconfig did not validate",
			"namespace", opArgs.AppConfig().ServerTLSSecretNamespace,
			"reason", err)
		return nil, err
	}

	return &config, nil
}

// BuildKubernetesClientConfig creates a Kubernetes client configuration.
func BuildKubernetesClientConfig(logger hclog.Logger) (*rest.Config, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	config, err := kubeConfig.ClientConfig()
	if err != nil {
		logger.Error("error while trying to create a client config", "reason", err)
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		return config, nil
	}
	return config, nil
}

// CreateOrUpdateKubeConfigSecret creates or updates a kubeconfig secret in the target namespace.
func CreateOrUpdateKubeConfigSecret(opArgs OperationArgs, kubeconfig *clientcmdapi.Config) error {

	existingSecret, err := opArgs.ClientSet().CoreV1().Secrets(opArgs.AppConfig().TargetNamespace).Get(
		context.Background(),
		opArgs.AppConfig().TenantSecretName(),
		metav1.GetOptions{})

	if err != nil {
		if !apiErrors.IsAlreadyExists(err) {
			opArgs.Logger().Error("Failed checking if secret exists",
				"namespace", opArgs.AppConfig().TargetNamespace,
				"secret-name", opArgs.AppConfig().TenantSecretName(),
				"reason", err)
			return err
		}
	}

	configBuffer, err := clientcmd.Write(*kubeconfig)
	if err != nil {
		opArgs.Logger().Error("Failed serializing kubeconfig to buffer",
			"reason", err)
		return err
	}

	// Just a skeleton of a secret, maybe we use the existing one.
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{},
		Data:       map[string][]byte{},
	}

	if existingSecret != nil {

		if opArgs.AppConfig().DisallowUpdates {
			opArgs.Logger().Info("Secret already exists and updates are disabled",
				"namespace", opArgs.AppConfig().TargetNamespace,
				"secret-name", opArgs.AppConfig().TenantSecretName(),
				"existing-secret-generation", existingSecret.Generation,
				"existing-secret-resource-version", existingSecret.ResourceVersion)
			return fmt.Errorf("secret '%s' already exists in namespace '%s'", opArgs.AppConfig().TenantSecretName(), opArgs.AppConfig().TargetNamespace)
		}

		opArgs.Logger().Info("Secret already exists, will be updated",
			"namespace", opArgs.AppConfig().TargetNamespace,
			"secret-name", opArgs.AppConfig().TenantSecretName(),
			"secret-key", opArgs.AppConfig().KubeConfigSecretKey,
			"existing-secret-generation", existingSecret.Generation,
			"existing-secret-resource-version", existingSecret.ResourceVersion,
			"secret-data-size", len(configBuffer))
		secret = existingSecret.DeepCopy()
	} else {
		secret.ObjectMeta.Name = opArgs.AppConfig().TenantSecretName()
	}

	secret.Data[opArgs.AppConfig().KubeConfigSecretKey] = configBuffer

	if opArgs.AppConfig().ReportOnly {
		if existingSecret != nil {
			opArgs.Logger().Info("Report only: would update a secret",
				"namespace", opArgs.AppConfig().TargetNamespace,
				"secret-name", opArgs.AppConfig().TenantSecretName(),
				"secret-key", opArgs.AppConfig().KubeConfigSecretKey,
				"existing-secret-generation", existingSecret.Generation,
				"existing-secret-resource-version", existingSecret.ResourceVersion,
				"secret-data-size", len(configBuffer))
		} else {
			opArgs.Logger().Info("Report only: would create a secret",
				"namespace", opArgs.AppConfig().TargetNamespace,
				"secret-name", opArgs.AppConfig().TenantSecretName(),
				"secret-key", opArgs.AppConfig().KubeConfigSecretKey,
				"secret-data-size", len(configBuffer))
		}
		return nil
	}

	if existingSecret != nil {

		_, err = opArgs.ClientSet().CoreV1().Secrets(opArgs.AppConfig().TargetNamespace).Update(
			context.Background(),
			secret,
			metav1.UpdateOptions{},
		)
		if err != nil {
			opArgs.Logger().Error("Failed updating secret",
				"target-namespace", opArgs.AppConfig().TenantSecretName(),
				"secret-key", opArgs.AppConfig().KubeConfigSecretKey,
				"reason", err)
			return err
		}

	} else {

		_, err = opArgs.ClientSet().CoreV1().Secrets(opArgs.AppConfig().TargetNamespace).Create(
			context.Background(),
			secret,
			metav1.CreateOptions{},
		)
		if err != nil {
			opArgs.Logger().Error("Failed creating secret",
				"target-namespace", opArgs.AppConfig().TenantSecretName(),
				"secret-key", opArgs.AppConfig().KubeConfigSecretKey,
				"reason", err)
			return err
		}

	}

	return nil
}

// GetServiceAccountSecret retrieves a secret for the service account.
func GetServiceAccountSecret(opArgs OperationArgs) (*corev1.Secret, error) {

	serviceAccount, err := opArgs.ClientSet().CoreV1().ServiceAccounts(opArgs.AppConfig().TargetNamespace).Get(
		context.Background(),
		opArgs.AppConfig().ServiceAccountName,
		metav1.GetOptions{},
	)
	if err != nil {
		opArgs.Logger().Error("Problem fetching service account",
			"namespace", opArgs.AppConfig().TargetNamespace,
			"service-account-name", opArgs.AppConfig().ServiceAccountName,
			"reason", err)
		return nil, err
	}

	if len(serviceAccount.Secrets) < 1 {
		err := fmt.Errorf("no secret found for the service account '%s' in namepsace '%s'", serviceAccount.Name, serviceAccount.Namespace)
		opArgs.Logger().Error("No secret found for the service account",
			"namespace", serviceAccount.Namespace,
			"service-account-name", serviceAccount.Name,
			"reason", err)
		return nil, err
	}

	saSecret, err := opArgs.ClientSet().CoreV1().Secrets(opArgs.AppConfig().TargetNamespace).Get(
		context.Background(),
		serviceAccount.Secrets[0].Name,
		metav1.GetOptions{},
	)

	if err != nil {
		opArgs.Logger().Error("Failed fetching the secret for a service account",
			"namespace", serviceAccount.Namespace,
			"service-account-name", serviceAccount.Name,
			"service-account-secret-name", serviceAccount.Secrets[0].Name,
			"reason", err)
		return nil, err
	}

	return saSecret, nil
}

// GetSourceSecretField loads the CA certificate data from the source secret.
func GetSourceSecretField(opArgs OperationArgs) ([]byte, error) {

	s, err := opArgs.ClientSet().CoreV1().Secrets(opArgs.AppConfig().ServerTLSSecretNamespace).Get(
		context.Background(),
		opArgs.AppConfig().ServerTLSSecretName,
		metav1.GetOptions{},
	)
	if err != nil {
		opArgs.Logger().Error("Failed fetching a source secret",
			"namespace", opArgs.AppConfig().ServerTLSSecretNamespace,
			"service-account-name", opArgs.AppConfig().ServerTLSSecretName,
			"reason", err)
		return nil, err
	}

	field, ok := s.Data[opArgs.AppConfig().ServerTLSSecretCAKey]
	if !ok {
		err := fmt.Errorf("no '%s' key for tenant kubeconfig secret '%s'", opArgs.AppConfig().ServerTLSSecretCAKey, s.Name)
		opArgs.Logger().Error("Required secret CA key not found in secret",
			"namespace", opArgs.AppConfig().ServerTLSSecretNamespace,
			"service-account-name", opArgs.AppConfig().ServerTLSSecretName,
			"secret-ca-key", opArgs.AppConfig().ServerTLSSecretCAKey,
			"reason", err)
		return nil, err
	}

	return field, nil
}
