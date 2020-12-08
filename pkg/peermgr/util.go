package peermgr

import (
	"encoding/json"
)

// VPInfo defines a rbft vp node.
type VPInfo struct {
	IPAddr     string
	KeyAddr    string
}

func MasherVPInfo(vpInfo *VPInfo) []byte {
	vpInfoBytes, _ := json.Marshal(vpInfo)
	return vpInfoBytes
}

func UnMasherVPInfo(vpInfoBytes []byte) (*VPInfo, error) {
	vpInfo := &VPInfo{}
	if err := json.Unmarshal(vpInfoBytes, vpInfo); err != nil {
		return nil, err
	}
	return vpInfo, nil
}



