package client

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/urfave/cli"
)

func httpGet(ctx *cli.Context, url string) ([]byte, error) {
	/* #nosec */
	var (
		client *http.Client
		err    error
	)
	certPath := ctx.GlobalString("cert")
	if certPath != "" {
		client, err = getHttpsClient(certPath)
		if err != nil {
			return nil, err
		}
	} else {
		client = http.DefaultClient
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}

	c, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = resp.Body.Close()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func httpPost(ctx *cli.Context, url string, data []byte) ([]byte, error) {
	var (
		client *http.Client
		err    error
	)
	certPath := ctx.GlobalString("cert")
	if certPath != "" {
		client, err = getHttpsClient(certPath)
		if err != nil {
			return nil, err
		}
	} else {
		client = http.DefaultClient
	}
	buffer := bytes.NewBuffer(data)

	/* #nosec */
	resp, err := client.Post(url, "application/json", buffer)
	if err != nil {
		return nil, err
	}
	c, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = resp.Body.Close()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func getHttpsClient(certPath string) (*http.Client, error) {
	caCert, err := ioutil.ReadFile(certPath)
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}, nil
}

func getURL(ctx *cli.Context, p string) (string, error) {
	api := ctx.GlobalString("gateway")
	api = strings.TrimSpace(api)
	if api[len(api)-1:] != "/" {
		api = api + "/"
	}

	return api + p, nil
}
