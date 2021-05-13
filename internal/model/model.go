package model

import "encoding/json"

type MerkleWrapperSign struct {
	Address   string `json:"address"`
	Signature []byte `json:"signature"`
}

func (m *MerkleWrapperSign) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

func (m *MerkleWrapperSign) Unmarshal(data []byte) error {
	return json.Unmarshal(data, m)
}

type CertsMessage struct {
	AgencyCert []byte
	NodeCert   []byte
}

func (c *CertsMessage) Marshal() ([]byte, error) {
	return json.Marshal(c)
}

func (c *CertsMessage) Unmarshal(data []byte) error {
	return json.Unmarshal(data, c)
}
