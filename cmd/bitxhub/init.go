package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/gobuffalo/packd"
	"github.com/gobuffalo/packr"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/fileutil"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/urfave/cli"
)

const (
	packPath = "../../config"
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
		return err
	}

	fmt.Printf("initializing bitxhub at %s\n", repoRoot)

	if Initialized(repoRoot) {
		fmt.Println("bitxhub configuration file already exists")
		fmt.Println("reinitializing would overwrite your configuration, Y/N?")
		input := bufio.NewScanner(os.Stdin)
		input.Scan()
		if input.Text() == "Y" || input.Text() == "y" {
			return Initialize(repoRoot)
		}
		return nil
	}

	return Initialize(repoRoot)
}

func Initialize(repoRoot string) error {
	box := packr.NewBox(packPath)
	if err := box.Walk(func(s string, file packd.File) error {
		p := filepath.Join(repoRoot, s)
		dir := filepath.Dir(p)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			err := os.MkdirAll(dir, 0755)
			if err != nil {
				return err
			}
		}

		return ioutil.WriteFile(p, []byte(file.String()), 0644)
	}); err != nil {
		return err
	}

	certPath := filepath.Join(repoRoot, "certs")
	if err := generateNodeCert(certPath); err != nil {
		return fmt.Errorf("generate node cert error: %w", err)
	}

	if err := updateConfig(repoRoot); err != nil {
		return err
	}

	return nil
}

func Initialized(repoRoot string) bool {
	return fileutil.Exist(filepath.Join(repoRoot, repo.ConfigName))
}

func generateNodeCert(target string) error {
	caPrivPath := filepath.Join(target, "ca.priv")
	nodePrivPath := filepath.Join(target, fmt.Sprintf("%s.priv", repo.NodeName))
	nodeCsrPath := filepath.Join(target, fmt.Sprintf("%s.csr", repo.NodeName))
	agencyPrivPath := filepath.Join(target, fmt.Sprintf("%s.priv", repo.AgencyName))
	agencyCertPath := filepath.Join(target, fmt.Sprintf("%s.cert", repo.AgencyName))

	defer os.Remove(caPrivPath)
	defer os.Remove(agencyPrivPath)
	err := GeneratePrivKey(repo.NodeName, target, crypto.ECDSA_P256)
	if err != nil {
		return fmt.Errorf("bitxhub cert priv gen error: %s", err)
	}

	err = GenerateCsr(repo.NodeOrg, nodePrivPath, target)
	if err != nil {
		return fmt.Errorf("bitxhub cert csr error: %s", err)
	}

	err = CertIssue(nodeCsrPath, false, agencyPrivPath, agencyCertPath, target)
	if err != nil {
		return fmt.Errorf("bitxhub cert issue error: %s", err)
	}

	err = GeneratePrivKey(repo.KeyN, target, crypto.Secp256k1)
	if err != nil {
		return fmt.Errorf("bitxhub key pirv gen error: %s", err)
	}

	return nil
}

func updateConfig(target string) error {
	nodePrivPath := filepath.Join(target, repo.CertsDir, fmt.Sprintf("%s.priv", repo.NodeName))
	pid, err := repo.GetPidFromPrivFile(nodePrivPath)
	if err != nil {
		return fmt.Errorf("get pid error: %s", err)
	}

	keyPrivPath := filepath.Join(target, repo.CertsDir, fmt.Sprintf("%s.priv", repo.KeyN))
	addr, err := GetAddressFromPrivFile(keyPrivPath)
	if err != nil {
		return fmt.Errorf("get addr error: %s", err)
	}

	infos := repo.DefaultNetworkNodes()
	infos[1].Pid = pid
	infos[1].Account = addr
	err = repo.RewriteNetworkConfig(target, infos, false)
	if err != nil {
		return fmt.Errorf("rewrite network config error: %w", err)
	}

	err = repo.RewriteBitxhubConfigAddr(target, addr)
	if err != nil {
		return fmt.Errorf("rewrite bitxhub config error: %w", err)
	}

	return nil
}
