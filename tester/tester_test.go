package tester

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/stretchr/testify/suite"
	"github.com/tidwall/gjson"
)

func TestTester(t *testing.T) {
	err := retry.Retry(func(attempt uint) error {
		resp, err := http.Get(host + "chain_status")
		if err != nil {
			fmt.Println(err)
			return err
		}

		c, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
			return err
		}

		if err := resp.Body.Close(); err != nil {
			return err
		}

		res := gjson.Get(string(c), "data")

		ret, err := base64.StdEncoding.DecodeString(res.String())
		if err != nil {
			fmt.Println(err)
			return err
		}

		if string(ret) != "normal" {
			fmt.Println("abnormal")
			return fmt.Errorf("abnormal")
		}

		return nil
	}, strategy.Wait(1*time.Second), strategy.Limit(60))

	if err != nil {
		log.Fatal(err)
	}

	suite.Run(t, &API{})
	suite.Run(t, &RegisterAppchain{})
	suite.Run(t, &Interchain{})
	suite.Run(t, &Role{})
	suite.Run(t, &Gateway{})
}

func grpcAddresses() []string {
	return []string{
		"localhost:60011",
		"localhost:60012",
		"localhost:60013",
		"localhost:60014",
	}
}
