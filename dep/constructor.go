package dep

import (
	"fmt"
	"github.com/FIL_FIL_Snapshot/common"
	"github.com/FIL_FIL_Snapshot/lib/bsex"
	"github.com/FIL_FIL_Snapshot/lib/cliex"
	"github.com/FIL_FIL_Snapshot/snapshot"
	"github.com/FIL_FIL_Snapshot/snapshot/saaf"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	levelds "github.com/ipfs/go-ds-leveldb"
	ldbopts "github.com/syndtr/goleveldb/leveldb/opt"
	"go.uber.org/fx"
	"os"
	"strconv"
)

var (
	_ common.HeadNotifier = (*cliex.HeadSub)(nil)
)

func LoadConfig(path RepoPath) (snapshot.Config, error) {
	cfgPath := ConfigFilePath(path)
	fmt.Println("LoadConfig...")
	var cfg snapshot.Config
	_, err := FromFile(cfgPath, &cfg)
	if err != nil {
		return snapshot.Config{}, fmt.Errorf("read config from file %s: %w", cfgPath, err)
	}
	//set 'FULLNODE_API_INFO' env var
	lotusAPIInfo := fmt.Sprintf("%s:%s", cfg.LotusAPI.APIToken, cfg.LotusAPI.APIAddr)
	fmt.Println(lotusAPIInfo)
	if err := os.Setenv("FULLNODE_API_INFO", lotusAPIInfo); err != nil {
		return snapshot.Config{}, fmt.Errorf("failed to set FULLNODE_API_INFO into os environment variable: %s", err)
	}
	return cfg, nil
}

type snapshotIn struct {
	fx.In
	Ctx GlobalContext

	Sub common.HeadNotifier
	Cs  common.ChainStore

	Dag *saaf.DAG
	Src *saaf.FilFilSource
}

func NewSnapshot(in snapshotIn) *snapshot.Shutter {
	return snapshot.New(in.Ctx, in.Sub, in.Cs, in.Dag, in.Src)
}

type WrapAPIBlockstore struct {
	blockstore.Blockstore
}

func ChainIOBlockstore(full v0api.FullNode) (dtypes.HotBlockstore, error) {
	bs := blockstore.NewAPIBlockstore(full)
	wrapBlockStore := &WrapAPIBlockstore{
		bs,
	}

	cacheSize := 1 << 25
	if size := os.Getenv("FILFIL_CACHE_SIZE"); size != "" {
		var err error
		cacheSize, err = strconv.Atoi(size)
		if err != nil {
			panic(err)
		}
	}

	cached, err := bsex.NewCachedBlockstore(cacheSize, wrapBlockStore)
	if err != nil {
		return nil, err
	}

	return cached, nil
}

func levelDs(path string, readonly bool) (dtypes.MetadataDS, error) {
	return levelds.NewDatastore(path, &levelds.Options{
		Compression: ldbopts.NoCompression,
		NoSync:      false,
		Strict:      ldbopts.StrictAll,
		ReadOnly:    readonly,
	})
}

func OpenSegmentDS(rpath RepoPath) (SegmentMetaDS, error) {
	return levelDs(SegmentMetaDSPath(rpath), false)
}
