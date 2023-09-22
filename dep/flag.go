package dep

import (
	"context"
	"fmt"
	"github.com/FIL_FIL_Snapshot/lib/ffx"
	"github.com/filecoin-project/lotus/api/v0api"
	cliutil "github.com/filecoin-project/lotus/cli/util"
	"github.com/filecoin-project/lotus/node/config"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"
	"os"
	"path/filepath"
)

// common flags
var (
	FullNodeAPIFlag = &cli.StringFlag{
		Name: "api-url",
	}

	RepoFlag = &cli.StringFlag{
		Name:  "fil-fil-repo",
		Usage: "repo path for filfil",
		Value: "~/.filfil",
	}
)

func InjectFullNode(cctx *cli.Context) ffx.Option {
	return ffx.Override(new(v0api.FullNode), func(lc fx.Lifecycle) (v0api.FullNode, error) {
		full, closer, err := cliutil.GetFullNodeAPI(cctx)
		if err != nil {
			return nil, err
		}

		lc.Append(fx.Hook{
			OnStop: func(_ context.Context) error {
				closer()
				return nil
			},
		})

		return full, nil
	})
}

func GetRepoPath(cctx *cli.Context) (RepoPath, error) {
	dir, err := homedir.Expand(cctx.String(RepoFlag.Name))
	if err != nil {
		return "", fmt.Errorf("expand homedir: %w", err)
	}

	if err := os.Mkdir(dir, 0755); err != nil {
		return "", fmt.Errorf("mkdir at %s: %w", dir, err)
	}

	return RepoPath(dir), nil
}

func ConfigFilePath(rpath RepoPath) string {
	return filepath.Join(string(rpath), "config.toml")
}

func InjectRepoPath(cctx *cli.Context) ffx.Option {
	return ffx.Override(new(RepoPath), func() (RepoPath, error) {
		dir, err := homedir.Expand(cctx.String(RepoFlag.Name))
		return RepoPath(dir), err
	})
}

// FromFile loads config from a specified file overriding defaults specified in
// the def parameter. If file does not exist or is empty defaults are assumed.
func FromFile(path string, def interface{}) (interface{}, error) {
	file, err := os.Open(path)
	switch {
	case os.IsNotExist(err):
		return def, nil
	case err != nil:
		return nil, err
	}

	defer file.Close() //nolint:errcheck // The file is RO
	return config.FromReader(file, def)
}
func SegmentMetaDSPath(rpath RepoPath) string {
	return filepath.Join(string(rpath), "filfilstore")
}
