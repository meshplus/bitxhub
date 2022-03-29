package coreapi

import (
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/event"
	nodemgr "github.com/meshplus/bitxhub-core/node-mgr"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/meshplus/bitxhub/pkg/utils"
)

type AuditAPI CoreAPI

var _ api.AuditAPI = (*AuditAPI)(nil)

func (api *AuditAPI) SubscribeAuditEvent(ch chan<- *pb.AuditTxInfo) event.Subscription {
	return api.bxh.BlockExecutor.SubscribeAuditEvent(ch)
}

func (api AuditAPI) HandleAuditNodeSubscription(dataCh chan<- *pb.AuditTxInfo, auditNodeID string, blockStart uint64) error {
	// 1. get the auditable appchain id
	chainIDMap, auditNodeIDMap, err := api.getPermitChains(auditNodeID)
	if err != nil {
		return fmt.Errorf("get permit chains error: %w", err)
	}

	// 2. subscribe to real-time audit info
	auditTxInfoCh := make(chan *pb.AuditTxInfo, 1024)
	sub := api.SubscribeAuditEvent(auditTxInfoCh)
	defer sub.Unsubscribe()

	// 3. send historical audit info from the current block height
	blockCur := api.bxh.Ledger.GetChainMeta().Height
	for height := blockStart; height <= blockCur; height++ {
		block, err := api.bxh.Ledger.GetBlock(height)
		if err != nil {
			return fmt.Errorf("get block error: %v", block)
		}

		if utils.TestAuditPermitBloom(api.logger, block.BlockHeader.Bloom, chainIDMap, auditNodeIDMap) {
			for _, tx := range block.Transactions.Transactions {
				// support bxhTx
				switch tx.(type) {
				case *pb.BxhTransaction:
					// 3.1 get receipt
					receipt, err := api.bxh.Ledger.GetReceipt(tx.GetHash())
					if err != nil {
						break
					}

					isPermited := false
					if utils.TestAuditPermitBloom(api.logger, receipt.Bloom, chainIDMap, auditNodeIDMap) {
						// 3.2 get event from receipt and check event info permission
						for _, ev := range receipt.Events {
							if isPermited = hasPermitedInfo(ev, auditNodeIDMap, chainIDMap); isPermited {
								break
							}
						}

						// 3.3 send info
						if isPermited {
							auditTxInfo := &pb.AuditTxInfo{
								Tx:          tx.(*pb.BxhTransaction),
								Rec:         receipt,
								BlockHeight: height,
							}
							dataCh <- auditTxInfo
						}
					}
				}

			}
		}
	}

	// 4. send real-time audit info
	for auditTxinfo := range auditTxInfoCh {
		isPermited := false
		for nodeID, _ := range auditNodeIDMap {
			if _, ok := auditTxinfo.RelatedNodeIDList[nodeID]; ok {
				isPermited = true
				break
			}
		}

		for chainID, _ := range chainIDMap {
			if _, ok := auditTxinfo.RelatedChainIDList[chainID]; ok {
				isPermited = true
				break
			}
		}

		if isPermited {
			dataCh <- auditTxinfo
		}
	}

	return nil
}

func (api AuditAPI) getPermitChains(auditNodeID string) (map[string]struct{}, map[string]struct{}, error) {
	ok, nodeData := api.bxh.Ledger.GetState(constant.NodeManagerContractAddr.Address(), []byte(nodemgr.NodeKey(auditNodeID)))
	if !ok {
		return make(map[string]struct{}), make(map[string]struct{}), nil
	}

	node := &nodemgr.Node{}
	if err := json.Unmarshal(nodeData, node); err != nil {
		return make(map[string]struct{}), make(map[string]struct{}), fmt.Errorf("json unmarshal node error: %w", err)
	}
	if node.IsAvailable() {
		return node.Permissions, map[string]struct{}{auditNodeID: {}}, nil
	}

	return make(map[string]struct{}), make(map[string]struct{}), nil
}

func hasPermitedInfo(event *pb.Event, permitNodes, permitChains map[string]struct{}) bool {
	if !event.IsAuditEvent() {
		return false
	}

	auditRelatedObjInfo := pb.AuditRelatedObjInfo{}
	if err := json.Unmarshal(event.Data, &auditRelatedObjInfo); err != nil {
		return false
	}

	for auditNodeID, _ := range permitNodes {
		if _, ok := auditRelatedObjInfo.RelatedNodeIDList[auditNodeID]; ok {
			return true
		}
	}

	for chainID, _ := range permitChains {
		if _, ok := auditRelatedObjInfo.RelatedChainIDList[chainID]; ok {
			return true
		}
	}

	return false
}
