package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/urfave/cli"
)

func initCMD() cli.Command {
	return cli.Command{
		Name:   "init",
		Usage:  "Initialize BitXHub local configuration",
		Action: initialize,
	}
}

func initialize(ctx *cli.Context) error {
	repoRoot, err := repo.PathRootWithDefault(ctx.GlobalString("repo"))
	if err != nil {
		return fmt.Errorf("pathRootWithDefault error: %w", err)
	}

	fmt.Printf("initializing bitxhub at %s\n", repoRoot)

	if repo.Initialized(repoRoot) {
		fmt.Println("bitxhub configuration file already exists")
		fmt.Println("reinitializing would overwrite your configuration, Y/N?")
		input := bufio.NewScanner(os.Stdin)
		input.Scan()
		if input.Text() == "Y" || input.Text() == "y" {
			return repo.Initialize(repoRoot)
		}
		return nil
	}

	return repo.Initialize(repoRoot)
}
