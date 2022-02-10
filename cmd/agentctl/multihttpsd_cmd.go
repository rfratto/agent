package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"

	"github.com/go-kit/log"
	"github.com/grafana/agent/pkg/discoveryext"
	"github.com/grafana/agent/pkg/discoveryext/multihttp"
	"github.com/prometheus/prometheus/discovery"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func multiHTTPSDCmd() *cobra.Command {
	var (
		listenAddr = "127.0.0.1:9999"
		configFile string
	)

	cmd := &cobra.Command{
		Use:   "multihttpsd",
		Short: "Runs an http_sd server that discovers and aggregates the results from other http_sd servers",
		RunE: func(*cobra.Command, []string) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var cfg multihttp.Config

			f, err := os.Open(configFile)
			if err != nil {
				return fmt.Errorf("failed to open config file %q: %w", configFile, err)
			}
			defer f.Close()

			dec := yaml.NewDecoder(f)
			dec.SetStrict(true)
			if err := dec.Decode(&cfg); err != nil {
				return fmt.Errorf("failed to read config file %q: %w", configFile, err)
			}

			disc, err := cfg.NewDiscoverer(discovery.DiscovererOptions{
				Logger: log.NewNopLogger(),
			})
			if err != nil {
				return fmt.Errorf("failed to create multihttp discovery: %w", err)
			}

			w := discoveryext.NewWatcher(disc)
			defer func() { _ = w.Close() }()
			srv := http.Server{Handler: w}

			go func() {
				sig := make(chan os.Signal, 1)
				signal.Notify(sig, os.Interrupt)

				select {
				case <-sig:
				case <-ctx.Done():
				}

				_ = srv.Close()
				signal.Stop(sig)
			}()

			lis, err := net.Listen("tcp", listenAddr)
			if err != nil {
				return fmt.Errorf("failed to listen on %q: %w", listenAddr, err)
			}
			return srv.Serve(lis)
		},
	}

	cmd.Flags().StringVar(&listenAddr, "listen-addr", listenAddr, "TCP host:port to expose http_sd on")
	cmd.Flags().StringVar(&configFile, "config-file", "", "Config file to load.")

	must(cmd.MarkFlagRequired("config-file"))

	return cmd
}
