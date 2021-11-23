package coreapi

import (
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/event"
	nodemgr "github.com/meshplus/bitxhub-core/node-mgr"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
)

type AuditAPI CoreAPI

var _ api.AuditAPI = (*AuditAPI)(nil)

func (api *AuditAPI) SubscribeAuditEvent(ch chan<- *pb.AuditTxInfo) event.Subscription {
	return api.bxh.BlockExecutor.SubscribeAuditEvent(ch)
}

func (api AuditAPI) HandleAuditNodeSubscription(dataCh chan<- *pb.AuditTxInfo, auditNodeID string, blockStart uint64) error {
	// 1. get the auditable appchain id
	chainIDMap, err := api.getPermitChains(auditNodeID)
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
		isPermited := false
		for _, tx := range block.Transactions.Transactions {
			if tx.GetType() != pb.NormalBxhTx {
				continue
			}

			// 3.1 get receipt
			receipt, err := api.bxh.Ledger.GetReceipt(tx.GetHash())
			if err != nil {
				break
			}

			// 3.2 get event from receipt and check event info permission
			for _, ev := range receipt.Events {
				if isPermited = hasPermitedInfo(ev, auditNodeID, chainIDMap); isPermited {
					break
				}
			}

			// 3.3 send info
			if isPermited {
				auditTxInfo := &pb.AuditTxInfo{
					Tx:  tx.(*pb.BxhTransaction),
					Rec: receipt,
				}
				dataCh <- auditTxInfo
			}
		}
	}

	// 4. send real-time audit info
	for auditTxinfo := range auditTxInfoCh {
		isPermited := false
		if _, ok := auditTxinfo.RelatedNodeIDList[auditNodeID]; ok {
			isPermited = true
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

func (api AuditAPI) getPermitChains(auditNodeID string) (map[string]struct{}, error) {
	ok, nodeData := api.bxh.Ledger.GetState(constant.NodeManagerContractAddr.Address(), []byte(nodemgr.NodeKey(auditNodeID)))
	if !ok {
		return make(map[string]struct{}), nil
	}

	node := &nodemgr.Node{}
	if err := json.Unmarshal(nodeData, node); err != nil {
		return nil, fmt.Errorf("json unmarshal node error: %w", err)
	}

	return node.Permissions, nil
}

func hasPermitedInfo(event *pb.Event, auditNodeID string, permitChains map[string]struct{}) bool {
	if !event.IsAuditEvent() {
		return false
	}

	auditRelatedObjInfo := pb.AuditRelatedObjInfo{}
	if err := json.Unmarshal(event.Data, &auditRelatedObjInfo); err != nil {
		return false
	}

	if _, ok := auditRelatedObjInfo.RelatedNodeIDList[auditNodeID]; ok {
		return true
	}

	for chainID, _ := range permitChains {
		if _, ok := auditRelatedObjInfo.RelatedChainIDList[chainID]; ok {
			return true
		}
	}

	return false
}
