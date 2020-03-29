package tester

import (
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/tidwall/gjson"
)

const (
	host = "http://localhost:9091/v1/"
)

type Gateway struct {
	suite.Suite
}

func (suite *Gateway) TestGetBlock() {
	data := httpGet(suite.Suite, "block?type=0&value=1")
	height := gjson.Get(data, "block_header.number").Int()
	suite.Assert().EqualValues(1, height, data)
}

func (suite *Gateway) TestGetBlockByHash() {
	data := httpGet(suite.Suite, "block?type=0&value=1")
	hash := gjson.Get(data, "block_hash").String()
	m := httpGet(suite.Suite, "block?type=1&value="+hash)
	suite.Assert().EqualValues(1, gjson.Get(m, "block_header.number").Int(), m)
	suite.Assert().Equal(hash, gjson.Get(m, "block_hash").String(), m)
}

func TestGateway(t *testing.T) {
	suite.Run(t, &Gateway{})
}

func httpGet(suite suite.Suite, path string) string {
	resp, err := http.Get(host + path)
	suite.Assert().Nil(err)
	c, err := ioutil.ReadAll(resp.Body)
	suite.Assert().Nil(err)
	err = resp.Body.Close()
	suite.Assert().Nil(err)

	return string(c)
}
