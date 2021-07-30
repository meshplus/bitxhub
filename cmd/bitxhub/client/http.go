package client

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
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

	req, err := http.NewRequest("GET", url, nil)
	accessPath := ctx.GlobalString("access")
	if accessPath != "" {
		caCertData, err := ioutil.ReadFile(accessPath)
		if err != nil {
			return nil, err
		}
		caCertDataString := base64.StdEncoding.EncodeToString(caCertData)
		req.Header.Add("grpc-metadata-access", caCertDataString)
	}
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)

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
	req, err := http.NewRequest("POST", url, buffer)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	accessPath := ctx.GlobalString("access")
	if accessPath != "" {
		caCertData, err := ioutil.ReadFile(accessPath)
		if err != nil {
			return nil, err
		}
		caCertDataString := base64.StdEncoding.EncodeToString(caCertData)
		req.Header.Add("grpc-metadata-access", caCertDataString)
	}
	resp, err := client.Do(req)

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
