package configuration

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
)

const (
	DefaultTLSecretCAKey       = "ca"
	DefaultNamespace           = "default"
	DefaultKubeconfigSecretKey = "kubeconfig"
	DefaultIterationInterval   = time.Second * 10
)

type Config struct {
	ServiceAccountName       string
	Namespace                string
	Server                   string
	ServerTLSSecretNamespace string
	ServerTLSSecretName      string
	ServerTLSSecretCAKey     string
	KubeConfigSecretKey      string
	IterationInterval        time.Duration
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
