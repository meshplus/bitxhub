package contracts

import (
	"fmt"
	"github.com/meshplus/bitxhub-core/boltvm"
)

type Store struct {
	boltvm.Stub
}

func (s *Store) Set(key string, value string) *boltvm.Response {
	s.SetObject(key, value)

	return boltvm.Success(nil)
}

func (s *Store) Get(key string) *boltvm.Response {
	var v string
	ok := s.GetObject(key, &v)
	if !ok {
		return boltvm.Error(boltvm.OtherInternalErrCode, fmt.Sprintf(string(boltvm.OtherInternalErrMsg), "there is not exist key"))
	}

	return boltvm.Success([]byte(v))
}
