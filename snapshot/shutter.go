package snapshot

import (
	"context"
	"github.com/FIL_FIL_Snapshot/common"
	"github.com/FIL_FIL_Snapshot/snapshot/saaf"
	"github.com/filecoin-project/lotus/chain/types"
	lconfig "github.com/filecoin-project/lotus/node/config"
	logging "github.com/ipfs/go-log/v2"
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

func New(ctx context.Context, sub common.HeadNotifier, cs common.ChainStore, dag *saaf.DAG, src *saaf.FilFilSource) *Shutter {
	shutter := &Shutter{
		sub: sub,
		cs:  cs,
		dag: dag,
		src: src,
	}
	return shutter
}

type Shutter struct {
	sub common.HeadNotifier
	cs  common.ChainStore

	dag *saaf.DAG
	src *saaf.FilFilSource
}

func (s *Shutter) Run(ctx context.Context, doneCh <-chan struct{}, tsCh <-chan types.TipSetKey) {
	for {
		select {
		case <-doneCh:
			log.Info("quite head change loop")
			return
		case tsk, ok := <-tsCh:
			if !ok {
				log.Warn("tsk chan closed")
				return
			}
			// get ts
			ts, err := s.cs.LoadTipSet(ctx, tsk)
			if err != nil {
				log.Errorf("failed to load tipset %s: %s", tsk, err)
				continue
			}

			log.Infow("incoming tipset", "tsk", tsk, "height", ts.Height())
			estart := time.Now()
			// build filfil dag
			if err := saaf.FilFilDAGBuilder(ctx, ts, s.dag, s.src); err != nil {
				log.Errorf("failed to build filfil dag err: %s", err)
			}
			log.Infow("done tipset building", "tsk", tsk, "height", ts.Height(), "elapsed", time.Now().Sub(estart))
		}
	}
}
