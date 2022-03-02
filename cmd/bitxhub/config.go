package main

import (
	"encoding/json"
	"fmt"

	"github.com/hokaccha/go-prettyjson"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/spf13/viper"
	"github.com/urfave/cli"
)

func configCMD() cli.Command {
	return cli.Command{
		Name:   "config",
		Usage:  "Show BitXHub config",
		Action: showConfig,
		Flags:  []cli.Flag{},
	}
}

func showConfig(ctx *cli.Context) error {
	repoRoot := ctx.GlobalString("repo")
	var err error
	if len(repoRoot) == 0 {
		if repoRoot, err = repo.PathRoot(); err != nil {
			return fmt.Errorf("pathRoot error: %w", err)
		}
	}

	cfg, err := repo.UnmarshalConfig(viper.New(), repoRoot, "")
	if err != nil {
		return fmt.Errorf("unmarshal config error: %w", err)
	}

	if ctx.NArg() == 0 {
		s, err := prettyjson.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("marshal config error: %w", err)
		}

		fmt.Println(string(s))

		return nil
	}
	m := make(map[string]interface{})
	data, err := cfg.Bytes()
	if err != nil {
		return fmt.Errorf("convert config to bytes failed: %w", err)
	}

	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("unmarshal data error: %w", err)
	}

	v := m[ctx.Args()[0]]

	s, err := prettyjson.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal config error: %w", err)
	}

	fmt.Println(string(s))

	return nil
}
