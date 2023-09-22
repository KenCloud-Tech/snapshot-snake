package bsex

import (
	"context"
	"github.com/filecoin-project/lotus/blockstore"
	lru "github.com/hashicorp/golang-lru"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"go4.org/syncutil/singleflight"
)

var _ blockstore.Blockstore = (*CachedBlockstore)(nil)

func NewCachedBlockstore(cacheSize int, bs blockstore.Blockstore) (*CachedBlockstore, error) {
	cache, err := lru.New2Q(cacheSize)
	if err != nil {
		return nil, err
	}

	getSg := new(singleflight.Group)
	hasSg := new(singleflight.Group)
	res := &CachedBlockstore{
		cache:      cache,
		Blockstore: bs,

		getSg: getSg,
		hasSg: hasSg,
	}

	return res, nil
}

type CachedBlockstore struct {
	cache *lru.TwoQueueCache
	blockstore.Blockstore

	getSg *singleflight.Group
	hasSg *singleflight.Group
}

func (cbs *CachedBlockstore) Get(ctx context.Context, c cid.Cid) (blocks.Block, error) {
	if cached, has := cbs.cache.Get(c); has {
		if b, ok := cached.(blocks.Block); ok {
			return b, nil
		}
	}

	b, err := cbs.getSg.Do(c.String(), func() (interface{}, error) {
		b, err := cbs.Blockstore.Get(ctx, c)
		if err != nil {
			return nil, err
		}

		cbs.cache.Add(c, b)
		return b, nil
	})

	if err != nil {
		return nil, err
	}

	return b.(blocks.Block), nil
}

func (cbs *CachedBlockstore) View(ctx context.Context, c cid.Cid, callback func([]byte) error) error {

	if cached, has := cbs.cache.Get(c); has {
		if b, ok := cached.(blocks.Block); ok {
			return callback(b.RawData())
		}
	}
	b, err := cbs.getSg.Do(c.String(), func() (interface{}, error) {
		b, err := cbs.Blockstore.Get(ctx, c)
		if err != nil {
			return nil, err
		}

		cbs.cache.Add(c, b)
		return b, nil
	})

	if err != nil {
		return err
	}

	return callback(b.(blocks.Block).RawData())
}

func (cbs *CachedBlockstore) Has(ctx context.Context, c cid.Cid) (bool, error) {

	if has := cbs.cache.Contains(c); has {
		return true, nil
	}
	b, err := cbs.hasSg.Do(c.String(), func() (interface{}, error) {
		b, err := cbs.Blockstore.Has(ctx, c)
		if err != nil {
			return false, err
		}
		return b, nil
	})

	if err != nil {
		return false, err
	}

	return b.(bool), nil
}
