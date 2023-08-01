package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/axiomesh/axiom-kit/crypto"
	"github.com/axiomesh/axiom-kit/crypto/asym"
	"github.com/axiomesh/axiom-kit/fileutil"
	"github.com/axiomesh/axiom/internal/repo"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli"
)

func keyCMD() cli.Command {
	return cli.Command{
		Name:  "key",
		Usage: "Axiom private key tools",
		Subcommands: []cli.Command{
			{
				Name:  "gen",
				Usage: "Generate new private key in specified directory",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "target",
						Usage: "Specify target directory",
					},
					cli.StringFlag{
						Name:     "passwd",
						Usage:    "Specify password",
						Required: false,
					},
					cli.StringFlag{
						Name:     "algo",
						Usage:    "Specify crypto algorithm",
						Value:    "Secp256k1",
						Required: false,
					},
				},
				Action: genPrivKey,
			},
			{
				Name:  "show",
				Usage: "Show Axiom private key info",
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
				Action: showKey,
			},
			{
				Name:  "address",
				Usage: "Show address from Axiom private key",
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
				Action: getAddress,
			},
		},
	}
}

func genPrivKey(ctx *cli.Context) error {
	target := ctx.String("target")
	passwd := ctx.String("passwd")
	cryptoAlgo := ctx.String("algo")

	if passwd == "" {
		passwd = repo.DefaultPasswd
	}
	target, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("get absolute key path: %w", err)
	}

	cryptoType, err := crypto.CryptoNameToType(cryptoAlgo)
	if err != nil {
		return fmt.Errorf("change crypto name to type failed: %w", err)
	}

	if !asym.SupportedKeyType(cryptoType) {
		return fmt.Errorf("unsupport crypto algo:%s", cryptoAlgo)
	}

	privKey, err := asym.GenerateKeyPair(cryptoType)
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
		return fmt.Errorf("store private key failed: %w", err)
	}
	fmt.Printf("key.json key is generated under directory %s\n", target)
	return nil
}

func showKey(ctx *cli.Context) error {
	privPath := ctx.String("path")
	passwd := ctx.String("passwd")
	if passwd == "" {
		passwd = repo.DefaultPasswd
	}

	privKey, err := asym.RestorePrivateKey(privPath, passwd)
	if err != nil {
		return fmt.Errorf("restore private key failed: %w", err)
	}

	data, err := privKey.Bytes()
	if err != nil {
		return fmt.Errorf("convert private key to bytes failed: %w", err)
	}

	pubData, err := privKey.PublicKey().Bytes()
	if err != nil {
		return fmt.Errorf("convert public key to bytes failed: %w", err)
	}
	addr, err := privKey.PublicKey().Address()
	if err != nil {
		return fmt.Errorf("get address from public key failed: %w", err)
	}

	fmt.Printf("private key: %s\n", common.Bytes2Hex(data))
	fmt.Printf("public key: %s\n", common.Bytes2Hex(pubData))
	fmt.Printf("address: %s\n", addr)
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
		return fmt.Errorf("restore private key failed: %w", err)
	}

	addr, err := privKey.PublicKey().Address()
	if err != nil {
		return fmt.Errorf("get address from public key failed: %w", err)
	}

	fmt.Println(addr.String())

	return nil
}
