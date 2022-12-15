package k8s

import (
	"github.com/hashicorp/go-hclog"
	"github.com/radekg/proxy-kubeconfig-generator/pkg/configuration"
	"k8s.io/client-go/kubernetes"
)

type OperationArgs interface {
	AppConfig() *configuration.Config
	ClientSet() *kubernetes.Clientset
	Logger() hclog.Logger
}

type defaultOperationArgs struct {
	appConfig *configuration.Config
	clientSet *kubernetes.Clientset
	logger    hclog.Logger
}

func NewDefaultOperationArgs(appConfig *configuration.Config, clientSet *kubernetes.Clientset, logger hclog.Logger) OperationArgs {
	return &defaultOperationArgs{
		appConfig: appConfig,
		clientSet: clientSet,
		logger:    logger,
	}
}

func (v *defaultOperationArgs) AppConfig() *configuration.Config {
	return v.appConfig
}

func (v *defaultOperationArgs) ClientSet() *kubernetes.Clientset {
	return v.clientSet
}

func (v *defaultOperationArgs) Logger() hclog.Logger {
	return v.logger
}
