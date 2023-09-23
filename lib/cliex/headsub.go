package cliex

import (
	"context"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	logging "github.com/ipfs/go-log/v2"
	"time"
)

var log = logging.Logger("headsub")

const (
	minReListenInterval = time.Second
	maxReListenInterval = 10 * time.Second

	nonChanModeInterval = 10 * time.Second
)

func NewHeadSub(full v0api.FullNode) (*HeadSub, error) {
	return &HeadSub{
		full:     full,
		interval: minReListenInterval,
	}, nil
}

type HeadSub struct {
	full     v0api.FullNode
	interval time.Duration
}

func (h *HeadSub) GetTipSet(ctx context.Context, tsk <-chan types.TipSetKey, tsCh chan *types.TipSet) {
	for {
		select {
		case <-ctx.Done():
			log.Infof("stop load tipset")
			return
		case tipSetKey := <-tsk:
			rawTipSet, err := h.full.ChainGetTipSet(ctx, tipSetKey)
			if err != nil {
				log.Errorf("failed to get tipset: %s", err)
				return
			}

			tsCh <- rawTipSet
		}
	}
}

func (h *HeadSub) Sub(ctx context.Context) (<-chan types.TipSetKey, error) {
	ch := make(chan types.TipSetKey, 1)
	go h.watch(ctx, ch)
	return ch, nil
}

func (h *HeadSub) watch(ctx context.Context, tx chan types.TipSetKey) {
	log.Info("head change loop start")
	defer log.Info("head change loop stop")

	ch, err := h.full.ChainNotify(ctx)
	if err != nil {
		log.Fatalf("failed to get chain notify channel: %s", err)
		return
	}
	cancel := context.CancelFunc(func() {})

	for {
		select {
		case <-ctx.Done():
			cancel()
			return
		case changes, ok := <-ch:
			if !ok {
				log.Error("failed to get chain head update")
				return
			}
			cancel()

			applyCtx, applyCancel := context.WithCancel(ctx)
			cancel = applyCancel
			h.applyChanges(applyCtx, tx, changes)
		}
	}
}

func (h *HeadSub) applyChanges(ctx context.Context, tx chan types.TipSetKey, changes []*api.HeadChange) {
	idx := -1

	for i := range changes {
		switch changes[i].Type {
		case store.HCCurrent, store.HCApply:
			idx = i
		}
	}

	if idx == -1 {
		return
	}

	tsk := changes[idx].Val.Key()
	go delaySend(ctx, tx, tsk)
}

func delaySend(ctx context.Context, ch chan types.TipSetKey, tsk types.TipSetKey) {
	slog := log.With("tsk", tsk)

	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		slog.Debug("aborted")
		return

	case <-timer.C:

	}

	wait := time.NewTimer(time.Second)
	defer wait.Stop()

	select {
	case <-ctx.Done():
		slog.Debug("aborted")

	case ch <- tsk:
		slog.Debug("sent")

	case <-wait.C:
		slog.Debug("out chan is full")
	}
}
