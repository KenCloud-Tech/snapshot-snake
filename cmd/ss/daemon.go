package main

import (
	"context"
	"fmt"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"github.com/snapshot_snake/api"
	"github.com/snapshot_snake/common"
	"github.com/snapshot_snake/dep"
	"github.com/snapshot_snake/lib/ffx"
	"github.com/snapshot_snake/lib/monitor"
	"github.com/snapshot_snake/snapshot"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"
	"net/http"
	"time"
)

var daemonCmd = &cli.Command{
	Name: "daemon",
	Subcommands: []*cli.Command{
		daemonStartCmd,
		daemonStopCmd,
	},
}

var daemonStartCmd = &cli.Command{
	Name: "run",
	Action: func(cctx *cli.Context) error {
		ctx := context.Background()
		shutdownCh := make(chan struct{})
		var components struct {
			fx.In
			NodeAPI  api.SnapNodeAPI
			Cfg      snapshot.Config
			Mux      *http.ServeMux
			Notifier common.HeadNotifier
			Ds       common.DagStore
			Shutter  *snapshot.Shutter
		}
		// di
		stopper, err := ffx.New(ctx,
			dep.InjectFullNode(cctx),
			dep.InjectRepoPath(cctx),

			dep.Core(ctx, fxlog, &components),

			ffx.Override(new(dtypes.ShutdownChan), shutdownCh),
		)
		if err != nil {
			return err
		}

		// http
		httpStopper, errCh := serveHTTP(components.Cfg.HTTP.Listen, components.Mux)
		select {
		case err = <-errCh:
		case <-time.After(time.Duration(components.Cfg.HTTP.StableWait)):
		}
		// monitor
		doneCh := monitor.MonitorShutdown(
			shutdownCh,
			monitor.ShutdownHandler{Component: "http server", StopFunc: httpStopper},
			monitor.ShutdownHandler{Component: "application", StopFunc: monitor.StopFunc(stopper)},
		)
		// monitor tsKey channel
		var tsCh = make(chan *types.TipSet, 0)
		ch, err := components.Notifier.Sub(ctx)
		if err != nil {
			return fmt.Errorf("sub head change: %w", err)
		}

		go components.Notifier.GetTipSet(ctx, ch, tsCh)
		go components.Shutter.Run(ctx, doneCh, tsCh)

		// RPC
		addr := components.Cfg.HTTP.RPCListen
		if addr == "" {
			addr = snapshot.DefaultRPCListenAddr
		}
		endpoint, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			return fmt.Errorf("parse addr: %s, err: %v", addr, err)
		}

		return ServeRPC(&components.NodeAPI, stopper, endpoint, doneCh, 0)
	},
}

func serveHTTP(addr string, mux *http.ServeMux) (func(context.Context) error, <-chan error) {
	errCh := make(chan error, 1)
	if addr == "" {
		close(errCh)
		log.Warn("no listen address provided")
		return func(context.Context) error {
			return nil
		}, errCh
	}

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// start http server
	go func() {
		defer close(errCh)

		log.Infof("http server will start on %s", addr)
		err := srv.ListenAndServe()
		if err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				errCh <- err
			}
		}

		return
	}()

	return srv.Shutdown, errCh
}

var daemonStopCmd = &cli.Command{}
