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
	"github.com/radekg/proxy-kubeconfig-generator/pkg/k8s"
	"github.com/radekg/proxy-kubeconfig-generator/pkg/metrics"
	"github.com/radekg/proxy-kubeconfig-generator/pkg/server"
)

var appConfig *configuration.Config
var httpConfig *configuration.HttpConfig
var logConfig *configuration.LogConfig

func initFlags() {
	// Main program flags
	flag.StringVar(&appConfig.ServiceAccountName, "serviceaccount", "", "The name of the service account for which to create the kubeconfig")
	flag.StringVar(&appConfig.NamespaceFromCLI, "namespace", configuration.DefaultNamespace, "(optional) The namespace of the service account and where the kubeconfig secret will be created, ignored when selectors are in use")
	flag.Var(&appConfig.TargetNamespaceSelector, "namespace-label-selector", "(optional) The namespace of the service account and where the kubeconfig secret will be created")
	flag.StringVar(&appConfig.Server, "server", "", "The server url of the kubeconfig where API requests will be sent")
	flag.StringVar(&appConfig.ServerTLSSecretNamespace, "server-tls-secret-namespace", configuration.DefaultNamespace, "(optional) The namespace of the server TLS secret")
	flag.StringVar(&appConfig.ServerTLSSecretName, "server-tls-secret-name", "", "The server TLS secret name")
	flag.StringVar(&appConfig.ServerTLSSecretCAKey, "server-tls-secret-ca-key", configuration.DefaultTLSecretCAKey, "(optional) The CA key in the server TLS secret")
	flag.StringVar(&appConfig.KubeConfigSecretKey, "kubeconfig-secret-key", configuration.DefaultKubeConfigSecretKey, "(optional) The key of the kubeconfig in the secret that will be created")
	flag.StringVar(&appConfig.SourceSecretRevisionLabel, "source-secret-revision-label", configuration.DefaultSourceSecretResourceVersionLabel, "(optional) Label of the target secret where the last know source secret resource version is stored")
	flag.DurationVar(&appConfig.IterationInterval, "iteration-interval", configuration.DefaultIterationInterval, "(optional) How long to wait between iterations")
	flag.BoolVar(&appConfig.DisallowUpdates, "disallow-updates", false, "(optional) When set, program does not update existing secrets")
	flag.BoolVar(&appConfig.ReportOnly, "report-only", false, "(optional) When set, program does not mutate anything, only logs what would have been done")
	// Logging flags
	flag.StringVar(&logConfig.LogLevel, "log-level", "info", "Log level")
	flag.BoolVar(&logConfig.LogAsJSON, "log-as-json", false, "Log as JSON")
	flag.BoolVar(&logConfig.LogColor, "log-color", false, "Log color")
	flag.BoolVar(&logConfig.LogForceColor, "log-force-color", false, "Force log color output")
	// HTTP flags
	flag.StringVar(&httpConfig.MetricsBindHostPort, "metrics-server-bind-host-port", ":10000", "Host port to bind the metrics server on")
	flag.StringVar(&httpConfig.URIPathHealth, "uri-path-health", "/health", "URI path at which the health endpoint responds")
	flag.StringVar(&httpConfig.URIPathMetrics, "uri-path-metrics", "/metrics", "URI path at which the metrics endpoint responds")
}

func initConfigs() {
	appConfig = &configuration.Config{
		TargetNamespaceSelector: configuration.NamespaceSelectorLabels{
			Values: []string{},
		},
	}
	httpConfig = new(configuration.HttpConfig)
	logConfig = new(configuration.LogConfig)
}

func init() {
	initConfigs()
	initFlags()
}

func main() {
	flag.Parse()
	os.Exit(program())
}

func program() int {

	appLogger := logConfig.NewLogger("generator")

	if err := appConfig.Validate(); err != nil {
		flag.Usage()
		appLogger.Error("Invalid configuration", "reason", err)
		return 1
	}

	config, err := k8s.BuildKubernetesClientConfig(appLogger)
	if err != nil {
		appLogger.Error("Failed building client configuration", "reason", err)
		return 1
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		appLogger.Error("Failed building new Kubernetes client", "reason", err)
		return 1
	}

	serverRunner := server.NewDefaultRunner(appLogger.Named("server"))
	status := serverRunner.Start(httpConfig)
	select {
	case <-status.OnStarted():
	case err := <-status.OnError():
		appLogger.Error("Server did not start", "reason", err)
		return 1
	}

	opArgs := k8s.NewDefaultOperationArgs(appConfig, clientset, appLogger)

	wg := sync.WaitGroup{}
	wg.Add(1)
	exitCtx, exitCtxCancelFunc := context.WithCancel(context.Background())

	if len(appConfig.TargetNamespaceSelector.Values) > 0 {
		appLogger.Info("Namespace selectors defined, going to use namespace discovery instead of the --namespace value")
	}

	// Execute at least once:
	if errs := runOnce(exitCtx, opArgs); len(errs) > 0 {
		for ns, err := range errs {
			appLogger.Error("kubeconfig generator failed to generate", "namespace", ns, "reason", err)
		}
	}

	go func() {
		for {
			select {
			case <-time.After(appConfig.IterationInterval):
				if errs := runOnce(exitCtx, opArgs); len(errs) > 0 {
					for ns, err := range errs {
						appLogger.Error("kubeconfig generator failed to generate", "namespace", ns, "reason", err)
					}
				}
			case <-exitCtx.Done():
				appLogger.Info("stopping run loop")
				wg.Done()
				return
			}
		}
	}()

	sigintc := make(chan os.Signal, 1)
	signal.Notify(sigintc, os.Interrupt)

	go func() {
		<-sigintc
		appLogger.Info("sigint handled, going to stop")
		exitCtxCancelFunc()
	}()

	wg.Wait()

	appLogger.Info("all done")

	return 0
}

func runOnce(ctx context.Context, opArgs k8s.OperationArgs) map[string]error {
	errors := map[string]error{}
	namespaces := []string{opArgs.AppConfig().NamespaceFromCLI}
	if len(appConfig.TargetNamespaceSelector.Values) > 0 {
		namespaceList, err := k8s.FindNamespaces(ctx, opArgs)
		if err != nil {
			opArgs.Logger().Error("Failed loading namespace list", "reason", err)
		} else {
			namespaces = []string{}
			for _, ns := range namespaceList.Items {
				namespaces = append(namespaces, ns.Name)
			}
			opArgs.Logger().Info("Discovered namespaces",
				"number-of-namespaces", len(namespaces),
				"namespaces", namespaces,
				"selectors", appConfig.TargetNamespaceSelector.Values)
		}
	}

	metrics.RecordNamespaceCount(float64(len(namespaces)))

	for _, ns := range namespaces {
		sourceSecret, tenantConfig, err := generator.GenerateProxyKubeConfigFromSA(ctx, ns, opArgs)
		if err != nil { // Logging taken care of.
			errors[ns] = err
			continue
		}
		err = k8s.CreateOrUpdateKubeConfigSecret(ctx, ns, opArgs, tenantConfig, sourceSecret)
		if err != nil { // Logging taken care of.
			errors[ns] = err
			continue
		}
	}

	metrics.RecordRunCount()

	return errors
}
