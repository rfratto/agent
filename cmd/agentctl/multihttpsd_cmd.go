package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/grafana/agent/pkg/discoveryext"
	"github.com/grafana/agent/pkg/discoveryext/multihttp"
	"github.com/grafana/agent/pkg/util"
	"github.com/prometheus/prometheus/discovery"
	"github.com/spf13/cobra"
	"github.com/weaveworks/common/server"
	"gopkg.in/yaml.v2"
)

func multiHTTPSDCmd() *cobra.Command {
	var (
		scfg    server.Config
		cfgFile string
	)

	// Inject default values for server.Config.
	util.DefaultConfigFromFlags(&scfg)

	// We don't use gRPC here. gRPC can't be disabled, so we'll set it to listen to a random address.
	scfg.GRPCListenAddress = "127.0.0.1"
	scfg.GRPCListenPort = 0

	cmd := &cobra.Command{
		Use:   "multihttpsd",
		Short: "Runs an http_sd server that discovers and aggregates the results from other http_sd servers",
		RunE: func(*cobra.Command, []string) error {
			logger := util.NewLogger(&scfg)
			scfg.Log = util.GoKitLogger(logger)

			cfg, err := loadMultihttpConfig(cfgFile)
			if err != nil {
				return err
			}

			disc, err := cfg.NewDiscoverer(discovery.DiscovererOptions{Logger: logger})
			if err != nil {
				return fmt.Errorf("failed to create multihttp discovery: %w", err)
			}

			w := discoveryext.NewWatcher(disc)
			defer func() { _ = w.Close() }()

			srv, err := server.New(scfg)
			if err != nil {
				return err
			}

			srv.HTTP.Handle("/targets", w)
			return srv.Run()
		},
	}

	// Register a subset of weaveworks/common/server flags
	cmd.Flags().StringVar(&scfg.HTTPListenAddress, "server.http-listen-address", "127.0.0.1", "HTTP server listen address.")
	cmd.Flags().IntVar(&scfg.HTTPListenPort, "server.http-listen-port", 9999, "HTTP server listen address.")

	cmd.Flags().AddGoFlag(&flag.Flag{
		Name:     "log.format",
		Usage:    "Output log messages in the given format. Valid formats: [logfmt, json]",
		Value:    &scfg.LogFormat,
		DefValue: scfg.LogFormat.String(),
	})

	cmd.Flags().AddGoFlag(&flag.Flag{
		Name:     "log.level",
		Usage:    "Only log messages with the given severity or above. Valid levels: [debug, info, warn, error]",
		Value:    &scfg.LogLevel,
		DefValue: scfg.LogLevel.String(),
	})

	cmd.Flags().StringVar(&cfgFile, "config.file", "", "Config file to load.")
	must(cmd.MarkFlagRequired("config.file"))

	return cmd
}

func loadMultihttpConfig(path string) (*multihttp.Config, error) {
	var cfg multihttp.Config

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file %q: %w", path, err)
	}
	defer f.Close()

	dec := yaml.NewDecoder(f)
	dec.SetStrict(true)
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to read config file %q: %w", path, err)
	}

	return &cfg, nil
}
