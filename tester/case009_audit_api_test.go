package tester

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/governance"
	node_mgr "github.com/meshplus/bitxhub-core/node-mgr"
	"github.com/meshplus/bitxhub-core/validator"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/stretchr/testify/suite"
)

type AuditAPI struct {
	suite.Suite
	api api.CoreAPI

	priAdmin1   crypto.PrivateKey
	priAdmin2   crypto.PrivateKey
	priAdmin3   crypto.PrivateKey
	adminNonce1 uint64
	adminNonce2 uint64
	adminNonce3 uint64

	k1      crypto.PrivateKey
	k2      crypto.PrivateKey
	kNonce1 uint64
	kNonce2 uint64
	addr1   string
	addr2   string

	chainID1 string
	chainID2 string

	nodeKey     crypto.PrivateKey
	nodeAccount string
}

func (suite *AuditAPI) SetupSuite() {
	path1 := "./test_data/config/node1/key.json"
	path2 := "./test_data/config/node2/key.json"
	path3 := "./test_data/config/node3/key.json"
	keyPath1 := filepath.Join(path1)
	keyPath2 := filepath.Join(path2)
	keyPath3 := filepath.Join(path3)
	priAdmin1, err := asym.RestorePrivateKey(keyPath1, "bitxhub")
	suite.Require().Nil(err)
	priAdmin2, err := asym.RestorePrivateKey(keyPath2, "bitxhub")
	suite.Require().Nil(err)
	priAdmin3, err := asym.RestorePrivateKey(keyPath3, "bitxhub")
	suite.Require().Nil(err)
	suite.priAdmin1 = priAdmin1
	suite.priAdmin2 = priAdmin2
	suite.priAdmin3 = priAdmin3

	fromAdmin1, err := priAdmin1.PublicKey().Address()
	suite.Require().Nil(err)
	fromAdmin2, err := priAdmin2.PublicKey().Address()
	suite.Require().Nil(err)
	fromAdmin3, err := priAdmin3.PublicKey().Address()
	suite.Require().Nil(err)

	suite.adminNonce1 = suite.api.Broker().GetPendingNonceByAccount(fromAdmin1.String())
	suite.adminNonce2 = suite.api.Broker().GetPendingNonceByAccount(fromAdmin2.String())
	suite.adminNonce3 = suite.api.Broker().GetPendingNonceByAccount(fromAdmin3.String())

	suite.k1, err = asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	suite.k2, err = asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	addr1, err := suite.k1.PublicKey().Address()
	suite.Require().Nil(err)
	suite.addr1 = addr1.String()
	addr2, err := suite.k2.PublicKey().Address()
	suite.Require().Nil(err)
	suite.addr2 = addr2.String()
	suite.Require().Nil(transfer(suite.Suite, suite.api, addr1, 10000000000000))
	suite.Require().Nil(transfer(suite.Suite, suite.api, addr2, 10000000000000))
	suite.kNonce1 = suite.api.Broker().GetPendingNonceByAccount(addr1.String())
	suite.kNonce2 = suite.api.Broker().GetPendingNonceByAccount(addr2.String())

	suite.chainID1 = "appchain1case009"
	suite.chainID2 = "appchain2case009"

	path4 := "./test_data/key.json"
	keyPath4 := filepath.Join(path4)
	nodeKey, err := asym.RestorePrivateKey(keyPath4, "bitxhub")
	suite.Require().Nil(err)
	suite.nodeKey = nodeKey
	fromNode1, err := nodeKey.PublicKey().Address()
	suite.Require().Nil(err)
	suite.nodeAccount = fromNode1.String()

}

func (suite *AuditAPI) TestHandleAuditNodeSubscription() {
	suite.registerAppchain(suite.k1, suite.kNonce1, suite.addr1, suite.chainID1, "应用链1case009", validator.HappyRuleAddr, appchainMgr.ChainTypeETH)
	suite.registerAppchain(suite.k2, suite.kNonce2, suite.addr2, suite.chainID2, "应用链2case009", validator.HappyRuleAddr, appchainMgr.ChainTypeFlato1_0_6)
	suite.registerAuditNode(suite.nodeAccount, "审计节点1", suite.chainID2)

	fmt.Printf("======================================================================================\n\n\n")
	ch := make(chan *pb.AuditTxInfo)
	go func() {
		err := suite.api.Audit().HandleAuditNodeSubscription(ch, suite.nodeAccount, 1)
		suite.Require().Nil(err)
	}()

	for auditTxInfo := range ch {
		if auditTxInfo.Tx.IsIBTP() {
			fmt.Printf("================ ibtp info: \n "+
				"from: %s\n"+
				"to: %s\n"+
				"payload: %v\n",
				auditTxInfo.Tx.GetIBTP().From,
				auditTxInfo.Tx.GetIBTP().To,
				auditTxInfo.Tx.GetIBTP().Payload,
			)
		} else {
			data := &pb.TransactionData{}
			err := data.Unmarshal(auditTxInfo.Tx.GetPayload())
			suite.Require().Nil(err)
			suite.Require().Equal(pb.TransactionData_INVOKE, data.Type)
			suite.Require().Equal(pb.TransactionData_BVM, data.VmType)

			payload := &pb.InvokePayload{}
			err = payload.Unmarshal(data.Payload)
			suite.Require().Nil(err)
			fmt.Printf("================ invoke info: \n"+
				"from: %s\n"+
				"to: %s\n"+
				"method: %s\n"+
				"args: %v\n",
				auditTxInfo.Tx.From.String(),
				auditTxInfo.Tx.To.String(),
				payload.Method,
				payload.Args,
			)
		}

		flag := false
		chains := map[string]struct{}{}
		nodes := map[string]struct{}{}
		for _, ev := range auditTxInfo.Rec.Events {
			if ev.IsAuditEvent() {
				eventInfo := &pb.AuditRelatedObjInfo{}
				err := json.Unmarshal(ev.Data, eventInfo)
				suite.Require().Nil(err)
				_, relateChain := eventInfo.RelatedChainIDList[suite.chainID2]
				_, relateNode := eventInfo.RelatedNodeIDList[suite.nodeAccount]
				if relateChain {
					for k, _ := range eventInfo.RelatedChainIDList {
						chains[k] = struct{}{}
					}
					flag = true
				}
				if relateNode {
					for k, _ := range eventInfo.RelatedNodeIDList {
						nodes[k] = struct{}{}
					}
					flag = true
				}
			}
		}
		fmt.Printf("================  audit rec info: \n "+
			"ret: %s\n"+
			"chains: %v\n"+
			"nodes: %v\n"+
			"\n\n\n",
			string(auditTxInfo.Rec.Ret),
			chains,
			nodes,
		)
		suite.Require().True(flag)
	}
}

