package common

import (
	"context"
	"github.com/filecoin-project/lotus/chain/types"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"io"
)

type HeadNotifier interface {
	Sub(ctx context.Context) (<-chan types.TipSetKey, error)
	GetTipSet(context.Context, <-chan types.TipSetKey, chan *types.TipSet)
}

type DagStore interface {
	Has(context.Context, cid.Cid) (bool, error)
	DeleteBlock(context.Context, cid.Cid) error
	Put(context.Context, cid.Cid, blocks.Block) error
	Get(context.Context, cid.Cid) (blocks.Block, error)
	Export(context.Context, *types.TipSet, io.Writer, int64) error
}
