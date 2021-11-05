package client

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
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
	certPath := ctx.GlobalString("ca")
	if certPath != "" {
		client, err = getHttpsClient(ctx, certPath)
		if err != nil {
			return nil, fmt.Errorf("get httpsClient failed: %w", err)
		}
	} else {
		client = http.DefaultClient
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("get response from http GET request failed: %w", err)
	}

	c, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read reponse body error: %w", err)
	}

	err = resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("close reponse body failed: %w", err)
	}

	return c, nil
}

func httpPost(ctx *cli.Context, url string, data []byte) ([]byte, error) {
	var (
		client *http.Client
		err    error
	)
	certPath := ctx.GlobalString("ca")
	if certPath != "" {
		client, err = getHttpsClient(ctx, certPath)
		if err != nil {
			return nil, fmt.Errorf("get httpsClient failed: %w", err)
		}
	} else {
		client = http.DefaultClient
	}
	buffer := bytes.NewBuffer(data)

	/* #nosec */
	resp, err := client.Post(url, "application/json", buffer)
	if err != nil {
		return nil, fmt.Errorf("get reponse from http POST request failed: %w", err)
	}
	c, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body error: %w", err)
	}

	err = resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("close reponse body failed: %w", err)
	}

	return c, nil
}

func getHttpsClient(ctx *cli.Context, certPath string) (*http.Client, error) {
	caCert, err := ioutil.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("read ca cert error: %w", err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	cert, err := tls.LoadX509KeyPair(ctx.GlobalString("cert"), ctx.GlobalString("key"))
	if err != nil {
		return nil, fmt.Errorf("load tls key failed: %w", err)
	}
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				ServerName:   "BitXHub",
				RootCAs:      caCertPool,
				Certificates: []tls.Certificate{cert},
			},
		},
	}, nil
}

func getURL(ctx *cli.Context, p string) string {
	api := ctx.GlobalString("gateway")
	api = strings.TrimSpace(api)
	if api[len(api)-1:] != "/" {
		api = api + "/"
	}

	return api + p
}
