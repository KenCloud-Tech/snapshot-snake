package api

import (
	"context"
	"github.com/FIL_FIL_Snapshot/snapshot/saaf"
	"github.com/filecoin-project/lotus/chain/types"
)

type FilFilAPI interface {
	GetDagNode(context.Context, saaf.Height) ([]saaf.Pointer, error)
	FilFilDagExport(ctx context.Context, height saaf.Height, tsk types.TipSetKey) (<-chan []byte, error)
	ChainGetTipSet(context.Context, types.TipSetKey) (*types.TipSet, error)
}
