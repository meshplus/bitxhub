package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"

	"github.com/axiomesh/axiom-kit/fileutil"
	"github.com/axiomesh/axiom/pkg/repo"
)

var configCMD = &cli.Command{
	Name:  "config",
	Usage: "The config manage commands",
	Subcommands: []*cli.Command{
		{
			Name:   "generate",
			Usage:  "Generate default config and node private key",
			Action: generate,
			Flags: []cli.Flag{
				&cli.IntFlag{
					Name:     "default-node-index",
					Usage:    "use default node config by specified index(1,2,3,4), regenerate if not specified",
					Required: false,
				},
				&cli.BoolFlag{
					Name:     "solo",
					Usage:    "generate solo config if specified",
					Required: false,
				},
			},
		},
		{
			Name:   "p2p-addr",
			Usage:  "Show p2p connection address",
			Action: p2pAddr,
		},
		{
			Name:   "node-id",
			Usage:  "show node id",
			Action: nodeID,
		},
		{
			Name:   "show",
			Usage:  "Show the complete config processed by the environment variable",
			Action: show,
		},
		{
			Name:   "show-order",
			Usage:  "Show the complete order config processed by the environment variable",
			Action: showOrder,
		},
		{
			Name:   "check",
			Usage:  "Check if the config file is valid",
			Action: check,
		},
		{
			Name:   "rewrite-with-env",
			Usage:  "Rewrite config with env",
			Action: rewriteWithEnv,
		},
	},
}

func generate(ctx *cli.Context) error {
	p, err := getRootPath(ctx)
	if err != nil {
		return err
	}
	existConfig := fileutil.Exist(filepath.Join(p, repo.CfgFileName))
	if existConfig {
		fmt.Println("axiom repo already exists")
		return nil
	}

	if !fileutil.Exist(p) {
		err = os.MkdirAll(p, 0755)
		if err != nil {
			return err
		}
	}

	nodeIndex := ctx.Int("default-node-index")
	r, err := repo.DefaultWithNodeIndex(p, nodeIndex-1)
	if err != nil {
		return err
	}
	if ctx.Bool("solo") {
		r.Config.Order.Type = repo.OrderTypeSolo
	}
	if err := r.Flush(); err != nil {
		return err
	}

	id, err := repo.KeyToNodeID(r.NodeKey)
	if err != nil {
		return err
	}

	fmt.Printf("initializing axiom at %s\n", p)
	fmt.Println("generate node-id:", id)
	return nil
}

func nodeID(ctx *cli.Context) error {
	p, err := getRootPath(ctx)
	if err != nil {
		return err
	}
	existConfig := fileutil.Exist(p)
	if !existConfig {
		fmt.Println("axiom repo not exist")
		return nil
	}

	r, err := repo.Load(p)
	if err != nil {
		return err
	}
	id, err := repo.KeyToNodeID(r.NodeKey)
	if err != nil {
		return err
	}
	fmt.Println(id)
	return nil
}

func p2pAddr(ctx *cli.Context) error {
	p, err := getRootPath(ctx)
	if err != nil {
		return err
	}
	existConfig := fileutil.Exist(p)
	if !existConfig {
		fmt.Println("axiom repo not exist")
		return nil
	}

	r, err := repo.Load(p)
	if err != nil {
		return err
	}
	fmt.Printf("/ip4/0.0.0.0/tcp/%d/p2p/%s\n", r.Config.Port.P2P, r.P2PID)
	return nil
}

func show(ctx *cli.Context) error {
	p, err := getRootPath(ctx)
	if err != nil {
		return err
	}
	existConfig := fileutil.Exist(p)
	if !existConfig {
		fmt.Println("axiom repo not exist")
		return nil
	}

	r, err := repo.Load(p)
	if err != nil {
		return err
	}
	str, err := repo.MarshalConfig(r.Config)
	if err != nil {
		return err
	}
	fmt.Println(str)
	return nil
}

func showOrder(ctx *cli.Context) error {
	p, err := getRootPath(ctx)
	if err != nil {
		return err
	}
	existConfig := fileutil.Exist(p)
	if !existConfig {
		fmt.Println("axiom repo not exist")
		return nil
	}

	r, err := repo.Load(p)
	if err != nil {
		return err
	}
	str, err := repo.MarshalConfig(r.OrderConfig)
	if err != nil {
		return err
	}
	fmt.Println(str)
	return nil
}

func check(ctx *cli.Context) error {
	p, err := getRootPath(ctx)
	if err != nil {
		return err
	}
	existConfig := fileutil.Exist(p)
	if !existConfig {
		fmt.Println("axiom repo not exist")
		return nil
	}

	_, err = repo.Load(p)
	if err != nil {
		fmt.Println("config file format error, please check:", err)
		os.Exit(1)
		return nil
	}

	return nil
}

func rewriteWithEnv(ctx *cli.Context) error {
	p, err := getRootPath(ctx)
	if err != nil {
		return err
	}
	existConfig := fileutil.Exist(p)
	if !existConfig {
		fmt.Println("axiom repo not exist")
		return nil
	}

	r, err := repo.Load(p)
	if err != nil {
		return err
	}
	if err := r.Flush(); err != nil {
		return err
	}
	return nil
}

func getRootPath(ctx *cli.Context) (string, error) {
	p := ctx.String("repo")

	var err error
	if p == "" {
		p, err = repo.LoadRepoRootFromEnv(p)
		if err != nil {
			return "", err
		}
	}
	return p, nil
}
