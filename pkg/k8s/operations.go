package k8s

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/radekg/proxy-kubeconfig-generator/pkg/metrics"
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
func CreateOrUpdateKubeConfigSecret(ctx context.Context, targetNamespace string, opArgs OperationArgs, kubeconfig *clientcmdapi.Config, sourceSecret *corev1.Secret) error {

	existingSecret, err := opArgs.ClientSet().CoreV1().Secrets(targetNamespace).Get(
		ctx,
		opArgs.AppConfig().TenantSecretName(),
		metav1.GetOptions{})

	if err != nil {
		if !apiErrors.IsAlreadyExists(err) {
			opArgs.Logger().Error("Failed checking if secret exists",
				"namespace", targetNamespace,
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

	sourceResourceRevision := fmt.Sprintf("%s_%d", sourceSecret.ResourceVersion, sourceSecret.Generation)

	if existingSecret != nil {

		if opArgs.AppConfig().DisallowUpdates {
			opArgs.Logger().Info("Secret exists and updates are disabled",
				"namespace", targetNamespace,
				"secret-name", opArgs.AppConfig().TenantSecretName(),
				"existing-secret-generation", existingSecret.Generation,
				"existing-secret-resource-version", existingSecret.ResourceVersion,
				"source-secret-resource-version", sourceResourceRevision)
			return nil
		}

		existingSecretSourceVersion := "<not set>"

		if targetLastKnownRevision, labelExists := existingSecret.Labels[opArgs.AppConfig().SourceSecretRevisionLabel]; labelExists {
			if targetLastKnownRevision == sourceResourceRevision {
				opArgs.Logger().Info("Nothing to do, secret already exists and and was generated using current source secret resource version",
					"namespace", targetNamespace,
					"secret-name", opArgs.AppConfig().TenantSecretName(),
					"source-secret-resource-version", sourceResourceRevision)
				return nil
			}
			existingSecretSourceVersion = targetLastKnownRevision
		}

		opArgs.Logger().Info("Secret will be updated",
			"namespace", targetNamespace,
			"secret-name", opArgs.AppConfig().TenantSecretName(),
			"secret-key", opArgs.AppConfig().KubeConfigSecretKey,
			"source-secret-resource-version", sourceResourceRevision,
			"existing-secret-generation", existingSecret.Generation,
			"existing-secret-resource-version", existingSecret.ResourceVersion,
			"secret-data-size", len(configBuffer))

		secretToUpdate := existingSecret.DeepCopy()
		if secretToUpdate.Labels == nil {
			secretToUpdate.Labels = map[string]string{}
		}
		if secretToUpdate.Data == nil {
			secretToUpdate.Data = map[string][]byte{}
		}

		secretToUpdate.Labels[opArgs.AppConfig().SourceSecretRevisionLabel] = sourceResourceRevision
		secretToUpdate.Data[opArgs.AppConfig().KubeConfigSecretKey] = configBuffer

		if opArgs.AppConfig().ReportOnly {
			opArgs.Logger().Info("Report only: would update a secret",
				"namespace", targetNamespace,
				"secret-name", opArgs.AppConfig().TenantSecretName(),
				"secret-key", opArgs.AppConfig().KubeConfigSecretKey,
				"source-secret-resource-version", sourceResourceRevision,
				"secret-data-size", len(configBuffer))
			return nil
		}

		updateBenchStart := time.Now().UTC().UnixMilli()

		_, err = opArgs.ClientSet().CoreV1().Secrets(targetNamespace).Update(
			ctx,
			secretToUpdate,
			metav1.UpdateOptions{},
		)

		metrics.RecordTargetSecretUpdateLatency(opArgs.AppConfig(),
			targetNamespace,
			float64(time.Now().UnixMilli()-updateBenchStart))

		if err != nil {

			opArgs.Logger().Error("Failed updating secret",
				"target-namespace", opArgs.AppConfig().TenantSecretName(),
				"source-secret-resource-version", existingSecret.Generation,
				"source-secret-resource-version", sourceResourceRevision,
				"secret-key", opArgs.AppConfig().KubeConfigSecretKey,
				"reason", err)
			return err
		}

		opArgs.Logger().Info("Secret updated",
			"target-namespace", opArgs.AppConfig().TenantSecretName(),
			"secret-name", opArgs.AppConfig().TenantSecretName(),
			"secret-key", opArgs.AppConfig().KubeConfigSecretKey,
			"old-resource-version", existingSecretSourceVersion,
			"new-resource-version", sourceResourceRevision)

		return nil // end of update

	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: opArgs.AppConfig().TenantSecretName(),
			Labels: map[string]string{
				opArgs.AppConfig().SourceSecretRevisionLabel: sourceResourceRevision,
			},
		},
		Data: map[string][]byte{
			opArgs.AppConfig().KubeConfigSecretKey: configBuffer,
		},
	}

	if opArgs.AppConfig().ReportOnly {
		opArgs.Logger().Info("Report only: would create a secret",
			"namespace", targetNamespace,
			"secret-name", opArgs.AppConfig().TenantSecretName(),
			"secret-key", opArgs.AppConfig().KubeConfigSecretKey,
			"source-secret-resource-version", sourceResourceRevision,
			"secret-data-size", len(configBuffer))
		return nil
	}

	createBenchStart := time.Now().UTC().UnixMilli()

	_, err = opArgs.ClientSet().CoreV1().Secrets(targetNamespace).Create(
		ctx,
		secret,
		metav1.CreateOptions{},
	)

	metrics.RecordTargetSecretCreateLatency(opArgs.AppConfig(),
		targetNamespace,
		float64(time.Now().UnixMilli()-createBenchStart))

	if err != nil {
		metrics.RecordCreateFailure(opArgs.AppConfig(), targetNamespace)
		opArgs.Logger().Error("Failed creating secret",
			"target-namespace", opArgs.AppConfig().TenantSecretName(),
			"secret-key", opArgs.AppConfig().KubeConfigSecretKey,
			"reason", err)
		return err
	}

	metrics.RecordCreateSuccess(opArgs.AppConfig(), targetNamespace)
	opArgs.Logger().Info("Secret created",
		"target-namespace", opArgs.AppConfig().TenantSecretName(),
		"secret-name", opArgs.AppConfig().TenantSecretName(),
		"secret-key", opArgs.AppConfig().KubeConfigSecretKey,
		"source-secret-resource-version", sourceResourceRevision)

	return nil
}

// GetServiceAccountSecret retrieves a secret for the service account.
func GetServiceAccountSecret(ctx context.Context, targetNamespace string, opArgs OperationArgs) (*corev1.Secret, error) {

	serviceAccount, err := opArgs.ClientSet().CoreV1().ServiceAccounts(targetNamespace).Get(
		ctx,
		opArgs.AppConfig().ServiceAccountName,
		metav1.GetOptions{})

	if err != nil {
		opArgs.Logger().Error("Problem fetching service account",
			"namespace", targetNamespace,
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

	saSecret, err := opArgs.ClientSet().CoreV1().Secrets(targetNamespace).Get(
		ctx,
		serviceAccount.Secrets[0].Name,
		metav1.GetOptions{})

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

// GetSourceSecret loads the source secret.
func GetSourceSecret(opArgs OperationArgs) (*corev1.Secret, error) {
	benchStart := time.Now().UTC().UnixMilli()
	defer func() {
		metrics.RecordSourceSecretLoadLatency(opArgs.AppConfig(),
			float64(time.Now().UTC().UnixMilli()-benchStart))
	}()
	s, err := opArgs.ClientSet().CoreV1().Secrets(opArgs.AppConfig().ServerTLSSecretNamespace).Get(
		context.Background(),
		opArgs.AppConfig().ServerTLSSecretName,
		metav1.GetOptions{})
	if err != nil {
		opArgs.Logger().Error("Failed fetching a source secret",
			"namespace", opArgs.AppConfig().ServerTLSSecretNamespace,
			"service-account-name", opArgs.AppConfig().ServerTLSSecretName,
			"reason", err)
		return nil, err
	}
	return s, nil
}

// GetSourceSecretField loads the CA certificate data from the source secret.
func GetSourceSecretField(secret *corev1.Secret, opArgs OperationArgs) ([]byte, error) {
	field, ok := secret.Data[opArgs.AppConfig().ServerTLSSecretCAKey]
	if !ok {
		err := fmt.Errorf("no '%s' key for tenant kubeconfig secret '%s'", opArgs.AppConfig().ServerTLSSecretCAKey, secret.Name)
		opArgs.Logger().Error("Required secret CA key not found in secret",
			"namespace", opArgs.AppConfig().ServerTLSSecretNamespace,
			"service-account-name", opArgs.AppConfig().ServerTLSSecretName,
			"secret-ca-key", opArgs.AppConfig().ServerTLSSecretCAKey,
			"reason", err)
		return nil, err
	}

	return field, nil
}
