package configuration

import (
	"fmt"
	"time"
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
