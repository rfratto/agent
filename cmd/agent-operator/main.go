// Command agent-operator implements a Kubernetes Operator to make deploying
// the Grafana Cloud Agent easier.
package main

import (
	// Adds version information
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/go-kit/kit/log/level"
	"github.com/gorilla/mux"
	_ "github.com/grafana/agent/pkg/build"
	"github.com/grafana/agent/pkg/operator"
	"github.com/grafana/agent/pkg/operator/controller"
	"github.com/grafana/agent/pkg/util"
	"github.com/grafana/agent/pkg/util/server"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"
	"github.com/weaveworks/common/signals"
	"google.golang.org/grpc"
)

func main() {
	var (
		printVersion bool
		cfg          operator.Config

		err error
	)

	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.BoolVar(&printVersion, "version", false, "Print this build's version information")
	cfg.RegisterFlags(fs)

	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Fatalln(err)
	}

	if printVersion {
		fmt.Println(version.Print("agent-operator"))
		os.Exit(0)
	}

	logger := util.NewLogger(&cfg.Server)
	cfg.Server.Log = util.GoKitLogger(logger)

	var (
		rg run.Group

		ctx, cancel   = context.WithCancel(context.Background())
		signalHandler = signals.NewHandler(cfg.Server.Log)
	)
	defer cancel()

	srv := server.New(prometheus.DefaultRegisterer, logger)
	if err := srv.ApplyConfig(cfg.Server, noOpWire); err != nil {
		level.Error(logger).Log("msg", "failed to apply server config", "err", err)
		os.Exit(1)
	}

	ctrl, err := controller.New(ctx, cfg, logger, prometheus.DefaultRegisterer)
	if err != nil {
		level.Error(logger).Log("msg", "failed to create controller", "err", err)
		os.Exit(1)
	}

	rg.Add(func() error {
		<-ctx.Done()
		return nil
	}, func(e error) {
		cancel()
	})

	rg.Add(func() error {
		signalHandler.Loop()
		return nil
	}, func(e error) {
		signalHandler.Stop()
	})

	rg.Add(func() error {
		return srv.Run()
	}, func(e error) {
		srv.Close()
	})

	rg.Add(func() error {
		return ctrl.Run(ctx)
	}, func(e error) {
		cancel()
	})

	if err := rg.Run(); err != nil {
		level.Error(logger).Log("msg", "shutting down operator due to error", "err", err)
	}
	level.Info(logger).Log("msg", "bye!")
}

func noOpWire(mux *mux.Router, grpc *grpc.Server) {}
