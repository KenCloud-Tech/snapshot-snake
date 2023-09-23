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

const MAX_HEIGHT = 1000

type FilFilNode struct {
	fCid cid.Cid
	fBlk types.BlockHeader
}

func (f *FilFilNode) Pointer() cid.Cid {
	return f.fCid
}

func (f *FilFilNode) GetBlkHeader() types.BlockHeader {
	return f.fBlk
}

func (f *FilFilNode) GetBlock() (block.Block, error) {
	blk, err := f.fBlk.ToStorageBlock()
	if err != nil {
		return nil, err
	}
	return blk, nil
}

func (f *FilFilNode) GetHeight() int64 {
	return int64(f.fBlk.Height)
}

func (f *FilFilNode) Parents() []cid.Cid {
	parents := f.fBlk.Parents

	return parents
}

func NewFilFilNode(id cid.Cid, header types.BlockHeader) *FilFilNode {
	return &FilFilNode{
		fCid: id,
		fBlk: header,
	}
}

type FilFilSource struct {
	hpMapping map[Height][]cid.Cid

	pnMapping map[cid.Cid]Node
}

func (f *FilFilSource) Latest() []cid.Cid {
	height := findLatestHeight(f.hpMapping)
	return f.FindPointersByHeight(height)
}

func (f *FilFilSource) Remove(pointer cid.Cid) {
	delete(f.pnMapping, pointer)
}

func (f *FilFilSource) FindPointersByHeight(height Height) []cid.Cid {
	return f.hpMapping[height]
}

func (f *FilFilSource) GetBlockByCid(id cid.Cid) block.Block {
	node := f.pnMapping[id]
	filNode := node.(*FilFilNode)

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

func (ffs *FilFilSource) AddSource(ts types.TipSet) []cid.Cid {
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
			delete(ffs.hpMapping, oldestHeight)
		}
	}

	addNewHeight(Height(height), cids)

	// add pnMapping
	for i := 0; i < len(cids); i++ {
		id := cids[i]
		header := blks[i]
		filFilNode := NewFilFilNode(id, *header)

		ffs.pnMapping[id] = filFilNode
	}

	for _, rcid := range rcids {
		delete(ffs.pnMapping, rcid)
	}

	return rcids
}

func (ffs *FilFilSource) Resolve(p cid.Cid) (Node, error) {
	node, ok := ffs.pnMapping[p]
	if !ok {
		return nil, fmt.Errorf("failed to resolve pointer to node %s", p)
	}
	return node, nil
}

func NewFilFilSource() *FilFilSource {
	return &FilFilSource{
		hpMapping: map[Height][]cid.Cid{},
		pnMapping: map[cid.Cid]Node{},
	}
}

var _ Node = (*FilFilNode)(nil)
var _ Source = (*FilFilSource)(nil)
