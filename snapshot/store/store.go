package store

import (
	"bytes"
	"context"
	"fmt"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/types"
	lru "github.com/hashicorp/golang-lru"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/ipld/go-car"
	carutil "github.com/ipld/go-car/util"
	"github.com/multiformats/go-multicodec"
	"github.com/snapshot_snake/common"
	"github.com/snapshot_snake/snapshot/saaf"
	typegen "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/xerrors"
	"io"
)

var DefaultBlkCacheCacheSize = 8192

var log = logging.Logger("store")

const HEIGHT_COUNT = 10

func NewCacheBlockStore(dag *saaf.DAG) (*CacheBlockStore, error) {
	cache, err := lru.New2Q(DefaultBlkCacheCacheSize)
	if err != nil {
		return nil, err
	}

	res := &CacheBlockStore{
		dag:   dag,
		cache: cache,
	}

	return res, nil
}

type CacheBlockStore struct {
	dag   *saaf.DAG
	cache *lru.TwoQueueCache
}

func (cbs *CacheBlockStore) Get(ctx context.Context, c cid.Cid) (blocks.Block, error) {
	has, err := cbs.Has(ctx, c)
	if !has {
		return nil, err
	}

	value, ok := cbs.cache.Get(c)
	if !ok {
		return nil, fmt.Errorf("get from cache err: %s", err)
	}
	return value.(blocks.Block), nil
}

func (cbs *CacheBlockStore) Has(ctx context.Context, c cid.Cid) (bool, error) {
	if has := cbs.cache.Contains(c); has {
		return true, fmt.Errorf("Has cid in cache")
	}

	return false, fmt.Errorf("Has not cid in cache")
}

func (cbs *CacheBlockStore) DeleteBlock(ctx context.Context, c cid.Cid) error {
	has, err := cbs.Has(ctx, c)
	if !has {
		return err
	}

	cbs.cache.Remove(c)
	return nil
}

func (cbs *CacheBlockStore) Put(ctx context.Context, c cid.Cid, block blocks.Block) error {
	var err error
	// cid - block
	has, _ := cbs.Has(ctx, c)
	if has {
		return fmt.Errorf("has cid %s in cache", c)
	}
	log.Infof("add cid %s to cache", c)
	cbs.cache.Add(c, block)

	var b types.BlockHeader
	if err = b.UnmarshalCBOR(bytes.NewBuffer(block.RawData())); err != nil {
		return err
	} else {
		// block messages - block
		has, _ := cbs.Has(ctx, b.Messages)
		if has {
			log.Infof("has messages %s in cache", b.Messages)

		} else {
			log.Infof("add messages %s to cache", b.Messages)
			cbs.cache.Add(b.Messages, block)
		}

		// block messageReceipts - block
		has, _ = cbs.Has(ctx, b.ParentMessageReceipts)
		if has {
			log.Warnf("has parentmessagereceipts %s in cache", b.ParentMessageReceipts)
		} else {
			log.Infof("add parentmessagereceipts %s to cache", b.ParentMessageReceipts)
			cbs.cache.Add(b.ParentMessageReceipts, block)
		}

		// block parentStateRoot - block
		has, _ = cbs.Has(ctx, b.ParentStateRoot)
		if has {
			log.Warnf("has parentstateroot %s in cache", b.ParentStateRoot)
		} else {
			log.Infof("add parentstateroot %s to cache", b.ParentStateRoot)
			cbs.cache.Add(b.ParentStateRoot, block)
		}
	}

	return err
}

func (cbs *CacheBlockStore) Export(ctx context.Context, ts *types.TipSet, w io.Writer, rs int64) error {
	h := &car.CarHeader{
		Roots:   ts.Cids(),
		Version: 1,
	}

	if err := car.WriteHeader(h, w); err != nil {
		return xerrors.Errorf("failed to write car header: %s", err)
	}

	return cbs.WalkSnapshot(ctx, ts, rs, func(c cid.Cid) error {
		blk, err := cbs.Get(ctx, c)
		if err != nil {
			log.Errorf("cid ====> %s", c)
			return xerrors.Errorf("writing object to car, bs.Get: %w", err)
		}

		if err := carutil.LdWrite(w, c.Bytes(), blk.RawData()); err != nil {
			return xerrors.Errorf("failed to write block to car output: %w", err)
		}

		return nil
	})
}

