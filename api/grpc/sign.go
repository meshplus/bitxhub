package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/meshplus/bitxhub-core/tss"
	"github.com/meshplus/bitxhub-core/tss/conversion"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/model"
	"github.com/meshplus/bitxhub/pkg/utils"
	"github.com/sirupsen/logrus"
)

func (cbs *ChainBrokerService) GetMultiSigns(ctx context.Context, req *pb.GetSignsRequest) (*pb.SignResponse, error) {
	var (
		wg     = sync.WaitGroup{}
		result = make(map[string][]byte)
	)

	wg.Add(1)
	go func(result map[string][]byte) {
		for k, v := range cbs.api.Broker().FetchSignsFromOtherPeers(req) {
			result[k] = v
		}
		wg.Done()
	}(result)

	address, sign, _, err := cbs.api.Broker().GetSign(req)
	wg.Wait()

	if err != nil {
		cbs.logger.WithFields(logrus.Fields{
			"id":  req.Content,
			"err": err.Error(),
		}).Errorf("Get sign on current node")
		return nil, fmt.Errorf("get sign on current node failed: %w", err)
	} else {
		result[address] = sign
	}
	return &pb.SignResponse{
		Sign: result,
	}, nil
}

func (cbs *ChainBrokerService) GetTssSigns(ctx context.Context, req *pb.GetSignsRequest) (*pb.SignResponse, error) {
	var (
		wg     = sync.WaitGroup{}
		result = [][]byte{}
	)

	// 1. check req type
	if !isTssReq(req) {
		return nil, fmt.Errorf("req type is not tss req")
	}

	// 2. make a tss req with threshold signers
	tssReq := &pb.GetSignsRequest{
		Type:    req.Type,
		Content: req.Content,
		Extra:   req.Extra,
	}
	signersALL := strings.Split(strings.Replace(string(tssReq.Extra), " ", "", -1), ",")

	for {

		// 3. check signers num
		if len(signersALL) <= 0 {
			cbs.logger.WithFields(logrus.Fields{
				"signers": signersALL,
			}).Errorf("too less signers")
			break
		}

		// 4. choose signers randomly
		nums := RandRangeNumbers(0, len(signersALL)-1, int(cbs.api.Broker().GetQuorum()))
		tssSigners := []string{}
		for _, i := range nums {
			tssSigners = append(tssSigners, signersALL[i])
		}
		tssReq.Extra = []byte(strings.Join(tssSigners, ","))
		cbs.logger.Infof("====================== tss signers: %s", string(tssReq.Extra))

		// 5. send sign req to others
		wg.Add(1)
		go func() {
			cbs.api.Broker().FetchSignsFromOtherPeers(tssReq)
			tssSignResCh := make(chan *pb.Message)
			tssSignResSub := cbs.api.Feed().SubscribeTssSignRes(tssSignResCh)
			defer tssSignResSub.Unsubscribe()
			select {
			case m := <-tssSignResCh:
				signRes := &model.MerkleWrapperSign{}
				if err := signRes.Unmarshal(m.Data); err != nil {
					cbs.logger.WithFields(logrus.Fields{
						"err": err,
					}).Errorf("unmarshal sign res error")
					break
				}

				_, pub, err := cbs.api.Broker().GetTssPubkey()
				if err != nil {
					cbs.logger.WithFields(logrus.Fields{
						"err": err,
					}).Errorf("Get tss pubkey error")
					break
				}

				if err := utils.VerifyTssSigns(signRes.Signature, pub, cbs.logger); err != nil {
					cbs.logger.WithFields(logrus.Fields{}).Errorf("Verify tss signs error")
					break
				} else {
					result = append(result, signRes.Signature)
					cbs.logger.WithFields(logrus.Fields{}).Debug("get verified tss signature from other peers")
					wg.Done()
					return
				}
			case <-time.After(cbs.config.Tss.KeySignTimeout):
				cbs.logger.WithFields(logrus.Fields{}).Warnf("wait for sign from other peers timeout: %v", cbs.config.Tss.KeySignTimeout)
				wg.Done()
				return
			}
		}()

		// 6. get sign by ourself
		addr, sign, culprits, err := cbs.api.Broker().GetSign(tssReq)
		wg.Wait()
		if err == nil {
			return &pb.SignResponse{
				Sign: map[string][]byte{
					addr: convertSignData(sign),
				},
			}, nil
		} else if errors.Is(err, tss.ErrNotActiveSigner) {
			if len(result) != 0 {
				return &pb.SignResponse{
					Sign: map[string][]byte{
						addr: convertSignData(result[0]),
					},
				}, nil
			} else {
				return nil, fmt.Errorf("get tss signs error")
			}
		}

		// 7. handle culprits
		cbs.logger.WithFields(logrus.Fields{
			"id":       req.Content,
			"culprits": culprits,
			"err":      err.Error(),
		}).Errorf("Get tss sign on current node")

		if culprits == nil {
			return nil, err
		}
		for _, idC := range culprits {
			for i, idS := range signersALL {
				if idC == idS {
					if i == 0 {
						signersALL = signersALL[1:]
					} else {
						signersALL = append(signersALL[:i-1], signersALL[i+1:]...)
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("GetTSSSigns error")

}

func convertSignData(signData []byte) []byte {
	signs := []conversion.Signature{}
	_ = json.Unmarshal(signData, &signs)

	signs1 := [][]byte{}
	for _, sign := range signs {
		signs1 = append(signs1, sign.SignEthData)
	}

	signs1Data, _ := json.Marshal(signs1)

	return signs1Data
}

func isTssReq(req *pb.GetSignsRequest) bool {
	switch req.Type {
	case pb.GetSignsRequest_TSS_IBTP_REQUEST, pb.GetSignsRequest_TSS_IBTP_RESPONSE:
		return true
	default:
		return false
	}
}

func RandRangeNumbers(min, max, count int) []int {
	//检查参数
	if max < min || (max-min+1) < count {
		return nil
	}
	nums := make([]int, max-min+1)
	position := -1            //记录nums[-min]的位置
	if min <= 0 && max >= 0 { //获取范围内有0，则先用max+1代替
		position = -min
		nums[position] = max + 1
	}
	//随机数生成器，加入时间戳保证每次生成的随机数不一样
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < count; i++ {
		num := r.Intn(max - min + 1)
		if nums[i] == 0 { //代表未赋值
			nums[i] = min + i
		}
		if nums[num] == 0 { //代表未赋值
			nums[num] = min + num
		}
		if position != -1 { //此时需要记录位置
			if i == position {
				position = num
			} else if num == position {
				position = i
			}
		}
		nums[i], nums[num] = nums[num], nums[i]
	}

	if position != -1 { //证明有位置记录，则还原为0
		nums[position] = 0
	}
	return nums[0:count]
}
