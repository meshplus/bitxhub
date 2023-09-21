package adaptor

func (a *RBFTAdaptor) Sign(msg []byte) ([]byte, error) {
	return []byte{}, nil
	// h := sha256.Sum256(msg)
	// return a.priv.Sign(h[:])
}

func (a *RBFTAdaptor) Verify(peerID string, signature []byte, msg []byte) error {
	// h := sha256.Sum256(msg)
	// id, ok := a.nodePIDToID[peerID]
	// if !ok {
	// 	return fmt.Errorf("not found peerID mapping: %v", peerID)
	// }
	// addr := types.NewAddressByStr(a.Nodes[id].Account)
	// ret, err := asym.Verify(crypto.Secp256k1, signature, h[:], *addr)
	// if err != nil {
	// 	return err
	// }

	// if !ret {
	// 	return fmt.Errorf("verify error")
	// }

	return nil
}
