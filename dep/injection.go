package dep

import (
	"context"
	"github.com/FIL_FIL_Snapshot/common"
	"github.com/FIL_FIL_Snapshot/lib/cliex"
	"github.com/FIL_FIL_Snapshot/lib/ffx"
	"github.com/FIL_FIL_Snapshot/snapshot"
	"github.com/FIL_FIL_Snapshot/snapshot/saaf"
	"github.com/filecoin-project/lotus/chain/consensus/filcns"
	"github.com/filecoin-project/lotus/chain/stmgr"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/node/modules"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/node/modules/helpers"
	"github.com/ipfs/go-metrics-interface"
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
		ffx.Override(new(helpers.MetricsCtx), metrics.CtxScope(ctx, "filfil")),

		ffx.If(logger != nil, ffx.Logger(logger)),
		ffx.If(len(target) > 0, ffx.Populate(invokePopulate, target...)),

		// config
		ffx.Override(new(snapshot.Config), LoadConfig),

		// notifier & dag
		ffx.Override(new(common.HeadNotifier), cliex.NewHeadSub),
		ffx.Override(new(*saaf.FilFilSource), saaf.NewFilFilSource),
		ffx.Override(new(*saaf.NodeStore), saaf.NewMapNodeStore),
		ffx.Override(new(*saaf.DAG), saaf.NewDAG),

		ffx.Override(new(dtypes.HotBlockstore), ChainIOBlockstore),
		ffx.Override(new(dtypes.ChainBlockstore), ffx.From(new(dtypes.HotBlockstore))),
		ffx.Override(new(dtypes.StateBlockstore), ffx.From(new(dtypes.HotBlockstore))),
		ffx.Override(new(dtypes.BaseBlockstore), ffx.From(new(dtypes.HotBlockstore))),
		ffx.Override(new(dtypes.ExposedBlockstore), ffx.From(new(dtypes.HotBlockstore))),

		ffx.Override(new(SegmentMetaDS), OpenSegmentDS),
		ffx.Override(new(store.WeightFunc), filcns.Weight),
		ffx.Override(new(stmgr.UpgradeSchedule), filcns.DefaultUpgradeSchedule),
		ffx.Override(new(*store.ChainStore), modules.ChainStore),

		//ffx.Override(new(*store.ChainStore), ffx.From(new(*store.ChainStore))),
		ffx.Override(new(*snapshot.Shutter), NewSnapshot),
		// override api
	)
}
