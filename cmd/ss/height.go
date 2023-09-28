package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
)

var heightCmd = &cli.Command{
	Name: "height",
	Action: func(cctx *cli.Context) error {
		apiv0, _, err := GetAPIV0(cctx)
		if err != nil {
			return fmt.Errorf("get apiv0 err: %s", err)
		}

		cacheRange, _ := apiv0.GetCacheRange()
		log.Infof("the current cache scope is %d", cacheRange)
		return nil
	},
}
