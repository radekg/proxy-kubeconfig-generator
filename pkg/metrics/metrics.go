package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/radekg/proxy-kubeconfig-generator/pkg/configuration"
)

var (
	secretSuccessTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "proxy_kubeconfig_generator_success_total",
		Help: "Proxy kubeconfig generator successful secret operation creation count",
	}, []string{"app_revision",
		"operation",
		"gen_service_account_name",
		"gen_source_secret_name",
		"gen_source_secret_namespace",
		"gen_target_secret_name",
		"gen_target_secret_namespace"})

	secretFailedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "proxy_kubeconfig_generator_failed_total",
		Help: "Proxy kubeconfig generator failed secret operation count",
	}, []string{"app_revision",
		"operation",
		"gen_service_account_name",
		"gen_source_secret_name",
		"gen_source_secret_namespace",
		"gen_target_secret_name",
		"gen_target_secret_namespace"})

	generatorRunCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "proxy_kubeconfig_generator_runs_total",
		Help: "Number of runs this generator executed",
	}, []string{"app_revision"})

	generatorNamespaceCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "proxy_kubeconfig_generator_namespaces_total",
		Help: "Number of namespaces configured for processing, instant",
	}, []string{"app_revision"})

	latencyNamespacesLoad = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "proxy_kubeconfig_generator_namespaces_load_ms",
		Help: "Kubernetes namespaces load latency in milliseconds",
	}, []string{"app_revision"})

	latencySourceSecretLoad = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "proxy_kubeconfig_generator_source_secret_load_ms",
		Help: "Source secret Kuberenets API get call latency",
	}, []string{"app_revision",
		"gen_source_secret_name",
		"gen_source_secret_namespace"})

	latencyTargetSecretOperation = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "proxy_kubeconfig_generator_target_secret_operation_ms",
		Help: "Target secret Kubernetes API operation call latency",
	}, []string{"app_revision",
		"operation",
		"gen_service_account_name",
		"gen_source_secret_name",
		"gen_source_secret_namespace",
		"gen_target_secret_name",
		"gen_target_secret_namespace"})
)

func RecordCreateSuccess(appConfig *configuration.Config, namespace string) {
	secretSuccessTotal.WithLabelValues(
		configuration.AppRevision(),
		"create",
		appConfig.ServiceAccountName,
		appConfig.ServerTLSSecretName,
		appConfig.ServerTLSSecretNamespace,
		appConfig.TenantSecretName(),
		namespace).Inc()
}

func RecordCreateFailure(appConfig *configuration.Config, namespace string) {
	secretFailedTotal.WithLabelValues(
		configuration.AppRevision(),
		"create",
		appConfig.ServiceAccountName,
		appConfig.ServerTLSSecretName,
		appConfig.ServerTLSSecretNamespace,
		appConfig.TenantSecretName(),
		namespace).Inc()
}

func RecordUpdateSuccess(appConfig *configuration.Config, namespace string) {
	secretSuccessTotal.WithLabelValues(
		configuration.AppRevision(),
		"update",
		appConfig.ServiceAccountName,
		appConfig.ServerTLSSecretName,
		appConfig.ServerTLSSecretNamespace,
		appConfig.TenantSecretName(),
		namespace).Inc()
}

func RecordUpdateFailure(appConfig *configuration.Config, namespace string) {
	secretFailedTotal.WithLabelValues(
		configuration.AppRevision(),
		"update",
		appConfig.ServiceAccountName,
		appConfig.ServerTLSSecretName,
		appConfig.ServerTLSSecretNamespace,
		appConfig.TenantSecretName(),
		namespace).Inc()
}

func RecordNamespaceCount(count float64) {
	generatorNamespaceCount.WithLabelValues(
		configuration.AppRevision()).Set(count)
}

func RecordRunCount() {
	generatorRunCount.WithLabelValues(
		configuration.AppRevision()).Inc()
}

func RecordNamespaceLoadLatency(value float64) {
	latencyNamespacesLoad.WithLabelValues(
		configuration.AppRevision()).Observe(value)
}

func RecordSourceSecretLoadLatency(appConfig *configuration.Config, value float64) {
	latencySourceSecretLoad.WithLabelValues(
		configuration.AppRevision(),
		appConfig.ServerTLSSecretName,
		appConfig.ServerTLSSecretNamespace).Observe(value)
}

func RecordTargetSecretCreateLatency(appConfig *configuration.Config, namespace string, value float64) {
	latencyTargetSecretOperation.WithLabelValues(
		configuration.AppRevision(),
		"create",
		appConfig.ServiceAccountName,
		appConfig.ServerTLSSecretName,
		appConfig.ServerTLSSecretNamespace,
		appConfig.TenantSecretName(),
		namespace).Observe(value)
}

func RecordTargetSecretUpdateLatency(appConfig *configuration.Config, namespace string, value float64) {
	latencyTargetSecretOperation.WithLabelValues(
		configuration.AppRevision(),
		"update",
		appConfig.ServiceAccountName,
		appConfig.ServerTLSSecretName,
		appConfig.ServerTLSSecretNamespace,
		appConfig.TenantSecretName(),
		namespace).Observe(value)
}
