package main

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/urfave/cli/v2"

	"github.com/axiomesh/axiom/pkg/repo"
)

const fileName = "account.key"

var accountCMD = &cli.Command{
	Name:  "account",
	Usage: "Account management command",
	Subcommands: []*cli.Command{
		{
			Name:   "generate",
			Usage:  "Generate an account",
			Action: generateAccount,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "output",
					Usage:    "Where the private key file save, if not specified, file will save to the working dir",
					Required: false,
				},
			},
		},
		{
			Name:   "print",
			Usage:  "Parse and print account detail from local file",
			Action: parseAccount,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "path",
					Usage:    "Read an account from given path if specified, otherwise will read from working directory",
					Required: false,
				},
			},
		},
	},
}

func generateAccount(ctx *cli.Context) error {
	key, err := ethcrypto.GenerateKey()
	if err != nil {
		return err
	}
	fmt.Printf("generate account success!\nyour addr is %s\n", ethcrypto.PubkeyToAddress(key.PublicKey))
	return writeAccountToFile(getSavePath(ctx), key)
}

func parseAccount(ctx *cli.Context) error {
	fromPath := ctx.String("path")
	if fromPath == "" {
		fromPath = fileName
	}
	content, err := os.ReadFile(fromPath)
	if err != nil {
		return err
	}
	trimmed := strings.TrimPrefix(string(content), "0x")
	key, err := repo.ParseKey([]byte(trimmed))
	if err != nil {
		return err
	}
	fmt.Printf("parse file success!\nyour addr is %s\n", ethcrypto.PubkeyToAddress(key.PublicKey))
	return nil
}

func getSavePath(ctx *cli.Context) string {
	outPath := ctx.String("output")
	return path.Join(outPath, fileName)
}

func writeAccountToFile(savePath string, key *ecdsa.PrivateKey) error {
	// write result to file
	var f *os.File
	defer f.Close()

	f, err := os.OpenFile(savePath, os.O_WRONLY, 0644)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if f, err = os.Create(savePath); err != nil {
			return err
		}
	} else {
		fmt.Printf("%s exists, do you want to overwrite it? yes/no\n", savePath)
		var choice string
		if _, err := fmt.Scanln(&choice); err != nil {
			return err
		}
		if choice != "yes" {
			return errors.New("interrupt by user")
		}
	}

	_, err = f.WriteString(hexutil.Encode(key.D.Bytes()))
	return err
}
