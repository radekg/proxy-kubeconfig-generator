package server

import (
	"context"
	"net/http"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/radekg/proxy-kubeconfig-generator/pkg/configuration"
	"github.com/radekg/proxy-kubeconfig-generator/pkg/errors"
	"github.com/radekg/proxy-kubeconfig-generator/pkg/utils"

	// import metrics so that counters are initialized regardless
	_ "github.com/radekg/proxy-kubeconfig-generator/pkg/metrics"
)

// Runner represents a snapshot coordinator runner.
type Runner interface {
	Close()
	Start(appConfig *configuration.Config) utils.StartStatus
}

// NewDefaultRunner returns an unconfigured and not started
// instance of a default runner.
func NewDefaultRunner(logger hclog.Logger) Runner {
	return &defaultRunner{
		logger: logger,
	}
}

type defaultRunner struct {
	logger hclog.Logger

	ctxCancelFunc context.CancelFunc
}

func (r *defaultRunner) Close() {
	if r.ctxCancelFunc != nil {
		r.ctxCancelFunc()
		r.ctxCancelFunc = nil
	}
}

// Start starts this server instance.
func (r *defaultRunner) Start(appConfig *configuration.Config) utils.StartStatus {

	status := utils.NewDefaultStartStatus()

	if r.ctxCancelFunc != nil {
		status.ReportError(errors.ErrServerAlreadyRunning)
		return status
	}

	_, cancelFunc := context.WithCancel(context.Background())
	r.ctxCancelFunc = cancelFunc

	go func() {
		r.logger.Info("Starting server with bind address",
			"address", appConfig.MetricsBindHostPort)
		chanListenErr := make(chan error, 1)

		http.HandleFunc(appConfig.URIPathHealth, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		http.Handle(appConfig.URIPathMetrics, promhttp.Handler())

		go func() {
			r.logger.Info("Starting server without TLS")
			if err := http.ListenAndServe(appConfig.MetricsBindHostPort, nil); err != nil {
				chanListenErr <- err
			}

		}()
		select {
		case err := <-chanListenErr:
			r.logger.Error("Server failed to start", "reason", err)
			status.ReportError(err)
		case <-time.After(time.Millisecond * 500):
			r.logger.Info("Server running and serving",
				"bind-host-port", appConfig.MetricsBindHostPort)
			status.ReportSuccess()
		}
	}()

	return status
}
