package client

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/urfave/cli"
)

func httpGet(url string) ([]byte, error) {
	/* #nosec */
	resp, err := http.Get(url)
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

func httpPost(url string, data []byte) ([]byte, error) {
	buffer := bytes.NewBuffer(data)

	/* #nosec */
	resp, err := http.Post(url, "application/json", buffer)
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

func getURL(ctx *cli.Context, path string) (string, error) {
	repoRoot, err := repo.PathRootWithDefault(ctx.GlobalString("repo"))
	if err != nil {
		return "", err
	}

	api, err := repo.GetAPI(repoRoot)
	if err != nil {
		return "", fmt.Errorf("get api file: %w", err)
	}

	api = strings.TrimSpace(api)

	return api + path, nil
}
