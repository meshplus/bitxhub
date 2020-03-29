package wasmlib

import (
	"io/ioutil"
	"testing"

	"github.com/golang/protobuf/proto"
	m "github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPolicyEnvelope(t *testing.T) {
	_, err := GetPolicyEnvelope("OR(AND('A.member', 'B.member'), OR('C.admin', 'D.member'))")
	assert.NoError(t, err)
}

func TestPayloadUnmarshal(t *testing.T) {
	bytes, err := ioutil.ReadFile("../testdata/proof")
	require.Nil(t, err)
	cap, err := protoutil.UnmarshalChaincodeActionPayload(bytes)

	require.Nil(t, err)

	prp := cap.Action.ProposalResponsePayload
	signatureSet := []*protoutil.SignedData{}
	for _, endorsement := range cap.Action.Endorsements {
		data := make([]byte, len(prp)+len(endorsement.Endorser))
		copy(data, prp)
		copy(data[len(prp):], endorsement.Endorser)

		signatureSet = append(signatureSet, &protoutil.SignedData{
			// set the data that is signed; concatenation of proposal response bytes and endorser ID
			Data: data,
			// set the identity that signs the message: it's the endorser
			Identity: endorsement.Endorser,
			// set the signature
			Signature: endorsement.Signature})
	}
	sId := &m.SerializedIdentity{}
	err = proto.Unmarshal(signatureSet[0].Identity, sId)
	require.Nil(t, err)
}

func TestUnmarshalValidatorInfo(t *testing.T) {
	vBytes, err := ioutil.ReadFile("../testdata/validator")
	require.Nil(t, err)
	_, err = UnmarshalValidatorInfo(vBytes)
	require.Nil(t, err)
}

func TestValidateV14(t *testing.T) {
	vBytes, err := ioutil.ReadFile("../testdata/validator")
	require.Nil(t, err)
	proof, err := ioutil.ReadFile("../testdata/proof")
	require.Nil(t, err)
	v, err := UnmarshalValidatorInfo(vBytes)
	require.Nil(t, err)
	err = ValidateV14(proof, []byte(v.Policy), v.ConfByte, v.Cid)
	require.Nil(t, err)
}
