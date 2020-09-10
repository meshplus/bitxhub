package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/bitxhub/pkg/cert"
	"github.com/urfave/cli"
)

func keyCMD() cli.Command {
	return cli.Command{
		Name:  "key",
		Usage: "Create and show key information",
		Subcommands: []cli.Command{
			{
				Name:  "gen",
				Usage: "Create new Secp256k1 private key",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "name",
						Usage:    "Specific private key name",
						Required: true,
					},
					cli.StringFlag{
						Name:  "target",
						Usage: "Specific target directory",
					},
				},
				Action: func(ctx *cli.Context) error {
					return generatePrivKey(ctx, crypto.Secp256k1)
				},
			},
			{
				Name:  "convert",
				Usage: "Convert new key file from private key",
				Flags: []cli.Flag{
					cli.BoolFlag{
						Name:  "save,s",
						Usage: "save key into repo",
					},
					cli.StringFlag{
						Name:     "priv",
						Usage:    "private key path",
						Required: true,
					},
				},
				Action: convertKey,
			},
			{
				Name:   "show",
				Usage:  "Show key from cert",
				Action: showKey,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "path",
						Usage: "Node Path",
					},
				},
			},
			{
				Name:   "address",
				Usage:  "Show address from private key",
				Action: getAddress,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "path",
						Usage:    "Specific private key path",
						Required: true,
					},
				},
			},
		},
	}
}

func convertKey(ctx *cli.Context) error {
	privPath := ctx.String("priv")

	data, err := ioutil.ReadFile(privPath)
	if err != nil {
		return fmt.Errorf("read private key: %w", err)
	}

	privKey, err := cert.ParsePrivateKey(data, crypto.Secp256k1)
	if err != nil {
		return err
	}

	if ctx.Bool("save") {
		repoRoot, err := repo.PathRootWithDefault(ctx.GlobalString("repo"))
		if err != nil {
			return err
		}

		keyPath := filepath.Join(repoRoot, repo.KeyName)
		if err := asym.StorePrivateKey(privKey, keyPath, "bitxhub"); err != nil {
			return err
		}
	} else {
		keyStore, err := asym.GenKeyStore(privKey, "bitxhub")
		if err != nil {
			return err
		}

		pretty, err := keyStore.Pretty()
		if err != nil {
			return err
		}

		fmt.Println(pretty)
	}

	return nil
}

func showKey(ctx *cli.Context) error {
	repoRoot, err := repo.PathRootWithDefault(ctx.GlobalString("repo"))
	if err != nil {
		return err
	}

	keyPath := filepath.Join(repoRoot, repo.KeyName)
	data, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return err
	}

	fmt.Println(string(data))

	return nil
}

func getAddress(ctx *cli.Context) error {
	privPath := ctx.String("path")

	data, err := ioutil.ReadFile(privPath)
	if err != nil {
		return fmt.Errorf("read private key: %w", err)
	}

	privKey, err := cert.ParsePrivateKey(data, crypto.Secp256k1)
	if err != nil {
		return err
	}

	addr, err := privKey.PublicKey().Address()
	if err != nil {
		return err
	}

	fmt.Println(addr.String())

	return nil
}
