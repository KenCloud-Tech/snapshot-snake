package dep

import (
	"fmt"
	"github.com/snapshot_snake/common"
	"github.com/snapshot_snake/lib/cliex"
	"github.com/snapshot_snake/snapshot"
	"github.com/snapshot_snake/snapshot/saaf"
	"go.uber.org/fx"
	"os"
)

var (
	_ common.HeadNotifier = (*cliex.HeadSub)(nil)
)

func LoadConfig(path RepoPath) (snapshot.Config, error) {
	cfgPath := ConfigFilePath(path)
	var cfg snapshot.Config
	_, err := FromFile(cfgPath, &cfg)
	fmt.Println(cfg)
	if err != nil {
		return snapshot.Config{}, fmt.Errorf("read config from file %s: %w", cfgPath, err)
	}
	//set 'FULLNODE_API_INFO' env var
	lotusAPIInfo := fmt.Sprintf("%s:%s", cfg.LotusAPI.APIToken, cfg.LotusAPI.APIAddr)
	if err := os.Setenv("FULLNODE_API_INFO", lotusAPIInfo); err != nil {
		return snapshot.Config{}, fmt.Errorf("failed to set FULLNODE_API_INFO into os environment variable: %s", err)
	}
	return cfg, nil
}

type snapshotIn struct {
	fx.In
	Ctx GlobalContext

	Sub common.HeadNotifier
	Cs  common.DagStore

	Dag *saaf.DAG
	Src *saaf.SnapSource
}

func NewSnapshot(in snapshotIn) *snapshot.Shutter {
	return snapshot.New(in.Ctx, in.Sub, in.Cs, in.Dag, in.Src)
}
