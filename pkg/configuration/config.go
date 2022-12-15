package configuration

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
)

const (
	// DefaultTLSecretCAKey is the default TLS secret CA key.
	DefaultTLSecretCAKey = "ca.crt"
	// DefaultNamespace is the default target namespace.
	DefaultNamespace = "default"
	// DefaultKubeConfigSecretKey is the default kubeconfig secret key.
	DefaultKubeConfigSecretKey = "kubeconfig"
	// DefaultSourceSecretResourceVersionLabel label of the target secret where the last know source secret resource version is stored.
	DefaultSourceSecretResourceVersionLabel = "proxy-kubeconfig-generator/last-known-source-resource-version"
	// DefaultIterationInterval is the default interval between individual iterations.
	DefaultIterationInterval = time.Second * 60
)

type Config struct {
	NamespaceFromCLI          string
	ServiceAccountName        string
	TargetNamespaceSelector   NamespaceSelectorLabels
	Server                    string
	ServerTLSSecretNamespace  string
	ServerTLSSecretName       string
	ServerTLSSecretCAKey      string
	KubeConfigSecretKey       string
	SourceSecretRevisionLabel string
	IterationInterval         time.Duration

	DisallowUpdates bool
	ReportOnly      bool
}

func (c *Config) TenantSecretName() string {
	return fmt.Sprintf("%s-kubeconfig", c.ServiceAccountName)
}

func (c *Config) Validate() error {
	if c.ServiceAccountName == "" {
		return fmt.Errorf("missing service account name")
	}

	if c.Server == "" {
		return fmt.Errorf("missing server url")
	}

	if c.ServerTLSSecretName == "" {
		return fmt.Errorf("missing server TLS secret name")
	}
	return nil
}

type HttpConfig struct {
	MetricsBindHostPort string
	URIPathMetrics      string
	URIPathHealth       string
}

type LogConfig struct {
	LogLevel      string
	LogColor      bool
	LogForceColor bool
	LogAsJSON     bool
}

// NewLogger returns a new configured logger.
func (c *LogConfig) NewLogger(name string) hclog.Logger {
	loggerColorOption := hclog.ColorOff
	if c.LogColor {
		loggerColorOption = hclog.AutoColor
	}
	if c.LogForceColor {
		loggerColorOption = hclog.ForceColor
	}

	return hclog.New(&hclog.LoggerOptions{
		Name:       name,
		Level:      hclog.LevelFromString(c.LogLevel),
		Color:      loggerColorOption,
		JSONFormat: c.LogAsJSON,
	})
}
