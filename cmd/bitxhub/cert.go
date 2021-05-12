package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/coreos/etcd/pkg/fileutil"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub/internal/repo"
	libp2pcert "github.com/meshplus/go-libp2p-cert"
	"github.com/urfave/cli"
)

var certCMD = cli.Command{
	Name:  "cert",
	Usage: "Certification tools",
	Subcommands: cli.Commands{
		caCMD,
		csrCMD,
		issueCMD,
		parseCMD,
		privCMD,
		verifyCMD,
	},
}

var caCMD = cli.Command{
	Name:  "ca",
	Usage: "Generate ca cert and private key",
	Action: func(ctx *cli.Context) error {
		privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return err
		}

		priKeyEncode, err := x509.MarshalECPrivateKey(privKey)
		if err != nil {
			return err
		}

		f, err := os.Create("./ca.priv")
		if err != nil {
			return err
		}
		defer f.Close()

		err = pem.Encode(f, &pem.Block{Type: "EC PRIVATE KEY", Bytes: priKeyEncode})
		if err != nil {
			return err
		}

		c, err := libp2pcert.GenerateCert(privKey, true, "Hyperchain")
		if err != nil {
			return err
		}

		x509certEncode, err := x509.CreateCertificate(rand.Reader, c, c, privKey.Public(), privKey)
		if err != nil {
			return err
		}

		f, err = os.Create("./ca.cert")
		if err != nil {
			return err
		}
		defer f.Close()

		return pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: x509certEncode})
	},
}

var csrCMD = cli.Command{
	Name:  "csr",
	Usage: "Generate csr file",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:     "key",
			Usage:    "Specify Secp256r1 private key path",
			Required: true,
		},
		cli.StringFlag{
			Name:     "org",
			Usage:    "Specify organization name",
			Required: true,
		},
		cli.StringFlag{
			Name:  "target",
			Usage: "Specific target directory",
		},
	},
	Action: func(ctx *cli.Context) error {
		org := ctx.String("org")
		privPath := ctx.String("key")
		target := ctx.String("target")

		privData, err := ioutil.ReadFile(privPath)
		if err != nil {
			return err
		}
		block, _ := pem.Decode(privData)
		privKey, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return fmt.Errorf("Error occured when parsing private key. Please make sure it's secp256r1 private key.")
		}

		template := &x509.CertificateRequest{
			Subject: pkix.Name{
				Country:            []string{"CN"},
				Locality:           []string{"HangZhou"},
				Province:           []string{"ZheJiang"},
				OrganizationalUnit: []string{"BitXHub"},
				Organization:       []string{org},
				StreetAddress:      []string{"street", "address"},
				PostalCode:         []string{"324000"},
				CommonName:         "BitXHub",
			},
		}
		data, err := x509.CreateCertificateRequest(rand.Reader, template, privKey)
		if err != nil {
			return err
		}

		name := getFileName(privPath)

		path := filepath.Join(target, fmt.Sprintf("%s.csr", name))
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		defer f.Close()

		return pem.Encode(f, &pem.Block{Type: "CSR", Bytes: data})
	},
}

var issueCMD = cli.Command{
	Name:  "issue",
	Usage: "Issue certification by ca",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:     "csr",
			Usage:    "Specify csr path",
			Required: true,
		},
		cli.StringFlag{
			Name:  "is_ca",
			Usage: "Specify whether it's ca",
		},
		cli.StringFlag{
			Name:     "key",
			Usage:    "Specify ca's secp256r1 private key path",
			Required: true,
		},
		cli.StringFlag{
			Name:     "cert",
			Usage:    "Specify ca certification path",
			Required: true,
		},
		cli.StringFlag{
			Name:  "target",
			Usage: "Specific target directory",
		},
	},
	Action: func(ctx *cli.Context) error {
		csrPath := ctx.String("csr")
		isCA := ctx.Bool("is_ca")
		privPath := ctx.String("key")
		certPath := ctx.String("cert")
		target := ctx.String("target")

		privData, err := ioutil.ReadFile(privPath)
		if err != nil {
			return fmt.Errorf("read ca private key: %w", err)
		}
		block, _ := pem.Decode(privData)
		privKey, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return fmt.Errorf("Error occured when parsing private key. Please make sure it's secp256r1 private key.")
		}

		caCertData, err := ioutil.ReadFile(certPath)
		if err != nil {
			return err
		}
		block, _ = pem.Decode(caCertData)
		caCert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return fmt.Errorf("parse ca cert: %w", err)
		}

		csrData, err := ioutil.ReadFile(csrPath)
		if err != nil {
			return fmt.Errorf("read csr: %w", err)
		}

		block, _ = pem.Decode(csrData)

		csr, err := x509.ParseCertificateRequest(block.Bytes)
		if err != nil {
			return fmt.Errorf("parse csr: %w", err)
		}

		if err := csr.CheckSignature(); err != nil {
			return fmt.Errorf("wrong csr sign: %w", err)
		}

		sn, err := rand.Int(rand.Reader, big.NewInt(1000000))
		if err != nil {
			return err
		}

		notBefore := time.Now().Add(-5 * time.Minute).UTC()
		template := &x509.Certificate{
			Signature:             csr.Signature,
			SignatureAlgorithm:    csr.SignatureAlgorithm,
			PublicKey:             csr.PublicKey,
			PublicKeyAlgorithm:    csr.PublicKeyAlgorithm,
			SerialNumber:          sn,
			NotBefore:             notBefore,
			NotAfter:              notBefore.Add(50 * 365 * 24 * time.Hour).UTC(),
			BasicConstraintsValid: true,
			IsCA:                  isCA,
			Issuer:                caCert.Subject,
			KeyUsage: x509.KeyUsageDigitalSignature |
				x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign |
				x509.KeyUsageCRLSign,
			ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
			Subject:     csr.Subject,
		}

		x509certEncode, err := x509.CreateCertificate(rand.Reader, template, caCert, csr.PublicKey, privKey)
		if err != nil {
			return fmt.Errorf("create cert: %w", err)
		}

		name := getFileName(csrPath)

		path := filepath.Join(target, fmt.Sprintf("%s.cert", name))
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		defer f.Close()

		return pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: x509certEncode})
	},
}

