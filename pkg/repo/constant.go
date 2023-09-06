package repo

const (
	AppName = "Axiom"

	// CfgFileName is the default config name
	CfgFileName = "axiom.toml"

	orderCfgFileName = "order.toml"

	// defaultRepoRoot is the path to the default config dir location.
	defaultRepoRoot = "~/.axiom"

	// rootPathEnvVar is the environment variable used to change the path root.
	rootPathEnvVar = "AXIOM_PATH"

	nodeKeyFileName = "node.key"

	pidFileName = "axiom.pid"

	LogsDirName = "logs"

	debugFileName = "axiom.debug.json"
)

const (
	OrderTypeSolo    = "solo"
	OrderTypeRbft    = "rbft"
	OrderTypeSoloDev = "solo_dev"

	KVStorageTypeLeveldb = "leveldb"
	KVStorageTypePebble  = "pebble"

	P2PSecurityTLS   = "tls"
	P2PSecurityNoise = "noise"

	PprofModeMem     = "mem"
	PprofModeCpu     = "cpu"
	PprofTypeHTTP    = "http"
	PprofTypeRuntime = "runtime"

	P2PPipeBroadcastSimple = "simple"
	P2PPipeBroadcastGossip = "gossip"
	P2PPipeBroadcastFlood  = "flood"

	ExecTypeNative = "native"
	ExecTypeDev    = "dev"
)

var (
	DefaultNodeNames = []string{
		"S2luZw==", // base64 encode King
		"UmVk",     // base64 encode Red
		"QXBwbGU=", // base64 encode Apple
		"Q2F0",     // base64 encode Cat
	}

	DefaultNodeKeys = []string{
		"b6477143e17f889263044f6cf463dc37177ac4526c4c39a7a344198457024a2f",
		"05c3708d30c2c72c4b36314a41f30073ab18ea226cf8c6b9f566720bfe2e8631",
		"85a94dd51403590d4f149f9230b6f5de3a08e58899dcaf0f77768efb1825e854",
		"72efcf4bb0e8a300d3e47e6a10f630bcd540de933f01ed5380897fc5e10dc95d",
	}

	DefaultNodeAddrs = []string{
		"0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013",
		"0x79a1215469FaB6f9c63c1816b45183AD3624bE34",
		"0x97c8B516D19edBf575D72a172Af7F418BE498C37",
		"0xc0Ff2e0b3189132D815b8eb325bE17285AC898f8",
	}

	defaultNodeIDs = []string{
		"16Uiu2HAmJ38LwfY6pfgDWNvk3ypjcpEMSePNTE6Ma2NCLqjbZJSF",
		"16Uiu2HAmRypzJbdbUNYsCV2VVgv9UryYS5d7wejTJXT73mNLJ8AK",
		"16Uiu2HAmTwEET536QC9MZmYFp1NUshjRuaq5YSH1sLjW65WasvRk",
		"16Uiu2HAmQBFTnRr84M3xNhi3EcWmgZnnBsDgewk4sNtpA3smBsHJ",
	}
)
