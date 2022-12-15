package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/radekg/proxy-kubeconfig-generator/pkg/configuration"
)

var (
	generatorSuccessCounters = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "proxy_kubeconfig_generator_success",
		Help: "Proxy kubeconfig generator success counts",
	}, []string{"gen_service_account_name",
		"gen_secret_name",
		"gen_secret_namespace",
		"gen_target_namespace"})

	generatorFailureCounters = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "proxy_kubeconfig_generator_failure",
		Help: "Proxy kubeconfig generator failure counts",
	}, []string{"gen_service_account_name",
		"gen_secret_name",
		"gen_secret_namespace",
		"gen_target_namespace"})
)

func init() {
	prometheus.Register(generatorSuccessCounters)
	prometheus.Register(generatorFailureCounters)
}

func RecordSuccess(appConfig *configuration.Config) {
	generatorSuccessCounters.WithLabelValues(appConfig.ServiceAccountName,
		appConfig.ServerTLSSecretName,
		appConfig.TargetNamespace,
		appConfig.TenantSecretName()).Inc()
}

func RecordFailure(appConfig *configuration.Config) {
	generatorFailureCounters.WithLabelValues(appConfig.ServiceAccountName,
		appConfig.ServerTLSSecretName,
		appConfig.TargetNamespace,
		appConfig.TenantSecretName()).Inc()
}
