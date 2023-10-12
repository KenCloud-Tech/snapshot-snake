package saaf

import (
	"fmt"
	"github.com/filecoin-project/lotus/chain/types"
	block "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"sync"
)

var log = logging.Logger("saaf")

const MAX_HEIGHT = 3000

type SnapNode struct {
	fCid cid.Cid
	fBlk types.BlockHeader
}

func (f *SnapNode) Pointer() cid.Cid {
	return f.fCid
}

func (f *SnapNode) GetBlkHeader() types.BlockHeader {
	return f.fBlk
}

func (f *SnapNode) GetBlock() (block.Block, error) {
	blk, err := f.fBlk.ToStorageBlock()
	if err != nil {
		return nil, err
	}
	return blk, nil
}

func (f *SnapNode) GetHeight() int64 {
	return int64(f.fBlk.Height)
}

func (f *SnapNode) Parents() []cid.Cid {
	parents := f.fBlk.Parents

	return parents
}

func NewSnapNode(id cid.Cid, header types.BlockHeader) *SnapNode {
	return &SnapNode{
		fCid: id,
		fBlk: header,
	}
}

type SnapSource struct {
	hpMapping map[Height][]cid.Cid

	pnMapping map[cid.Cid]Node
}

func (s *SnapSource) HpRange() int {
	return len(s.hpMapping)
}

func (f *SnapSource) Latest() []cid.Cid {
	height := findLatestHeight(f.hpMapping)
	log.Warnf("height %d", height)
	return f.FindPointersByHeight(height)
}

func (f *SnapSource) Remove(pointer cid.Cid) {
	delete(f.pnMapping, pointer)
}

func (f *SnapSource) FindPointersByHeight(height Height) []cid.Cid {
	return f.hpMapping[height]
}

func (f *SnapSource) GetBlockByCid(id cid.Cid) block.Block {
	node := f.pnMapping[id]
	filNode := node.(*SnapNode)

	blk, err := filNode.GetBlock()

	if err != nil {
		log.Errorf("node get block err: %s", err)
		return nil
	}
	return blk
}

// find the oldest height
func findOldestHeight(hpMapping map[Height][]cid.Cid) Height {
	oldestHeight := Height(0)
	for height := range hpMapping {
		if oldestHeight == 0 || height < oldestHeight {
			oldestHeight = height
		}
	}
	return oldestHeight
}

// find the latest height
func findLatestHeight(hpMapping map[Height][]cid.Cid) Height {
	latestHeight := Height(0)
	for height := range hpMapping {
		if latestHeight == 0 || height > latestHeight {
			latestHeight = height
		}
	}
	return latestHeight
}

func (ffs *SnapSource) AddSource(ts types.TipSet) []cid.Cid {
	height := ts.Height()
	cids := ts.Cids()
	blks := ts.Blocks()
	var rcids []cid.Cid
	var mu sync.Mutex

	// add hpMapping
	addNewHeight := func(height Height, cids []cid.Cid) {
		mu.Lock()
		defer mu.Unlock()

		// add new ts to hpMapping
		ffs.hpMapping[height] = cids

		// check height
		for len(ffs.hpMapping) > MAX_HEIGHT {
			oldestHeight := findOldestHeight(ffs.hpMapping)
			rcids = ffs.hpMapping[oldestHeight]
			// delete ts in hpMapping
			delete(ffs.hpMapping, oldestHeight)
		}
	}

	addNewHeight(Height(height), cids)

	// add pnMapping
	for i := 0; i < len(cids); i++ {
		id := cids[i]
		header := blks[i]
		snapNode := NewSnapNode(id, *header)

		ffs.pnMapping[id] = snapNode
	}

	// delete ts in pnMapping
	for _, rcid := range rcids {
		delete(ffs.pnMapping, rcid)
	}

	return rcids
}

func (ffs *SnapSource) Resolve(p cid.Cid) (Node, error) {
	node, ok := ffs.pnMapping[p]
	if !ok {
		return nil, fmt.Errorf("failed to resolve pointer to node %s", p)
	}
	return node, nil
}

func NewSnapSource() *SnapSource {
	return &SnapSource{
		hpMapping: map[Height][]cid.Cid{},
		pnMapping: map[cid.Cid]Node{},
	}
}

var _ Node = (*SnapNode)(nil)
var _ Source = (*SnapSource)(nil)
