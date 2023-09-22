package saaf

import (
	"fmt"
	"github.com/ipfs/go-cid"
)

/*
   saaf is a simple library implementing indirect reference counted GC on immutible DAGs
   `DAG` `Node`s implement a simple interface using native strings as pointers
   `DAG.Link` pins a node to the DAG, `DAG.Unlink` unpins it.
   Only linked nodes can be unlinked to disallow dangling references
   `Link` takes in nodes reachable from a root as resolved through a node `Source` and preserves them in a `NodeStore`
   `Unlink` removes nodes from the NodeStore that are no longer linked by any root

   `Link` and `Unlink` operations are intended to scale in node loads and ref updates as O(n + L)
   - n is the total number of new nodes being added or removed
   - L is the number of links from the connected component being linked or unlinked pointing into the existing DAG
*/

type Pointer cid.Cid
type Height int64

type Node interface {
	Pointer() Pointer
	Parents() []Pointer
	//GetBlockHeader() *types.BlockHeader
	//FilNodeToBuildBlock() (block.Block, error)
}

type Source interface {
	Resolve(Pointer) (Node, error)
}

type NodeStore interface {
	Put(Pointer, Node) error
	Get(Pointer) (Node, error)
	Delete(Pointer) error
	All() <-chan Node
}

type DAG struct {
	// Invariant: alls nodes are tracked in both refs and nodes or neither
	// refs tracks linked references to node at given pointer
	refs map[Pointer]uint64
	// nodes stores all nodes in the DAG
	nodes NodeStore
}

func NewDAG(s *NodeStore) *DAG {
	return &DAG{
		refs:  make(map[Pointer]uint64),
		nodes: *s,
	}
}

func (d *DAG) Link(p Pointer, src Source) error {
	toLink := []Pointer{p}
	for len(toLink) > 0 {
		p := toLink[0]
		toLink = toLink[1:]
		_, linked := d.refs[p]
		if linked {
			d.refs[p] += 1
			continue
		}
		// if not linked then link node and traverse children
		d.refs[p] = 1
		n, err := src.Resolve(p)
		if err != nil {
			return err
		}
		if err := d.nodes.Put(p, n); err != nil {
			fmt.Errorf("failed to put to node store: %w", err)
		}
		toLink = append(toLink, n.Parents()...)
	}
	return nil
}

func (d *DAG) Unlink(p Pointer) error {
	toUnlink := []Pointer{p}
	for len(toUnlink) > 0 {
		p := toUnlink[0]
		toUnlink = toUnlink[1:]
		r, linked := d.refs[p]
		if !linked {
			return fmt.Errorf("failed to delete pointer %s, node not linked in DAG \n", p)
		}
		if r > 1 {
			d.refs[p] -= 1
			continue
		}
		// if this is the last reference delete the sub DAG
		delete(d.refs, p)
		n, err := d.nodes.Get(p)
		if err != nil {
			return fmt.Errorf("internal DAG error, pointer %s reference counted but failed to get node: %w", p, err)
		}
		toUnlink = append(toUnlink, n.Parents()...)
		if err := d.nodes.Delete(p); err != nil {
			return fmt.Errorf("internal DAG error, failed to delete node %s, %w", p, err)
		}
	}
	return nil
}

func (d *DAG) Store() NodeStore {
	return d.nodes
}

// In memory node store backed by a simple map
type MapNodeStore struct {
	nodes map[Pointer]Node
}

func NewMapNodeStore() MapNodeStore {
	return MapNodeStore{
		nodes: make(map[Pointer]Node),
	}
}

func (s MapNodeStore) Put(p Pointer, n Node) error {
	s.nodes[p] = n
	return nil
}

func (s MapNodeStore) Get(p Pointer) (Node, error) {
	n, ok := s.nodes[p]
	if !ok {
		return nil, fmt.Errorf("could not resolve pointer %s", p)
	}
	return n, nil
}

func (s MapNodeStore) All() <-chan Node {
	ch := make(chan Node, 0)
	go func() {
		for p := range s.nodes {
			ch <- s.nodes[p]
		}
		close(ch)
	}()
	return ch
}

func (s MapNodeStore) Delete(p Pointer) error {
	if _, ok := s.nodes[p]; !ok {
		return fmt.Errorf("%s not stored", p)
	}
	delete(s.nodes, p)
	return nil
}

var _ NodeStore = MapNodeStore{}