func (cbs *CacheBlockStore) WalkSnapshot(ctx context.Context, ts *types.TipSet, rs int64, cb func(cid.Cid) error) error {
	seen := cid.NewSet()
	walked := cid.NewSet()

	blocksToWalk := ts.Cids()

	walkDAG := func(blk cid.Cid) error {
		if !seen.Visit(blk) {
			return nil
		}

		if err := cb(blk); err != nil {
			return err
		}

		data, err := cbs.Get(ctx, blk)
		if err != nil {
			return xerrors.Errorf("getting block: %w", err)
		}

		var b types.BlockHeader
		if err := b.UnmarshalCBOR(bytes.NewBuffer(data.RawData())); err != nil {
			return xerrors.Errorf("unmarshaling block header (cid=%s): %w", blk, err)
		}

		var cids []cid.Cid
		nodes := cbs.dag.Store()
		store := nodes.(*saaf.MapNodeStore)
		node, err := store.Get(blk)
		n := node.(*saaf.SnapNode)

		for _, pointer := range n.Parents() {
			blocksToWalk = append(blocksToWalk, pointer)
		}

		if b.Height > ts.Height()-abi.ChainEpoch(rs) {
			if walked.Visit(b.Messages) {
				mcids, err := recurseLinks(ctx, cbs, walked, b.Messages, []cid.Cid{b.Messages})
				if err != nil {
					return xerrors.Errorf("cid %s, bid %s, bid.msg %s, recursing messages failed: %w", blk, b.Cid(), b.Messages, err)
				}
				cids = mcids
			}
		}

		out := cids

		if b.Height == 0 || b.Height > ts.Height()-abi.ChainEpoch(rs) {
			if walked.Visit(b.ParentStateRoot) {
				cids, err := recurseLinks(ctx, cbs, walked, b.ParentStateRoot, []cid.Cid{b.ParentStateRoot})
				if err != nil {
					return xerrors.Errorf("recursing genesis state failed: %w", err)
				}

				out = append(out, cids...)
			}

			if walked.Visit(b.ParentMessageReceipts) {
				out = append(out, b.ParentMessageReceipts)
			}
		}

		for _, c := range out {
			if seen.Visit(c) {
				prefix := c.Prefix()

				// Don't include identity CIDs.
				if multicodec.Code(prefix.MhType) == multicodec.Identity {
					continue
				}

				// We only include raw, cbor, and dagcbor, for now.
				switch multicodec.Code(prefix.Codec) {
				case multicodec.Cbor, multicodec.DagCbor, multicodec.Raw:
				default:
					continue
				}

				if err := cb(c); err != nil {
					return err
				}

			}
		}

		return nil
	}

	log.Infow("export started")
	exportStart := build.Clock.Now()

	for len(blocksToWalk) > 0 {
		next := blocksToWalk[0]
		blocksToWalk = blocksToWalk[1:]
		if err := walkDAG(next); err != nil {
			return xerrors.Errorf("walk chain failed: %w", err)
		}
	}

	log.Infow("export finished", "duration", build.Clock.Now().Sub(exportStart).Seconds())

	return nil
}

func recurseLinks(ctx context.Context, bs common.DagStore, walked *cid.Set, root cid.Cid, in []cid.Cid) ([]cid.Cid, error) {
	if multicodec.Code(root.Prefix().Codec) != multicodec.DagCbor {
		return in, nil
	}

	data, err := bs.Get(ctx, root)
	if err != nil {
		log.Errorf("recurseLinks end %s not in cache", root)
		return in, nil
	}

	in = append(in, root)

	var rerr error
	err = typegen.ScanForLinks(bytes.NewReader(data.RawData()), func(c cid.Cid) {
		if rerr != nil {
			// No error return on ScanForLinks :(
			return
		}

		// traversed this already...
		if !walked.Visit(c) {
			return
		}

		var err error
		in, err = recurseLinks(ctx, bs, walked, c, in)
		if err != nil {
			rerr = err
		}
	})
	if err != nil {
		return nil, xerrors.Errorf("scanning for links failed: %w", err)
	}

	return in, rerr
}
