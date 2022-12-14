package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"

	"github.com/radekg/proxy-kubeconfig-generator/pkg/configuration"
	"github.com/radekg/proxy-kubeconfig-generator/pkg/generator"
	"github.com/radekg/proxy-kubeconfig-generator/pkg/utils"
)

var appConfig = new(configuration.Config)

func main() {

	flag.StringVar(&appConfig.ServiceAccountName, "serviceaccount", "", "The name of the service account for which to create the kubeconfig")
	flag.StringVar(&appConfig.Namespace, "namespace", configuration.DefaultNamespace, "(optional) The namespace of the service account and where the kubeconfig secret will be created.")
	flag.StringVar(&appConfig.Server, "server", "", "The server url of the kubeconfig where API requests will be sent")
	flag.StringVar(&appConfig.ServerTLSSecretNamespace, "server-tls-secret-namespace", configuration.DefaultNamespace, "(optional) The namespace of the server TLS secret.")
	flag.StringVar(&appConfig.ServerTLSSecretName, "server-tls-secret-name", "", "The server TLS secret name")
	flag.StringVar(&appConfig.ServerTLSSecretCAKey, "server-tls-secret-ca-key", configuration.DefaultTLSecretCAKey, "(optional) The CA key in the server TLS secret.")
	flag.StringVar(&appConfig.KubeConfigSecretKey, "kubeconfig-secret-key", configuration.DefaultKubeconfigSecretKey, "(optional) The key of the kubeconfig in the secret that will be created")
	flag.DurationVar(&appConfig.IterationInterval, "iteration-interval", configuration.DefaultIterationInterval, "(optional) How long to wait between iterations")

	flag.Parse()

	if err := appConfig.Validate(); err != nil {
		flag.Usage()
		panic(err)
	}

	config, err := utils.BuildClientConfig()
	if err != nil {
		panic(err)
	}

	clientset := kubernetes.NewForConfigOrDie(config)

	// Execute at least once:
	if err := runOnce(clientset, appConfig); err != nil {
		panic(err)
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	exitCtx, exitCtxCancelFunc := context.WithCancel(context.Background())

	go func() {
		for {
			select {
			case <-time.After(appConfig.IterationInterval):
				runOnce(clientset, appConfig)
			case <-exitCtx.Done():
				// we're done
				wg.Done()
				return
			}
		}
	}()

	sigintc := make(chan os.Signal, 1)
	signal.Notify(sigintc, os.Interrupt)

	go func() {
		<-sigintc
		// Handle exit
		exitCtxCancelFunc()
	}()

	wg.Wait()

}

func runOnce(clientset *kubernetes.Clientset, appConfig *configuration.Config) error {
	tenantConfig, err := generator.GenerateProxyKubeconfigFromSA(clientset, appConfig)
	if err != nil {
		return err
	}
	err = utils.CreateKubeconfigSecret(clientset, tenantConfig, appConfig)
	if err != nil {
		return err
	}
	return nil
}
