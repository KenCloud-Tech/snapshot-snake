package api

import (
	"bufio"
	"context"
	"github.com/FIL_FIL_Snapshot/common"
	"github.com/FIL_FIL_Snapshot/snapshot"
	"github.com/FIL_FIL_Snapshot/snapshot/saaf"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	logging "github.com/ipfs/go-log/v2"
	"go.uber.org/fx"
	"golang.org/x/xerrors"
	"io"
)

var _ FilFilAPI = (*FilFilNodeAPI)(nil)
var log = logging.Logger("rpc")

type FilFilNodeAPI struct {
	fx.In

	Snapshot *snapshot.Shutter
	Sub      common.HeadNotifier
	Cs       common.ChainStore

	Dag *saaf.DAG
	Src *saaf.FilFilSource
}

func (f FilFilNodeAPI) ChainGetTipSet(ctx context.Context, key types.TipSetKey) (*types.TipSet, error) {
	return f.Cs.LoadTipSet(ctx, key)
}

func (f FilFilNodeAPI) FilFilDagExport(ctx context.Context, height saaf.Height, tsk types.TipSetKey) (<-chan []byte, error) {
	// todo tsk => ts
	//f.Cs.Get
	ts, err := f.Cs.GetTipSetFromKey(ctx, tsk)
	if err != nil {
		return nil, xerrors.Errorf("loading tipset %s: %w", tsk, err)
	}

	r, w := io.Pipe()
	out := make(chan []byte)
	go func() {
		bw := bufio.NewWriterSize(w, 1<<20)

		err := f.Cs.Export(ctx, ts, abi.ChainEpoch(height), true, bw)
		bw.Flush()
		w.CloseWithError(err)
	}()

	go func() {
		defer close(out)
		for {
			buf := make([]byte, 1<<20)
			n, err := r.Read(buf)
			if err != nil && err != io.EOF {
				log.Errorf("chain export pipe read failed: %s", err)
				return
			}
			if n > 0 {
				select {
				case out <- buf[:n]:
				case <-ctx.Done():
					log.Warnf("export writer failed: %s", ctx.Err())
					return
				}
			}
			if err == io.EOF {
				// send empty slice to indicate correct eof
				select {
				case out <- []byte{}:
				case <-ctx.Done():
					log.Warnf("export writer failed: %s", ctx.Err())
					return
				}

				return
			}
		}
	}()

	return out, nil
}

func (f FilFilNodeAPI) GetDagNode(ctx context.Context, height saaf.Height) ([]saaf.Pointer, error) {
	return f.Src.FindPointersByHeight(height), nil
}
