package grpc

//func (cbs *ChainBrokerService) DelVPNode(ctx context.Context, req *pb.DelVPNodeRequest) (*pb.Response, error) {
//	delPid := req.Pid
//	peersBytes,_ := cbs.api.Network().PeerInfo()
//	peers := make(map[uint64]*pb.VpInfo)
//	err := json.Unmarshal(peersBytes,&peers)
//	if err != nil {
//		return nil, err
//	}
//	var isExist bool
//	var delID uint64
//	for id, peer := range peers {
//		if peer.Pid == delPid {
//			isExist = true
//			delID = id
//			break
//		}
//	}
//	// If self isn't vp, rejects the rpc request and returns error.
//	if !isExist {
//		return nil, fmt.Errorf("can't find pid %s from consentor or pid illegal",delPid)
//	}
//
//	if err := cbs.api.Broker().OrderReady(); err != nil {
//		return nil, err
//	}
//
//	// if there're only 4 vp nodes, we don't support delete request, return error;
//	if len(peers) == 4 {
//		return nil, errors.New("can't delete node as there're only 4 vp nodes")
//	}
//
//	// TODO (YH): don't support delete primary node
//	if err := cbs.api.Broker().DelVPNode(delID); err != nil {
//		return nil, err
//	}
//	return &pb.Response {
//		Data: nil,
//	},nil
//}
