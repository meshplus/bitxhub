package client

import (
	"fmt"

	"github.com/urfave/cli"
)

func validatorsCMD() cli.Command {
	return cli.Command{
		Name:   "validators",
		Usage:  "Query validator address",
		Action: getValidators,
	}
}

func getValidators(ctx *cli.Context) error {
	url, err := getURL(ctx, "info?type=2")
	if err != nil {
		return err
	}

	data, err := httpGet(url)
	if err != nil {
		return fmt.Errorf("http get: %w", err)
	}

	ret, err := parseResponse(data)
	if err != nil {
		return err
	}

	retJson, err := prettyJson(ret)
	if err != nil {
		return fmt.Errorf("wrong response: %w", err)
	}

	fmt.Println(retJson)

	return nil
}
