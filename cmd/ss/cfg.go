package main

import (
	"fmt"
	"github.com/filecoin-project/lotus/node/config"
	"github.com/snapshot_snake/dep"
	"github.com/snapshot_snake/snapshot"
	"github.com/urfave/cli/v2"
	"os"
	"path/filepath"
)

var cfgCmd = &cli.Command{
	Name: "cfg",
	Subcommands: []*cli.Command{
		cfgInitCmd,
	},
}

var cfgInitCmd = &cli.Command{
	Name:  "init",
	Flags: []cli.Flag{},
	Action: func(cctx *cli.Context) error {
		rpath, err := dep.GetRepoPath(cctx)
		if err != nil {
			return err
		}

		cfgPath := dep.ConfigFilePath(rpath)

		_, err = os.Stat(cfgPath)
		if err == nil {
			log.Warn("config file already exists")
			return nil
		}

		log.Infof("init config: %s", cfgPath)

		if !os.IsNotExist(err) {
			return fmt.Errorf("fs error: %w", err)
		}

		err = os.MkdirAll(filepath.Dir(cfgPath), 0755)
		if err != nil {
			return fmt.Errorf("MkdirAll for %s: %w", cfgPath, err)
		}

		cfg := snapshot.DefaultConfig()
		content, err := config.ConfigComment(cfg)
		if err != nil {
			return fmt.Errorf("marshal default config: %w", err)
		}

		err = os.WriteFile(cfgPath, content, 0644)
		if err != nil {
			return fmt.Errorf("write config file: %w", err)
		}

		log.Info("init done")

		return nil
	},
}
