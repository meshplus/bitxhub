package client

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/tidwall/gjson"
)

const (
	empty = ""
	tab   = "  "
)

func parseResponse(data []byte) (string, error) {
	res := gjson.Get(string(data), "data")

	ret, err := base64.StdEncoding.DecodeString(res.String())
	if err != nil {
		return "", fmt.Errorf("wrong data: %w", err)
	}

	return string(ret), nil
}

func prettyJson(data string) (string, error) {
	var out bytes.Buffer
	err := json.Indent(&out, []byte(data), empty, tab)
	if err != nil {
		return "", fmt.Errorf("wrong data: %w", err)
	}
	return out.String(), nil
}