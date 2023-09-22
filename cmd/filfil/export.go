package main

import (
	"context"
	"github.com/FIL_FIL_Snapshot/api"
	"github.com/FIL_FIL_Snapshot/snapshot/saaf"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
)

var exportCmd = &cli.Command{
	Name: "export",
	Subcommands: []*cli.Command{
		exportSnapshotCmd,
	},
}

var exportSnapshotCmd = &cli.Command{
	Name: "snapshot",
	Flags: []cli.Flag{
		&cli.Int64Flag{
			Name:  "height",
			Usage: "export data from this height",
		},
		&cli.Int64Flag{
			Name:  "recent-stateroots",
			Usage: "specify the number of recent state roots to include in the export",
		},
	},
	Action: func(cctx *cli.Context) error {
		// todo get api
		apiv0, _, err := GetAPIV0(cctx)
		ctx := context.Background()

		// CreateExportFile
		fi, err := CreateExportFile(cctx.App, cctx.Args().First())
		if err != nil {
			log.Errorf("create export file err: %s", err)
			return err
		}

		// load tipset form dag
		h := cctx.Int64("height")
		if h == 0 {

		}
		ts, err := LoadTipSet(ctx, cctx, apiv0)
		if err != nil {
			return err
		}

		stream, err := apiv0.FilFilDagExport(ctx, saaf.Height(h), ts.Key())
		if err != nil {
			return err
		}

		var last bool
		for b := range stream {
			last = len(b) == 0

			_, err := fi.Write(b)
			if err != nil {
				return err
			}
		}

		if !last {
			return xerrors.Errorf("incomplete export (remote connection lost?)")
		}

		return nil
	},
}

func LoadTipSet(ctx context.Context, cctx *cli.Context, api api.FilFilAPI) (*types.TipSet, error) {
	tss := cctx.String("tipset")
	if tss == "" {
		//return api.ChainHead(ctx)
		nodes, err := api.GetDagNode(ctx, 1)
		if err != nil {
			return nil, err
		}
		// get from cache or build a ts
		key := types.NewTipSetKey(saaf.PointersToCids(nodes)...)
		// load tipset
		ts, err := api.ChainGetTipSet(ctx, key)
		if err != nil {
			return nil, err
		}
		return ts, err
	}
	return nil, nil
}
