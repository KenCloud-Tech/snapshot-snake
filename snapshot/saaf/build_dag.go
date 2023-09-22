package saaf

import (
	"context"
	"fmt"
	"github.com/filecoin-project/lotus/chain/types"
	block "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"sync"
)

var log = logging.Logger("saaf")

type FilFilNode struct {
	fCid Pointer
	fBlk types.BlockHeader
}

func (f *FilFilNode) Pointer() Pointer {
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

func CidToPointer(cid cid.Cid) Pointer {
	return Pointer(cid)
}

func CidsToPointers(cids []cid.Cid) []Pointer {
	pts := make([]Pointer, len(cids))

	for i, parent := range cids {
		pts[i] = Pointer(parent)
	}
	return pts
}

func PointersToCids(pointers []Pointer) []cid.Cid {
	cids := make([]cid.Cid, len(pointers))
	for i, pointer := range pointers {
		cids[i] = cid.Cid(pointer)
	}
	return cids
}

func (f *FilFilNode) Parents() []Pointer {
	parents := f.fBlk.Parents

	pts := CidsToPointers(parents)

	return pts
}

func NewFilFilNode(id cid.Cid, header types.BlockHeader) *FilFilNode {
	return &FilFilNode{
		fCid: Pointer(id),
		fBlk: header,
	}
}

type FilFilSource struct {
	hpMapping map[Height][]Pointer
	// todo
	pnMapping map[Pointer]Node
}

func (f *FilFilSource) FindPointersByHeight(height Height) []Pointer {
	return f.hpMapping[height]
}

// 找到最旧的高度
func findOldestHeight(hpMapping map[Height][]Pointer) Height {
	oldestHeight := Height(0)
	for height := range hpMapping {
		if oldestHeight == 0 || height < oldestHeight {
			oldestHeight = height
		}
	}
	return oldestHeight
}

func (ffs *FilFilSource) AddSource(ts types.TipSet, dag *DAG) error {
	height := ts.Height()
	cids := ts.Cids()
	blks := ts.Blocks()
	var mu sync.Mutex

	// add hpMapping
	addNewHeight := func(height Height, cids []cid.Cid) error {
		mu.Lock()
		defer mu.Unlock()

		// add new ts to hpMapping
		ffs.hpMapping[Height(height)] = CidsToPointers(cids)

		// check height
		for len(ffs.hpMapping) > 900 {
			oldestHeight := findOldestHeight(ffs.hpMapping)
			dCids := ffs.hpMapping[oldestHeight]
			// delete hpMapping
			delete(ffs.hpMapping, oldestHeight)
			for _, id := range dCids {
				// delete pnMapping
				delete(ffs.pnMapping, id)
				// unlink dag store
				err := dag.Unlink(id)
				return fmt.Errorf("dag unlink node err: %s", err)
			}

		}
		return nil
	}

	err := addNewHeight(Height(height), cids)
	if err != nil {
		return err
	}

	// add pnMapping
	for i := 0; i < len(cids); i++ {
		id := cids[i]
		header := blks[i]
		filFilNode := NewFilFilNode(id, *header)

		ffs.pnMapping[CidToPointer(id)] = filFilNode
	}

	return nil
}

func (ffs *FilFilSource) Resolve(p Pointer) (Node, error) {
	node, ok := ffs.pnMapping[p]
	if !ok {
		return nil, fmt.Errorf("failed to resolve pointer to node %s", p)
	}
	return node, nil
}

func NewFilFilSource() *FilFilSource {
	return &FilFilSource{
		hpMapping: map[Height][]Pointer{},
		pnMapping: map[Pointer]Node{},
	}
}

var _ Node = (*FilFilNode)(nil)
var _ Source = (*FilFilSource)(nil)

//var FilFilDAG = NewDAG(NewMapNodeStore())

//func FilFilDAGBuilder(ctx context.Context, tsCh chan *types.TipSet) error {
//	var ffSource = NewFilFilSource()
//
//	// todo fix
//	cancel := context.CancelFunc(func() {})
//	for {
//		select {
//		case <-ctx.Done():
//			cancel()
//			return fmt.Errorf("FilFilDAG down...")
//		case ts, ok := <-tsCh:
//			if !ok {
//				log.Warn("ts chan closed")
//				return fmt.Errorf("ts chan closed ...")
//			}
//			ffSource.AddSource(*ts)
//
//			cids := ts.Cids()
//
//			for i := 0; i < len(cids); i++ {
//				id := cids[i]
//
//				err := FilFilDAG.Link(CidToPointer(id), ffSource)
//
//				if err != nil {
//					return fmt.Errorf("FilFilDAG put node err: %s", err)
//				}
//			}
//		}
//	}
//}

func FilFilDAGBuilder(ctx context.Context, ts *types.TipSet, dag *DAG, src *FilFilSource) error {
	err := src.AddSource(*ts, dag)
	if err != nil {
		return fmt.Errorf("add source err: %s", err)
	}

	cids := ts.Cids()

	for i := 0; i < len(cids); i++ {
		id := cids[i]

		err := dag.Link(CidToPointer(id), src)

		if err != nil {
			return fmt.Errorf("FilFilDAG put node err: %s", err)
		}
	}

	return nil
}
