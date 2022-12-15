package k8s

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-hclog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func BuildClientConfig(logger hclog.Logger) (*rest.Config, error) {
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

func GetServiceAccountSecret(opArgs OperationArgs) (*corev1.Secret, error) {
	serviceAccount, err := opArgs.ClientSet().CoreV1().ServiceAccounts(opArgs.AppConfig().TargetNamespace).Get(
		context.Background(),
		opArgs.AppConfig().ServiceAccountName,
		metav1.GetOptions{},
	)
	if err != nil {
		return nil, err
	}

	if len(serviceAccount.Secrets) < 1 {
		return nil, fmt.Errorf("no secret found for the service account %s in namepsace %s", serviceAccount.Name, serviceAccount.Namespace)
	}

	saSecret, err := opArgs.ClientSet().CoreV1().Secrets(opArgs.AppConfig().TargetNamespace).Get(
		context.Background(),
		serviceAccount.Secrets[0].Name,
		metav1.GetOptions{},
	)
	if err != nil {
		return nil, err
	}

	return saSecret, nil
}

func BuildKubeconfigFromToken(token []byte, CACertificate []byte, opArgs OperationArgs) (*clientcmdapi.Config, error) {
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
		return nil, err
	}

	return &config, nil
}

func CreateKubeconfigSecret(opArgs OperationArgs, kubeconfig *clientcmdapi.Config) error {
	configBuffer, err := clientcmd.Write(*kubeconfig)
	if err != nil {
		return err
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: opArgs.AppConfig().TenantSecretName(),
		},
		Data: map[string][]byte{
			opArgs.AppConfig().KubeConfigSecretKey: configBuffer,
		},
	}
	if opArgs.AppConfig().ReportOnly {
		opArgs.Logger().Info("Report only: would create a secret", "namespace", opArgs.AppConfig().TargetNamespace)
	}
	_, err = opArgs.ClientSet().CoreV1().Secrets(opArgs.AppConfig().TargetNamespace).Create(
		context.Background(),
		secret,
		metav1.CreateOptions{},
	)
	if err != nil {
		return err
	}

	return nil
}

func GetSecretField(opArgs OperationArgs) ([]byte, error) {
	s, err := opArgs.ClientSet().CoreV1().Secrets(opArgs.AppConfig().ServerTLSSecretNamespace).Get(
		context.Background(),
		opArgs.AppConfig().ServerTLSSecretName,
		metav1.GetOptions{},
	)
	if err != nil {
		return nil, err
	}

	field, ok := s.Data[opArgs.AppConfig().ServerTLSSecretCAKey]
	if !ok {
		return nil, fmt.Errorf("no %s key for tenant kubeconfig secret %s", opArgs.AppConfig().ServerTLSSecretCAKey, s.Name)
	}

	return field, nil
}

func BuildClientConfigFromSecret(opArgs OperationArgs) (*rest.Config, error) {
	o, err := GetSecretField(opArgs)
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
