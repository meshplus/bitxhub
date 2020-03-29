package client

import (
	"encoding/base64"
	"fmt"

	"github.com/tidwall/gjson"
)

func parseResponse(data []byte) (string, error) {
	res := gjson.Get(string(data), "data")

	ret, err := base64.StdEncoding.DecodeString(res.String())
	if err != nil {
		return "", fmt.Errorf("wrong data: %w", err)
	}

	return string(ret), nil
}