var parseCMD = cli.Command{
	Name:  "parse",
	Usage: "parse certification",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:     "path",
			Usage:    "certification path",
			Required: true,
		},
	},
	Action: func(ctx *cli.Context) error {
		path := ctx.String("path")

		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		block, _ := pem.Decode(data)
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return fmt.Errorf("parse cert: %w", err)
		}

		ret, err := json.Marshal(cert)
		if err != nil {
			return err
		}

		fmt.Println(string(ret))

		return nil
	},
}

var privCMD = cli.Command{
	Name:  "priv",
	Usage: "Generate and show private key for certificate",
	Subcommands: []cli.Command{
		{
			Name:  "gen",
			Usage: "Create new private key",
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
				return generatePrivKey(ctx, crypto.ECDSA_P256)
			},
		},
		{
			Name:  "pid",
			Usage: "Show pid from private key",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:     "path",
					Usage:    "Specific private key path",
					Required: true,
				},
			},
			Action: func(ctx *cli.Context) error {
				privPath := ctx.String("path")

				pid, err := repo.GetPidFromPrivFile(privPath)
				if err != nil {
					return err
				}

				fmt.Println(pid)
				return nil
			},
		},
	},
}

var verifyCMD = cli.Command{
	Name:  "verify",
	Usage: "verify cert",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:     "sub",
			Usage:    "sub cert path",
			Required: true,
		},
		cli.StringFlag{
			Name:     "ca",
			Usage:    "ca cert path",
			Required: true,
		},
	},
	Action: func(ctx *cli.Context) error {
		subPath := ctx.String("sub")
		caPath := ctx.String("ca")

		subCertData, err := ioutil.ReadFile(subPath)
		if err != nil {
			return err
		}
		block, _ := pem.Decode(subCertData)
		subCert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return fmt.Errorf("parse sub cert: %w", err)
		}

		caCertData, err := ioutil.ReadFile(caPath)
		if err != nil {
			return err
		}
		block, _ = pem.Decode(caCertData)
		caCert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return fmt.Errorf("parse ca cert: %w", err)
		}

		return subCert.CheckSignatureFrom(caCert)
	},
}

func getFileName(path string) string {
	def := "default"
	name := filepath.Base(path)
	bs := strings.Split(name, ".")
	if len(bs) != 2 {
		return def
	}

	return bs[0]
}

func generatePrivKey(ctx *cli.Context, opt crypto.KeyType) error {
	name := ctx.String("name")
	target := ctx.String("target")

	target, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("get absolute key path: %w", err)
	}

	privKey, err := asym.GenerateKeyPair(opt)
	if err != nil {
		return fmt.Errorf("generate key: %w", err)
	}

	priKeyEncode, err := privKey.Bytes()
	if err != nil {
		return fmt.Errorf("marshal key: %w", err)
	}

	if !fileutil.Exist(target) {
		err := os.MkdirAll(target, 0755)
		if err != nil {
			return fmt.Errorf("create folder: %w", err)
		}
	}
	path := filepath.Join(target, fmt.Sprintf("%s.priv", name))
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}

	err = pem.Encode(f, &pem.Block{Type: "EC PRIVATE KEY", Bytes: priKeyEncode})
	if err != nil {
		return fmt.Errorf("pem encode: %w", err)
	}

	fmt.Printf("%s.priv key is generated under directory %s\n", name, target)
	return nil
}
