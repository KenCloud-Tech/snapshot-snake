package snapshot

import (
	"context"
	"fmt"
	"github.com/filecoin-project/lotus/chain/types"
	lconfig "github.com/filecoin-project/lotus/node/config"
	logging "github.com/ipfs/go-log/v2"
	"github.com/snapshot_snake/common"
	"github.com/snapshot_snake/snapshot/saaf"
	"time"
)

const DefaultHTTPListenAddr = ":15002"
const DefaultRPCListenAddr = "/ip4/127.0.0.1/tcp/6789"

var log = logging.Logger("snapshot")

func DefaultConfig() Config {
	return Config{
		LotusAPI: DefaultLotusAPIOptions(),
		HTTP:     DefaultHTTPOptions(),
	}
}

type Config struct {
	LotusAPI LotusAPI
	HTTP     HTTPOptions
}

type LotusAPI struct {
	APIAddr  string
	APIToken string
}

func DefaultLotusAPIOptions() LotusAPI {
	return LotusAPI{
		APIAddr:  "/ip4/127.0.0.1/tcp/1234",
		APIToken: "",
	}
}

type HTTPOptions struct {
	RPCListen  string
	Listen     string
	StableWait lconfig.Duration
}

func DefaultHTTPOptions() HTTPOptions {
	return HTTPOptions{
		RPCListen:  DefaultRPCListenAddr,
		Listen:     DefaultHTTPListenAddr,
		StableWait: lconfig.Duration(5 * time.Second),
	}
}

func New(ctx context.Context, sub common.HeadNotifier, cs common.DagStore, dag *saaf.DAG, src *saaf.SnapSource) *Shutter {
	shutter := &Shutter{
		sub: sub,
		cd:  cs,
		dag: dag,
		src: src,
	}
	return shutter
}

type Shutter struct {
	sub common.HeadNotifier
	cd  common.DagStore

	dag *saaf.DAG
	src *saaf.SnapSource
}

func (s *Shutter) Run(ctx context.Context, doneCh <-chan struct{}, tsCh <-chan *types.TipSet) {
	// update dag
	go s.DAGUpdate(ctx, s.dag, s.src)

	for {
		select {
		case <-doneCh:
			log.Info("quite head change loop")
			return
		case ts, ok := <-tsCh:
			if !ok {
				log.Warn("tsk chan closed")
				return
			}

			log.Infow("incoming tipset", "height", ts.Height(), "tipset", ts)

			// build snapshot dag
			if err := s.DAGBuilder(ctx, ts, s.dag, s.src); err != nil {
				log.Warnf("failed to build snapshot dag err: %s", err)
			}
		}
	}
}

func (s *Shutter) DAGBuilder(ctx context.Context, ts *types.TipSet, dag *saaf.DAG, src *saaf.SnapSource) error {
	// add ts to source
	rcids := s.src.AddSource(*ts)

	// remove cache cid
	for _, rcid := range rcids {
		err := s.cd.DeleteBlock(ctx, rcid)
		if err != nil {
			return err
		}
	}

	cids := ts.Cids()

	for _, cid := range cids {
		id := cid

		p, err := dag.Link(id, src)

		if err != nil {
			log.Warnf("link id %s err: %s", p, err)
		}

		err = s.cd.Put(ctx, id, src.GetBlockByCid(id))
		if err != nil {
			return fmt.Errorf("put block to cache err: %s", err)
		}

	}

	return nil
}

func (s *Shutter) DAGUpdate(ctx context.Context, dag *saaf.DAG, src *saaf.SnapSource) {
	nodes := dag.Store()
	nodeCh := nodes.All()

	select {
	case node := <-nodeCh:
		pointer := node.Pointer()
		cnt := dag.GetRefs(pointer)
		if cnt == 0 {
			// dag unlink
			dag.Unlink(pointer)
			// src remove
			src.Remove(pointer)
			// cache store remove
			err := s.cd.DeleteBlock(ctx, pointer)
			if err != nil {
				log.Warnf("delete block from cache err: %s", err)
			}
		}
	}
}
