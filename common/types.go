package common

import (
	"context"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"io"
)

type HeadNotifier interface {
	Sub(ctx context.Context) (<-chan types.TipSetKey, error)
}

type ChainStore interface {
	LoadTipSet(context.Context, types.TipSetKey) (*types.TipSet, error)
	GetTipSetFromKey(context.Context, types.TipSetKey) (*types.TipSet, error)
	Export(context.Context, *types.TipSet, abi.ChainEpoch, bool, io.Writer) error
}
