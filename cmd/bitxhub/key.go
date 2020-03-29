package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/meshplus/bitxhub-kit/key"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/meshplus/bitxhub-kit/crypto/asym/ecdsa"
	"github.com/meshplus/bitxhub-kit/fileutil"
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
				Usage: "Create new key file from private key",
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
				Action: generateKey,
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
				Name:   "pid",
				Usage:  "Show pid from cert",
				Action: getPid,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "path",
						Usage: "Private Key Path",
					},
				},
			},
		},
	}
}

func generateKey(ctx *cli.Context) error {
	privPath := ctx.String("priv")

	data, err := ioutil.ReadFile(privPath)
	if err != nil {
		return fmt.Errorf("read private key: %w", err)
	}
	stdPriv, err := cert.ParsePrivateKey(data)
	if err != nil {
		return err
	}

	privKey := &ecdsa.PrivateKey{K: stdPriv}

	act, err := key.NewWithPrivateKey(privKey, "bitxhub")
	if err != nil {
		return fmt.Errorf("create account error: %s", err)
	}

	out, err := json.Marshal(act)
	if err != nil {
		return err
	}

	if ctx.Bool("save") {
		repoRoot, err := repo.PathRootWithDefault(ctx.GlobalString("repo"))
		if err != nil {
			return err
		}

		keyPath := filepath.Join(repoRoot, repo.KeyName)
		ok := fileutil.Exist(keyPath)
		if ok {
			fmt.Println("Key file already exists")
			fmt.Println("Recreate would overwrite your key, Y/N?")
			input := bufio.NewScanner(os.Stdin)
			input.Scan()
			if input.Text() == "Y" || input.Text() == "y" {
				err := ioutil.WriteFile(keyPath, out, os.ModePerm)
				if err != nil {
					return fmt.Errorf("write key file: %w", err)
				}
			}

			return nil
		}

		err = ioutil.WriteFile(keyPath, out, os.ModePerm)
		if err != nil {
			return fmt.Errorf("write key file: %w", err)
		}
	} else {
		fmt.Println(string(out))
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

func getPid(ctx *cli.Context) error {
	privPath := ctx.String("path")

	data, err := ioutil.ReadFile(privPath)
	if err != nil {
		return fmt.Errorf("read private key: %w", err)
	}
	stdPriv, err := cert.ParsePrivateKey(data)
	if err != nil {
		return err
	}

	_, pk, err := crypto.KeyPairFromStdKey(stdPriv)
	if err != nil {
		return err
	}

	pid, err := peer.IDFromPublicKey(pk)
	if err != nil {
		return err
	}

	fmt.Println(pid)

	return nil
}
