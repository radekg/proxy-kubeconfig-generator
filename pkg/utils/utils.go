package utils

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func BuildClientConfig() (*rest.Config, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	config, err := kubeConfig.ClientConfig()
	if err != nil {
		fmt.Printf("error while trying to create a client config: %s", err)

		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}

		return config, nil
	}

	return config, nil
}

func GetServiceAccountSecret(clientSet *kubernetes.Clientset, serviceAccountName string, namespace string) (*corev1.Secret, error) {
	serviceAccount, err := clientSet.CoreV1().ServiceAccounts(namespace).Get(
		context.Background(),
		serviceAccountName,
		metav1.GetOptions{},
	)
	if err != nil {
		return nil, err
	}

	if len(serviceAccount.Secrets) < 1 {
		return nil, fmt.Errorf("no secret found for the service account %s in namepsace %s", serviceAccount.Name, serviceAccount.Namespace)
	}

	saSecret, err := clientSet.CoreV1().Secrets(namespace).Get(
		context.Background(),
		serviceAccount.Secrets[0].Name,
		metav1.GetOptions{},
	)
	if err != nil {
		return nil, err
	}

	return saSecret, nil
}

func BuildKubeconfigFromToken(token []byte, CACertificate []byte, server string, namespace string) (*clientcmdapi.Config, error) {
	clusters := make(map[string]*clientcmdapi.Cluster)
	clusters["default"] = &clientcmdapi.Cluster{
		Server:                   server,
		CertificateAuthorityData: CACertificate,
	}

	contexts := make(map[string]*clientcmdapi.Context)
	contexts["default"] = &clientcmdapi.Context{
		Cluster:   "default",
		Namespace: namespace,
		AuthInfo:  namespace,
	}

	authinfos := make(map[string]*clientcmdapi.AuthInfo)
	authinfos[namespace] = &clientcmdapi.AuthInfo{
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

func CreateKubeconfigSecret(clientset *kubernetes.Clientset, kubeconfig *clientcmdapi.Config, namespace string, secretName string, secretKey string) error {
	configBuffer, err := clientcmd.Write(*kubeconfig)
	if err != nil {
		return err
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: secretName,
		},
		Data: map[string][]byte{
			secretKey: configBuffer,
		},
	}
	_, err = clientset.CoreV1().Secrets(namespace).Create(
		context.Background(),
		secret,
		metav1.CreateOptions{},
	)
	if err != nil {
		return err
	}

	return nil
}

func GetSecretField(clientset *kubernetes.Clientset, name string, key string, namespace string) ([]byte, error) {
	s, err := clientset.CoreV1().Secrets(namespace).Get(
		context.Background(),
		name,
		metav1.GetOptions{},
	)
	if err != nil {
		return nil, err
	}

	field, ok := s.Data[key]
	if !ok {
		return nil, fmt.Errorf("no %s key for tenant kubeconfig secret %s", key, s.Name)
	}

	return field, nil
}

func BuildClientConfigFromSecret(clientset *kubernetes.Clientset, name string, key string, namespace string) (*rest.Config, error) {
	o, err := GetSecretField(clientset, name, key, namespace)
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
