package main

import (
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/lib/lotuslog"
	"github.com/snapshot_snake/dep"
	"github.com/urfave/cli/v2"
	"os"
)

func main() {
	lotuslog.SetupLogLevels()

	app := &cli.App{
		Name:                 "snapshot sneak",
		Usage:                "a small incentivized data network overlay on top of filecoin specifically for filecoin snapshots",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			cfgCmd,
			daemonCmd,
			exportCmd,
			heightCmd,
		},
		Version: build.UserVersion(),
		Flags: []cli.Flag{
			dep.RepoFlag,
			dep.FullNodeAPIFlag,
		},
	}

	app.Setup()

	if err := app.Run(os.Args); err != nil {
		log.Errorf("cli error: %s", err)
		os.Exit(1)
	}
}
