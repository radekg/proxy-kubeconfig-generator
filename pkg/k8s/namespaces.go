package k8s

import (
	"context"
	"time"

	"github.com/radekg/proxy-kubeconfig-generator/pkg/metrics"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func FindNamespaces(ctx context.Context, opArgs OperationArgs) (*corev1.NamespaceList, error) {
	// handle latency metric
	benchStart := time.Now().UTC().UnixMilli()
	defer func() {
		metrics.RecordNamespaceLoadLatency(float64(time.Now().UTC().UnixMilli() - benchStart))
	}()
	// execute call
	return opArgs.ClientSet().CoreV1().Namespaces().List(ctx, metav1.ListOptions{
		LabelSelector: opArgs.AppConfig().TargetNamespaceSelector.String(),
	})
}
