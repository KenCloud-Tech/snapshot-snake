package main

import (
	"context"
	"github.com/FIL_FIL_Snapshot/api"
	"github.com/FIL_FIL_Snapshot/dep"
	"github.com/FIL_FIL_Snapshot/lib/ffx"
	"github.com/FIL_FIL_Snapshot/snapshot"
	"github.com/filecoin-project/go-jsonrpc"
	cliutil "github.com/filecoin-project/lotus/cli/util"
	"github.com/filecoin-project/lotus/metrics"
	logging "github.com/ipfs/go-log/v2"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/tag"
	"golang.org/x/xerrors"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var (
	log   = logging.Logger("filfil")
	fxlog = &fxLogger{
		ZapEventLogger: log,
	}
)

type fxLogger struct {
	*logging.ZapEventLogger
}

// Printf impls fx.Printer.Printf
func (l *fxLogger) Printf(msg string, args ...interface{}) {
	l.ZapEventLogger.Debugf(msg, args...)
}

// GetAPIV0
func GetAPIV0(ctx *cli.Context) (api.FilFilAPI, jsonrpc.ClientCloser, error) {
	var res api.FilFilAPIStruct
	rpath, err := dep.GetRepoPath(ctx)
	if err != nil {
		return nil, nil, err
	}
	cfg, err := dep.LoadConfig(rpath)
	if err != nil {
		return nil, nil, err
	}
	muladdr := cfg.HTTP.RPCListen
	if muladdr == "" {
		muladdr = snapshot.DefaultRPCListenAddr
	}
	addr, err := cliutil.APIInfo{Addr: muladdr}.DialArgs("v0")
	if err != nil {
		return nil, nil, err
	}
	closer, err := jsonrpc.NewMergeClient(ctx.Context, addr, "filfil",
		[]interface{}{
			&res.Internal,
		},
		nil,
	)
	return &res, closer, err
}

func ServeRPC(a api.FilFilAPI, stop ffx.StopFunc, addr multiaddr.Multiaddr, shutdownCh <-chan struct{}, maxRequestSize int64) error {
	// Create a JSON-RPC server and set the maximum request size option if needed.
	serverOptions := make([]jsonrpc.ServerOption, 0)
	if maxRequestSize != 0 {
		serverOptions = append(serverOptions, jsonrpc.WithMaxRequestSize(maxRequestSize))
	}
	rpcServer := jsonrpc.NewServer(serverOptions...)
	rpcServer.Register("filfil", a)

	// Register the JSON-RPC server handler at the path "/rpc/v0" of the HTTP server.
	http.Handle("/rpc/v0", rpcServer)

	// Create a listener with the specified address.
	lst, err := manet.Listen(addr)
	if err != nil {
		return xerrors.Errorf("could not listen: %w", err)
	}

	// Create an HTTP server instance.
	srv := &http.Server{
		Handler: http.DefaultServeMux,
		BaseContext: func(listener net.Listener) context.Context {
			ctx, _ := tag.New(context.Background(), tag.Upsert(metrics.APIInterface, "lotus-daemon"))
			return ctx
		},
	}

	// Create channels for capturing signals and notifying the completion of shutdown.
	sigCh := make(chan os.Signal, 2)
	shutdownDone := make(chan struct{})
	// Start a goroutine to handle shutdown operations sync
	go func() {
		select {
		case sig := <-sigCh:
			log.Warnw("received shutdown", "signal", sig)
		case <-shutdownDone:
			log.Warn("received shutdown")
		}

		log.Warn("Shutting down...")
		if err := srv.Shutdown(context.TODO()); err != nil {
			log.Errorf("shutting down RPC server failed: %s", err)
		}
		if err := stop(context.TODO()); err != nil {
			log.Errorf("graceful shutting down failed: %s", err)
		}
		log.Warn("Graceful shutdown successful")
		_ = log.Sync()
		close(shutdownDone)
	}()
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	// Start listening with the HTTP server.
	err = srv.Serve(manet.NetListener(lst))
	if err == http.ErrServerClosed {
		<-shutdownDone
		return nil
	}
	return err
}

func CreateExportFile(app *cli.App, path string) (io.WriteCloser, error) {
	if wc, ok := app.Metadata["export-file"]; ok {
		return wc.(io.WriteCloser), nil
	}

	fi, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return fi, nil
}