func (suite *AuditAPI) registerAppchain(k1 crypto.PrivateKey, kNonce1 uint64, addr1, chainId, chainName, ruleAddr, chainType string) {
	args := []*pb.Arg{
		pb.String(chainId),
		pb.String(chainName),
		pb.String(chainType),
		pb.Bytes(nil),
		pb.String("broker"),
		pb.String("desc"),
		pb.String(ruleAddr),
		pb.String("url"),
		pb.String(addr1),
		pb.String("reason"),
	}
	ret, err := invokeBVMContract(suite.api, k1, kNonce1, constant.AppchainMgrContractAddr.Address(), "RegisterAppchain", args...)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	kNonce1++

	gRet := &governance.GovernanceResult{}
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	proposalId := gRet.ProposalID
	fmt.Printf("======== RegisterAppchain proposal id: %s\n", proposalId)

	suite.vote(proposalId, suite.priAdmin1, suite.adminNonce1, string(contracts.APPROVED))
	suite.adminNonce1++

	suite.vote(proposalId, suite.priAdmin2, suite.adminNonce2, string(contracts.APPROVED))
	suite.adminNonce2++

	suite.vote(proposalId, suite.priAdmin3, suite.adminNonce3, string(contracts.APPROVED))
	suite.adminNonce3++

	ret, err = invokeBVMContract(suite.api, k1, kNonce1, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(chainId))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	suite.kNonce1++
	chainInfo := &appchainMgr.Appchain{}
	err = json.Unmarshal(ret.Ret, chainInfo)
	suite.Require().Nil(err)
	suite.Require().Equal("desc", chainInfo.Desc)
	suite.Require().Equal(governance.GovernanceAvailable, chainInfo.Status)
	fmt.Sprintf("======== chain info: %v\n", chainInfo)
}

// nodeAccount, nodeType, nodePid string, nodeVpId uint64, nodeName, permitStr, reason string
func (suite *AuditAPI) registerAuditNode(nodeAccount, nodeName, permitStr string) {
	args := []*pb.Arg{
		pb.String(nodeAccount),
		pb.String(string(node_mgr.NVPNode)),
		pb.String(""),
		pb.Uint64(0),
		pb.String(nodeName),
		pb.String(permitStr),
		pb.String("reason"),
	}
	ret, err := invokeBVMContract(suite.api, suite.priAdmin1, suite.adminNonce1, constant.NodeManagerContractAddr.Address(), "RegisterNode", args...)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	suite.adminNonce1++

	gRet := &governance.GovernanceResult{}
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	proposalId := gRet.ProposalID
	fmt.Printf("======== RegisterNode proposal id: %s\n", proposalId)

	suite.vote(proposalId, suite.priAdmin1, suite.adminNonce1, string(contracts.APPROVED))
	suite.adminNonce1++

	suite.vote(proposalId, suite.priAdmin2, suite.adminNonce2, string(contracts.APPROVED))
	suite.adminNonce2++

	suite.vote(proposalId, suite.priAdmin3, suite.adminNonce3, string(contracts.APPROVED))
	suite.adminNonce3++

	ret, err = invokeBVMContract(suite.api, suite.priAdmin1, suite.adminNonce1, constant.NodeManagerContractAddr.Address(), "GetNode", pb.String(nodeAccount))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	suite.adminNonce1++
	nodeInfo := &node_mgr.Node{}
	err = json.Unmarshal(ret.Ret, nodeInfo)
	suite.Require().Nil(err)
	suite.Require().Equal(governance.GovernanceAvailable, nodeInfo.Status)
	fmt.Sprintf("======== node info: %v\n", nodeInfo)
}

func (suite *AuditAPI) vote(proposalId string, adminKey crypto.PrivateKey, adminNonce uint64, info string) {
	ret, err := invokeBVMContract(suite.api, adminKey, adminNonce, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(proposalId),
		pb.String(info),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
}
