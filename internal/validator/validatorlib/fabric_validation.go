package validatorlib

import (
	"encoding/json"
	"fmt"

	"github.com/gogo/protobuf/proto"
	mb "github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/protoutil"
)

type valiadationArtifacts struct {
	rwset        []byte
	prp          []byte
	endorsements []*peer.Endorsement
	cap          *peer.ChaincodeActionPayload
}

type validatorInfo struct {
	ConfByte []string `json:"conf_byte"`
	Policy   string   `json:"policy"`
	Cid      string   `json:"cid"`
}

func GetPolicyEnvelope(policy string) ([]byte, error) {
	policyEnv, err := cauthdsl.FromString(policy)
	if err != nil {
		return nil, err
	}
	policyBytes, err := proto.Marshal(policyEnv)
	if err != nil {
		return nil, err
	}
	return policyBytes, nil
}

func UnmarshalValidatorInfo(validatorBytes []byte) (*validatorInfo, error) {
	vInfo := &validatorInfo{}
	if err := json.Unmarshal(validatorBytes, vInfo); err != nil {
		return nil, err
	}
	return vInfo, nil
}

func extractValidationArtifacts(proof []byte) (*valiadationArtifacts, error) {
	cap, err := protoutil.UnmarshalChaincodeActionPayload(proof)
	if err != nil {
		return nil, err
	}

	pRespPayload, err := protoutil.UnmarshalProposalResponsePayload(cap.Action.ProposalResponsePayload)
	if err != nil {
		err = fmt.Errorf("GetProposalResponsePayload error %s", err)
		return nil, err
	}
	if pRespPayload.Extension == nil {
		err = fmt.Errorf("nil pRespPayload.Extension")
		return nil, err
	}
	respPayload, err := protoutil.UnmarshalChaincodeAction(pRespPayload.Extension)
	if err != nil {
		err = fmt.Errorf("GetChaincodeAction error %s", err)
		return nil, err
	}

	return &valiadationArtifacts{
		rwset:        respPayload.Results,
		prp:          cap.Action.ProposalResponsePayload,
		endorsements: cap.Action.Endorsements,
		cap:          cap,
	}, nil
}

func ValidateV14(proof, policyBytes []byte, confByte []string, cid string) error {
	// Get the validation artifacts that help validate the chaincodeID and policy
	artifact, err := extractValidationArtifacts(proof)
	if err != nil {
		return err
	}

	err = ValidateChainCodeID(artifact.prp, cid)
	if err != nil {
		return err
	}
	signatureSet := GetSignatureSet(artifact)
	pe, err := NewPolicyEvaluator(confByte)
	if err != nil {
		return err
	}

	return pe.Evaluate(policyBytes, signatureSet)
}

func ValidateChainCodeID(prp []byte, name string) error {
	payload := &peer.ProposalResponsePayload{}
	if err := proto.Unmarshal(prp, payload); err != nil {
		return err
	}
	chaincodeAct := &peer.ChaincodeAction{}
	if err := proto.Unmarshal(payload.Extension, chaincodeAct); err != nil {
		return err
	}
	if name != chaincodeAct.ChaincodeId.Name {
		return fmt.Errorf("chaincode id does not match")
	}

	return nil
}

type PolicyEvaluator struct {
	msp.IdentityDeserializer
}

func NewPolicyEvaluator(confBytes []string) (*PolicyEvaluator, error) {
	mspList := make([]msp.MSP, len(confBytes))
	for i, confByte := range confBytes {
		tempBccsp, err := msp.New(
			&msp.BCCSPNewOpts{NewBaseOpts: msp.NewBaseOpts{Version: msp.MSPv1_3}},
			factory.GetDefault(),
		)
		if err != nil {
			return nil, err
		}
		conf := &mb.MSPConfig{}
		if err := proto.UnmarshalText(confByte, conf); err != nil {
			return nil, err
		}
		err = tempBccsp.Setup(conf)
		if err != nil {
			return nil, err
		}
		mspList[i] = tempBccsp
	}

	manager := msp.NewMSPManager()
	err := manager.Setup(mspList)
	if err != nil {
		return nil, err
	}
	deserializer := &dynamicDeserializer{mspm: manager}
	pe := &PolicyEvaluator{IdentityDeserializer: deserializer}

	return pe, nil
}

func (id *PolicyEvaluator) Evaluate(policyBytes []byte, signatureSet []*protoutil.SignedData) error {
	pp := cauthdsl.NewPolicyProvider(id.IdentityDeserializer)
	policy, _, err := pp.NewPolicy(policyBytes)
	if err != nil {
		return err
	}
	return policy.EvaluateSignedData(signatureSet)
}

func GetSignatureSet(artifact *valiadationArtifacts) []*protoutil.SignedData {
	signatureSet := []*protoutil.SignedData{}
	for _, endorsement := range artifact.endorsements {
		data := make([]byte, len(artifact.prp)+len(endorsement.Endorser))
		copy(data, artifact.prp)
		copy(data[len(artifact.prp):], endorsement.Endorser)

		signatureSet = append(signatureSet, &protoutil.SignedData{
			// set the data that is signed; concatenation of proposal response bytes and endorser ID
			Data: data,
			// set the identity that signs the message: it's the endorser
			Identity: endorsement.Endorser,
			// set the signature
			Signature: endorsement.Signature})
	}
	return signatureSet
}

type dynamicDeserializer struct {
	mspm msp.MSPManager
}

func (ds *dynamicDeserializer) DeserializeIdentity(serializedIdentity []byte) (msp.Identity, error) {
	return ds.mspm.DeserializeIdentity(serializedIdentity)
}

func (ds *dynamicDeserializer) IsWellFormed(identity *mb.SerializedIdentity) error {
	return ds.mspm.IsWellFormed(identity)
}
