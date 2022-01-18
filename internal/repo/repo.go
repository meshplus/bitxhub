package repo

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/Knetic/govaluate"
	"github.com/ethereum/go-ethereum/event"
	libp2pcert "github.com/meshplus/go-libp2p-cert"
)

type Repo struct {
	Config           *Config
	NetworkConfig    *NetworkConfig
	Key              *Key
	Certs            *libp2pcert.Certs
	ConfigChangeFeed event.Feed
}

func (r *Repo) SubscribeConfigChange(ch chan *Config) event.Subscription {
	return r.ConfigChangeFeed.Subscribe(ch)
}

func Load(repoRoot string, passwd string) (*Repo, error) {
	config, err := UnmarshalConfig(repoRoot)
	if err != nil {
		return nil, err
	}

	if err := checkConfig(config); err != nil {
		return nil, err
	}

	networkConfig, err := loadNetworkConfig(repoRoot, config.Genesis)
	if err != nil {
		return nil, fmt.Errorf("load network config: %w", err)
	}

	certs, err := libp2pcert.LoadCerts(repoRoot, config.NodeCertPath, config.AgencyCertPath, config.CACertPath)
	if err != nil {
		return nil, err
	}

	key, err := loadPrivKey(repoRoot, passwd)
	if err != nil {
		return nil, fmt.Errorf("load private key: %w", err)
	}

	repo := &Repo{
		Config:        config,
		NetworkConfig: networkConfig,
		Key:           key,
		Certs:         certs,
	}

	WatchConfig(&repo.ConfigChangeFeed)

	return repo, nil
}

func checkConfig(config *Config) error {
	// check genesis admin info
	hasSuperAdmin := false
	for _, admin := range config.Genesis.Admins {
		if admin.Weight == SuperAdminWeight {
			hasSuperAdmin = true
		} else if admin.Weight != NormalAdminWeight {
			return fmt.Errorf("Illegal admin weight in genesis config!")
		}
	}

	if !hasSuperAdmin {
		return fmt.Errorf("Set up at least one super administrator in genesis config!")
	}

	// check strategy
	for _, s := range config.Genesis.Strategy {
		if err := CheckStrategyInfo(s.Typ, s.Module, s.Extra); err != nil {
			return err
		}
	}
	return nil
}

func CheckStrategyInfo(typ, module string, extra string) error {
	if err := CheckStrategyType(typ, extra); err != nil {
		return fmt.Errorf("illegal proposal strategy type:%s, err: %v", typ, err)
	}
	if CheckManageModule(module) != nil {
		return fmt.Errorf("illegal proposal strategy module:%s", typ)
	}
	return nil
}

func CheckStrategyType(typ string, extra string) error {
	if typ != SuperMajorityApprove &&
		typ != SuperMajorityAgainst &&
		typ != SimpleMajority {
		return fmt.Errorf("illegal proposal strategy type")
	}

	isEnd, _, err := MakeStrategyDecision(extra, 1, 0, 1, 1)
	if err != nil {
		return fmt.Errorf("illegal strategy expression: %w", err)
	}
	if !isEnd {
		return fmt.Errorf("illegal strategy expression: under this exp(%s), the proposal may never be concluded", extra)
	}

	return nil
}

// return:
// - whether the proposal is over, if not, you need to wait for the vote
// - whether the proposal is pass, that is, it has ended, may be passed or rejected
// - error
func MakeStrategyDecision(expressionStr string, approve, reject, total, avaliableNum uint64) (bool, bool, error) {
	expression, err := govaluate.NewEvaluableExpression(expressionStr)
	if err != nil {
		return false, false, err
	}

	parameters := make(map[string]interface{}, 8)
	parameters["a"] = approve
	parameters["r"] = reject
	parameters["t"] = total
	result, err := expression.Evaluate(parameters)
	if err != nil {
		return false, false, err
	}

	if result.(bool) {
		return true, true, nil
	}

	parameters["a"] = avaliableNum - reject
	result, err = expression.Evaluate(parameters)
	if err != nil {
		return false, false, err
	}

	if result.(bool) {
		return false, false, nil
	} else {
		return true, false, nil
	}
}

func CheckManageModule(moduleTyp string) error {
	if moduleTyp != AppchainMgr &&
		moduleTyp != RoleMgr &&
		moduleTyp != RuleMgr &&
		moduleTyp != DappMgr &&
		moduleTyp != NodeMgr &&
		moduleTyp != ServiceMgr {
		return fmt.Errorf("illegal manage module type")
	}
	return nil
}

func GetAPI(repoRoot string) (string, error) {
	data, err := ioutil.ReadFile(filepath.Join(repoRoot, APIName))
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func GetKeyPath(repoRoot string) string {
	return filepath.Join(repoRoot, KeyName)
}

func GetStoragePath(repoRoot string, subPath ...string) string {
	p := filepath.Join(repoRoot, "storage")
	for _, s := range subPath {
		p = filepath.Join(p, s)
	}

	return p
}
