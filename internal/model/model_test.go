package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMerkleWrapperSign_Unmarshal(t *testing.T) {
	m := MerkleWrapperSign{
		Address:   "0xba30d0dd7876318da451582",
		Signature: []byte("123456"),
	}

	data, err := m.Marshal()
	require.Nil(t, err)
	require.EqualValues(t,
		`{"address":"0xba30d0dd7876318da451582","signature":"MTIzNDU2"}`,
		string(data))

	s := &MerkleWrapperSign{}
	err = s.Unmarshal(data)
	require.Nil(t, err)
	require.EqualValues(t, "0xba30d0dd7876318da451582", s.Address)
}
