package main

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"

	"github.com/hokaccha/go-prettyjson"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/urfave/cli"
)

func configCMD() cli.Command {
	return cli.Command{
		Name:   "config",
		Usage:  "Operate bitxhub config",
		Action: showConfig,
		Flags:  []cli.Flag{},
	}
}

func showConfig(ctx *cli.Context) error {
	repoRoot := ctx.GlobalString("repo")
	var err error
	if len(repoRoot) == 0 {
		if repoRoot, err = repo.PathRoot(); err != nil {
			return err
		}
	}

	cfg, err := repo.UnmarshalConfig(viper.New(), repoRoot, "")
	if err != nil {
		return err
	}

	if ctx.NArg() == 0 {
		s, err := prettyjson.Marshal(cfg)
		if err != nil {
			return err
		}

		fmt.Println(string(s))

		return nil
	}
	m := make(map[string]interface{})
	data, err := cfg.Bytes()
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	v := m[ctx.Args()[0]]

	s, err := prettyjson.Marshal(v)
	if err != nil {
		return err
	}

	fmt.Println(string(s))

	return nil
}
