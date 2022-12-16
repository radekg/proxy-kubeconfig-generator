package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/radekg/proxy-kubeconfig-generator/pkg/configuration"
)

var (
	secretCreateSuccessTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "proxy_kubeconfig_generator_create_success_total",
		Help: "Proxy kubeconfig generator successful secret creation count",
	}, []string{"app_revision",
		"gen_service_account_name",
		"gen_source_secret_name",
		"gen_source_secret_namespace",
		"gen_target_secret_name",
		"gen_target_secret_namespace"})

	secretCreateFailedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "proxy_kubeconfig_generator_create_failed_total",
		Help: "Proxy kubeconfig generator failed secret creation count",
	}, []string{"app_revision",
		"gen_service_account_name",
		"gen_source_secret_name",
		"gen_source_secret_namespace",
		"gen_target_secret_name",
		"gen_target_secret_namespace"})

	secretUpdateSuccessTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "proxy_kubeconfig_generator_update_success_total",
		Help: "Proxy kubeconfig generator successful secret update count",
	}, []string{"app_revision",
		"gen_service_account_name",
		"gen_source_secret_name",
		"gen_source_secret_namespace",
		"gen_target_secret_name",
		"gen_target_secret_namespace"})

	secretUpdateFailedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "proxy_kubeconfig_generator_update_failed_total",
		Help: "Proxy kubeconfig generator failed secret update count",
	}, []string{"app_revision",
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

	latencyTargetSecretCreate = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "proxy_kubeconfig_generator_target_secret_create_ms",
		Help: "Target secret Kubernetes API create call latency",
	}, []string{"app_revision",
		"gen_service_account_name",
		"gen_source_secret_name",
		"gen_source_secret_namespace",
		"gen_target_secret_name",
		"gen_target_secret_namespace"})

	latencyTargetSecretUpdate = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "proxy_kubeconfig_generator_target_secret_update_ms",
		Help: "Target secret Kubernetes API update call latency",
	}, []string{"app_revision",
		"gen_service_account_name",
		"gen_source_secret_name",
		"gen_source_secret_namespace",
		"gen_target_secret_name",
		"gen_target_secret_namespace"})
)

func init() {

	prometheus.MustRegister(secretCreateSuccessTotal, secretCreateFailedTotal)
	prometheus.MustRegister(secretUpdateSuccessTotal, secretUpdateFailedTotal)
	prometheus.MustRegister(generatorNamespaceCount, generatorRunCount)
	prometheus.MustRegister(latencyNamespacesLoad,
		latencySourceSecretLoad,
		latencyTargetSecretCreate,
		latencyTargetSecretUpdate)

}

func RecordCreateSuccess(appConfig *configuration.Config, namespace string) {
	secretCreateSuccessTotal.WithLabelValues(
		configuration.AppRevision(),
		appConfig.ServiceAccountName,
		appConfig.ServerTLSSecretName,
		appConfig.ServerTLSSecretNamespace,
		appConfig.TenantSecretName(),
		namespace).Inc()
}

func RecordCreateFailure(appConfig *configuration.Config, namespace string) {
	secretCreateFailedTotal.WithLabelValues(
		configuration.AppRevision(),
		appConfig.ServiceAccountName,
		appConfig.ServerTLSSecretName,
		appConfig.ServerTLSSecretNamespace,
		appConfig.TenantSecretName(),
		namespace).Inc()
}

func RecordUpdateSuccess(appConfig *configuration.Config, namespace string) {
	secretUpdateSuccessTotal.WithLabelValues(
		configuration.AppRevision(),
		appConfig.ServiceAccountName,
		appConfig.ServerTLSSecretName,
		appConfig.ServerTLSSecretNamespace,
		appConfig.TenantSecretName(),
		namespace).Inc()
}

func RecordUpdateFailure(appConfig *configuration.Config, namespace string) {
	secretUpdateFailedTotal.WithLabelValues(
		configuration.AppRevision(),
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
	latencyTargetSecretCreate.WithLabelValues(
		configuration.AppRevision(),
		appConfig.ServiceAccountName,
		appConfig.ServerTLSSecretName,
		appConfig.ServerTLSSecretNamespace,
		appConfig.TenantSecretName(),
		namespace).Observe(value)
}

func RecordTargetSecretUpdateLatency(appConfig *configuration.Config, namespace string, value float64) {
	latencyTargetSecretUpdate.WithLabelValues(
		configuration.AppRevision(),
		appConfig.ServiceAccountName,
		appConfig.ServerTLSSecretName,
		appConfig.ServerTLSSecretNamespace,
		appConfig.TenantSecretName(),
		namespace).Observe(value)
}
