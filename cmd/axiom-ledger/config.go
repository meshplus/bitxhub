package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/axiomesh/axiom-kit/fileutil"
	"github.com/axiomesh/axiom-ledger/pkg/repo"
)

var configCMD = &cli.Command{
	Name:  "config",
	Usage: "The config manage commands",
	Subcommands: []*cli.Command{
		{
			Name:   "generate",
			Usage:  "Generate default config and node private key(if not exist)",
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
				&cli.BoolFlag{
					Name:     "epoch-enable",
					Usage:    "generate epoch and wrf enabled config if specified",
					Required: false,
				},
			},
		},
		{
			Name:   "generate-account-key",
			Usage:  "Generate account private key",
			Action: generateAccountKey,
		},
		{
			Name:   "node-info",
			Usage:  "show node info",
			Action: nodeInfo,
		},
		{
			Name:   "show",
			Usage:  "Show the complete config processed by the environment variable",
			Action: show,
		},
		{
			Name:   "show-consensus",
			Usage:  "Show the complete consensus config processed by the environment variable",
			Action: showConsensus,
		},
		{
			Name:   "check",
			Usage:  "Check if the config file is valid",
			Action: check,
		},
	},
}

func generate(ctx *cli.Context) error {
	p, err := getRootPath(ctx)
	if err != nil {
		return err
	}
	if fileutil.Exist(filepath.Join(p, repo.CfgFileName)) {
		fmt.Println("axiom-ledger repo already exists")
		return nil
	}

	if !fileutil.Exist(p) {
		err = os.MkdirAll(p, 0755)
		if err != nil {
			return err
		}
	}

	epochEnable := ctx.Bool("epoch-enable")
	nodeIndex := ctx.Int("default-node-index")
	r, err := repo.DefaultWithNodeIndex(p, nodeIndex-1, epochEnable)
	if err != nil {
		return err
	}
	if ctx.Bool("solo") {
		r.Config.Consensus.Type = repo.ConsensusTypeSolo
	}
	if err := r.Flush(); err != nil {
		return err
	}

	r.PrintNodeInfo()
	return nil
}

func generateAccountKey(ctx *cli.Context) error {
	p, err := getRootPath(ctx)
	if err != nil {
		return err
	}
	if !fileutil.Exist(filepath.Join(p, repo.CfgFileName)) {
		fmt.Println("axiom-ledger repo not exist")
		return nil
	}

	keyPath := path.Join(p, repo.AccountKeyFileName)
	if fileutil.Exist(keyPath) {
		fmt.Printf("%s exists, do you want to overwrite it? y/n\n", keyPath)
		var choice string
		if _, err := fmt.Scanln(&choice); err != nil {
			return err
		}
		if choice != "y" {
			return errors.New("interrupt by user")
		}
	}
	key, err := repo.GenerateKey()
	if err != nil {
		return err
	}
	if err := repo.WriteKey(keyPath, key); err != nil {
		return err
	}
	fmt.Println("generate account-addr:", ethcrypto.PubkeyToAddress(key.PublicKey).String())
	fmt.Println("generate account-key:", repo.KeyString(key))

	return nil
}

func nodeInfo(ctx *cli.Context) error {
	p, err := getRootPath(ctx)
	if err != nil {
		return err
	}
	if !fileutil.Exist(filepath.Join(p, repo.CfgFileName)) {
		fmt.Println("axiom-ledger repo not exist")
		return nil
	}

	r, err := repo.Load(p)
	if err != nil {
		return err
	}

	r.PrintNodeInfo()
	return nil
}

func show(ctx *cli.Context) error {
	p, err := getRootPath(ctx)
	if err != nil {
		return err
	}
	if !fileutil.Exist(filepath.Join(p, repo.CfgFileName)) {
		fmt.Println("axiom-ledger repo not exist")
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

func showConsensus(ctx *cli.Context) error {
	p, err := getRootPath(ctx)
	if err != nil {
		return err
	}
	if !fileutil.Exist(filepath.Join(p, repo.CfgFileName)) {
		fmt.Println("axiom-ledger repo not exist")
		return nil
	}

	r, err := repo.Load(p)
	if err != nil {
		return err
	}
	str, err := repo.MarshalConfig(r.ConsensusConfig)
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
	if !fileutil.Exist(filepath.Join(p, repo.CfgFileName)) {
		fmt.Println("axiom-ledger repo not exist")
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
