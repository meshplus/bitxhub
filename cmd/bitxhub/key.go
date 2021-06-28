package main

import (
	"fmt"
	"github.com/meshplus/bitxhub-kit/fileutil"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub/internal/repo"
	libp2pcert "github.com/meshplus/go-libp2p-cert"
	"github.com/urfave/cli"
)

func keyCMD() cli.Command {
	return cli.Command{
		Name:  "key",
		Usage: "Create and show key information",
		Subcommands: []cli.Command{
			{
				Name:  "gen",
				Usage: "Create new Secp256k1 private key in specified directory",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "target",
						Usage: "Specific target directory",
					},
					cli.StringFlag{
						Name:     "passwd",
						Usage:    "Specify password",
						Required: false,
					},
				},
				Action: genPrivKey,
			},
			{
				Name:  "convert",
				Usage: "Convert the Secp256k1 private key to BitXHub key format",
				Flags: []cli.Flag{
					cli.BoolFlag{
						Name:  "save,s",
						Usage: "Save BitXHub key into repo",
					},
					cli.StringFlag{
						Name:     "priv",
						Usage:    "Specify private key path",
						Required: true,
					},
					cli.StringFlag{
						Name:     "passwd",
						Usage:    "Specify password",
						Required: false,
					},
				},
				Action: convertKey,
			},
			{
				Name:   "show",
				Usage:  "Show BitXHub key from repo",
				Action: showKey,
			},
			{
				Name:   "address",
				Usage:  "Show address from Secp256k1 private key",
				Action: getAddress,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "path",
						Usage:    "Specify private key path",
						Required: true,
					},
					cli.StringFlag{
						Name:     "passwd",
						Usage:    "Specify password",
						Required: false,
					},
				},
			},
		},
	}
}


func genPrivKey(ctx *cli.Context) error {
	target := ctx.String("target")
	passwd := ctx.String("passwd")

	if passwd == "" {
		passwd = repo.DefaultPasswd
	}
	target, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("get absolute key path: %w", err)
	}

	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	if err != nil {
		return fmt.Errorf("generate key: %w", err)
	}

	if !fileutil.Exist(target) {
		err := os.MkdirAll(target, 0755)
		if err != nil {
			return fmt.Errorf("create folder: %w", err)
		}
	}
	path := filepath.Join(target, repo.KeyName)
	err = asym.StorePrivateKey(privKey, path, passwd)
	if err != nil {
		return err
	}
	fmt.Printf("key.json key is generated under directory %s\n", target)
	return nil
}

func convertKey(ctx *cli.Context) error {
	privPath := ctx.String("priv")
	passwd := ctx.String("passwd")

	if passwd == "" {
		passwd = repo.DefaultPasswd
	}

	data, err := ioutil.ReadFile(privPath)
	if err != nil {
		return fmt.Errorf("read private key: %w", err)
	}

	privKey, err := libp2pcert.ParsePrivateKey(data, crypto.Secp256k1)
	if err != nil {
		return err
	}

	if ctx.Bool("save") {
		repoRoot, err := repo.PathRootWithDefault(ctx.GlobalString("repo"))
		if err != nil {
			return err
		}

		keyPath := filepath.Join(repoRoot, repo.KeyName)
		if err := asym.StorePrivateKey(privKey, keyPath, passwd); err != nil {
			return err
		}
	} else {
		keyStore, err := asym.GenKeyStore(privKey, passwd)
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
	passwd := ctx.String("passwd")
	if passwd == "" {
		passwd = repo.DefaultPasswd
	}

	privKey, err := asym.RestorePrivateKey(privPath, passwd)
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
