package api

import (
	"context"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
)

type SnapAPI interface {
	GetDagNode() ([]cid.Cid, error)
	ChainGetTipSet(context.Context, types.TipSetKey) (*types.TipSet, error)
	SnapDagExport(context.Context, *types.TipSet) (<-chan []byte, error)
}
