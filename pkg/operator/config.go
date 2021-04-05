package operator

import (
	"flag"

	"github.com/grafana/agent/pkg/util/server"
)

// Config configures the operator.
type Config struct {
	Server server.Config

	// APIServer to connect to for local testing. Not recommended for production.
	APIServer APIServer
}

// RegisterFlags registers flags to the provided FlagSet.
func (c *Config) RegisterFlags(fs *flag.FlagSet) {
	c.Server.RegisterFlags(fs)
	c.APIServer.RegisterFlags(fs)
}

// APIServer controls development flags for connecting to the Kubernetes API server.
type APIServer struct {
	Addr       string
	CertFile   string
	KeyFile    string
	CAFile     string
	SkipVerify bool

	UseKubeConfig bool
}

// RegisterFlags registers flags to the provided FlagSet.
func (s *APIServer) RegisterFlags(fs *flag.FlagSet) {
	fs.StringVar(&s.Addr, "apiserver.addr", "", "(NOT RECOMMENDED FOR PRODUCTION) API Server address to connect to. Omit parameter to run in on-cluster mode and utilize the service account token.")
	fs.StringVar(&s.CertFile, "apiserver.cert-file", "", "(NOT RECOMMENDED FOR PRODUCTION) Path to public TLS certificate file.")
	fs.StringVar(&s.KeyFile, "apiserver.key-file", "", "(NOT RECOMMENDED FOR PRODUCTION) Path to private TLS certificate file.")
	fs.StringVar(&s.CAFile, "apiserver.ca-file", "", "(NOT RECOMMENDED FOR PRODUCTION) Path to TLS CA file.")
	fs.BoolVar(&s.SkipVerify, "apiserver.skip-verify", false, "(NOT RECOMMENDED FOR PRODUCTION) Skip verification of API server's CA certificate.")

	fs.BoolVar(&s.UseKubeConfig, "apiserver.use-kubeconfig", false, "(NOT RECOMMENDED FOR PRODUCTION) Retrieve API server settings from kubeconfig. If set, takes precedence over all over apiserver settings.")
}
