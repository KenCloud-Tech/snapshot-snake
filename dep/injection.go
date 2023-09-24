package dep

import (
	"context"
	"github.com/filecoin-project/lotus/node/modules/helpers"
	"github.com/ipfs/go-metrics-interface"
	"github.com/snapshot_snake/common"
	"github.com/snapshot_snake/lib/cliex"
	"github.com/snapshot_snake/lib/ffx"
	"github.com/snapshot_snake/snapshot"
	"github.com/snapshot_snake/snapshot/saaf"
	"github.com/snapshot_snake/snapshot/store"
	"go.uber.org/fx"
	"net/http"
)

const (
	invokeNone ffx.Invoke = iota

	invokePopulate
)

func Core(ctx context.Context, logger fx.Printer, target ...interface{}) ffx.Option {
	return ffx.Options(
		ffx.Override(new(GlobalContext), ctx),
		ffx.Override(new(*http.ServeMux), http.NewServeMux()),
		ffx.Override(new(helpers.MetricsCtx), metrics.CtxScope(ctx, "snapshot")),

		ffx.If(logger != nil, ffx.Logger(logger)),
		ffx.If(len(target) > 0, ffx.Populate(invokePopulate, target...)),

		// config
		ffx.Override(new(snapshot.Config), LoadConfig),

		// notifier & dag
		ffx.Override(new(common.HeadNotifier), cliex.NewHeadSub),
		ffx.Override(new(*saaf.SnapSource), saaf.NewSnapSource),
		ffx.Override(new(saaf.NodeStore), saaf.NewMapNodeStore),
		ffx.Override(new(*saaf.DAG), saaf.NewDAG),

		ffx.Override(new(common.DagStore), store.NewCacheBlockStore),

		ffx.Override(new(*snapshot.Shutter), NewSnapshot),
	)
}
