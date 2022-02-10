package ledger

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/eth-kit/ledger"
)

type AccountCache struct {
	innerAccounts     map[string]*ledger.InnerAccount
	states            map[string]map[string][]byte
	codes             map[string][]byte
	innerAccountCache *lru.Cache
	stateCache        *lru.Cache
	codeCache         *lru.Cache
	rwLock            sync.RWMutex
}

func NewAccountCache() (*AccountCache, error) {
	innerAccountCache, err := lru.New(1024 * 1024)
	if err != nil {
		return nil, fmt.Errorf("init innerAccountCache failed: %w", err)
	}

	stateCache, err := lru.New(1024 * 1024)
	if err != nil {
		return nil, fmt.Errorf("init stateCache failed: %w", err)
	}

	codeCache, err := lru.New(1024 * 1024)
	if err != nil {
		return nil, fmt.Errorf("init codeCache failed: %w", err)
	}

	return &AccountCache{
		innerAccounts:     make(map[string]*ledger.InnerAccount),
		states:            make(map[string]map[string][]byte),
		codes:             make(map[string][]byte),
		innerAccountCache: innerAccountCache,
		stateCache:        stateCache,
		codeCache:         codeCache,
		rwLock:            sync.RWMutex{},
	}, nil
}

func (ac *AccountCache) add(accounts map[string]ledger.IAccount) error {
	ac.addToWriteBuffer(accounts)
	if err := ac.addToReadCache(accounts); err != nil {
		return fmt.Errorf("add accounts to read cache failed: %w", err)
	}
	return nil
}

func (ac *AccountCache) addToReadCache(accounts map[string]ledger.IAccount) error {
	for addr, acc := range accounts {
		account := acc.(*SimpleAccount)
		var stateCache *lru.Cache

		if account.dirtyAccount != nil {
			ac.innerAccountCache.Add(addr, account.dirtyAccount)
		}
		value, ok := ac.stateCache.Get(addr)
		if ok {
			stateCache = value.(*lru.Cache)
		} else {
			cache, err := lru.New(1024 * 1024)
			if err != nil {
				return fmt.Errorf("init lru cache failed: %w", err)
			}
			stateCache = cache
		}

		account.dirtyState.Range(func(key, value interface{}) bool {
			stateCache.Add(key, value)
			return true
		})

		if !ok && stateCache.Len() != 0 {
			ac.stateCache.Add(addr, stateCache)
		}

		if !bytes.Equal(account.originCode, account.dirtyCode) {
			if !bytes.Equal(account.originCode, account.dirtyCode) {
				if account.dirtyAccount != nil {
					if account.originAccount != nil {
						if !bytes.Equal(account.dirtyAccount.CodeHash, account.originAccount.CodeHash) {
							ac.codeCache.Add(addr, account.dirtyCode)
						}
					} else {
						ac.codeCache.Add(addr, account.dirtyCode)
					}
				}
			}
		}
	}
	return nil
}

func (ac *AccountCache) addToWriteBuffer(accounts map[string]ledger.IAccount) {
	ac.rwLock.Lock()
	defer ac.rwLock.Unlock()

	for addr, acc := range accounts {
		account := acc.(*SimpleAccount)
		if account.dirtyAccount != nil {
			ac.innerAccounts[addr] = account.dirtyAccount
		}
		stateMap, ok := ac.states[addr]
		if !ok {
			stateMap = make(map[string][]byte)
		}

		account.dirtyState.Range(func(key, value interface{}) bool {
			stateMap[key.(string)] = value.([]byte)
			return true
		})

		if !ok && len(stateMap) != 0 {
			ac.states[addr] = stateMap
		}

		if !bytes.Equal(account.originCode, account.dirtyCode) {
			if account.dirtyAccount != nil {
				if account.originAccount != nil {
					if !bytes.Equal(account.dirtyAccount.CodeHash, account.originAccount.CodeHash) {
						ac.codes[addr] = account.dirtyCode
					}
				} else {
					ac.codes[addr] = account.dirtyCode
				}
			}
		}
	}
}

func (ac *AccountCache) remove(accounts map[string]ledger.IAccount) {
	ac.rwLock.Lock()
	defer ac.rwLock.Unlock()

	for addr, acc := range accounts {
		account := acc.(*SimpleAccount)
		if innerAccount, ok := ac.innerAccounts[addr]; ok {
			if !ledger.InnerAccountChanged(innerAccount, account.dirtyAccount) {
				delete(ac.innerAccounts, addr)
			}
		}

		if stateMap, ok := ac.states[addr]; ok {
			account.dirtyState.Range(func(key, value interface{}) bool {
				if v, ok := stateMap[key.(string)]; ok {
					if bytes.Equal(v, value.([]byte)) {
						delete(stateMap, key.(string))
					}
				}
				return true
			})
			if len(stateMap) == 0 {
				delete(ac.states, addr)
			}
		}

		if !bytes.Equal(account.dirtyCode, account.originCode) {
			if code, ok := ac.codes[addr]; ok {
				if bytes.Equal(code, account.dirtyCode) {
					delete(ac.codes, addr)
				}
			}
		}
	}
}

func (ac *AccountCache) rmAccount(addr *types.Address) {
	ac.rwLock.Lock()
	defer ac.rwLock.Unlock()

	delete(ac.innerAccounts, addr.String())
	ac.innerAccountCache.Remove(addr.String())
}

func (ac *AccountCache) getInnerAccount(addr *types.Address) (*ledger.InnerAccount, bool) {
	if ia, ok := ac.innerAccountCache.Get(addr.String()); ok {
		return ia.(*ledger.InnerAccount), true
	}

	return nil, false
}

func (ac *AccountCache) getState(addr *types.Address, key string) ([]byte, bool) {
	if value, ok := ac.stateCache.Get(addr.String()); ok {
		if val, ok := value.(*lru.Cache).Get(key); ok {
			return val.([]byte), true
		}
	}

	return nil, false
}

func (ac *AccountCache) getCode(addr *types.Address) ([]byte, bool) {
	if code, ok := ac.codeCache.Get(addr.String()); ok {
		return code.([]byte), true
	}

	return nil, false
}

func (ac *AccountCache) query(addr *types.Address, prefix string) map[string][]byte {
	ac.rwLock.RLock()
	defer ac.rwLock.RUnlock()

	ret := make(map[string][]byte)

	if stateMap, ok := ac.states[addr.String()]; ok {
		for key, val := range stateMap {
			if strings.HasPrefix(key, prefix) {
				ret[key] = val
			}
		}
	}
	return ret
}

func (ac *AccountCache) clear() {
	ac.rwLock.Lock()
	defer ac.rwLock.Unlock()

	ac.innerAccounts = make(map[string]*ledger.InnerAccount)
	ac.states = make(map[string]map[string][]byte)
	ac.codes = make(map[string][]byte)
	ac.innerAccountCache.Purge()
	ac.stateCache.Purge()
	ac.codeCache.Purge()
}
