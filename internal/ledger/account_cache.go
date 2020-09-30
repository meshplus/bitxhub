package ledger

import (
	"bytes"
	"strings"
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"github.com/meshplus/bitxhub-kit/types"
)

type AccountCache struct {
	innerAccounts     map[types.Address]*innerAccount
	states            map[types.Address]map[string][]byte
	codes             map[types.Address][]byte
	innerAccountCache *lru.Cache
	stateCache        *lru.Cache
	codeCache         *lru.Cache
	rwLock            sync.RWMutex
}

func NewAccountCache() (*AccountCache, error) {
	innerAccountCache, err := lru.New(1024 * 1024)
	if err != nil {
		return nil, err
	}

	stateCache, err := lru.New(1024 * 1024)
	if err != nil {
		return nil, err
	}

	codeCache, err := lru.New(1024 * 1024)
	if err != nil {
		return nil, err
	}

	return &AccountCache{
		innerAccounts:     make(map[types.Address]*innerAccount),
		states:            make(map[types.Address]map[string][]byte),
		codes:             make(map[types.Address][]byte),
		innerAccountCache: innerAccountCache,
		stateCache:        stateCache,
		codeCache:         codeCache,
		rwLock:            sync.RWMutex{},
	}, nil
}

func (ac *AccountCache) add(accounts map[types.Address]*Account) error {
	ac.addToWriteBuffer(accounts)
	if err := ac.addToReadCache(accounts); err != nil {
		return err
	}
	return nil
}

func (ac *AccountCache) addToReadCache(accounts map[types.Address]*Account) error {
	for addr, account := range accounts {
		var stateCache *lru.Cache

		ac.innerAccountCache.Add(addr, account.dirtyAccount)
		value, ok := ac.stateCache.Get(addr)
		if ok {
			stateCache = value.(*lru.Cache)
		} else {
			cache, err := lru.New(1024 * 1024)
			if err != nil {
				return err
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
			ac.codeCache.Add(addr, account.dirtyCode)
		}
	}
	return nil
}

func (ac *AccountCache) addToWriteBuffer(accounts map[types.Address]*Account) {
	ac.rwLock.Lock()
	defer ac.rwLock.Unlock()

	for addr, account := range accounts {
		ac.innerAccounts[addr] = account.dirtyAccount
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
			ac.codes[addr] = account.dirtyCode
		}
	}
}

func (ac *AccountCache) remove(accounts map[types.Address]*Account) {
	ac.rwLock.Lock()
	defer ac.rwLock.Unlock()

	for addr, account := range accounts {
		if innerAccount, ok := ac.innerAccounts[addr]; ok {
			if !innerAccountChanged(innerAccount, account.dirtyAccount) {
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

func (ac *AccountCache) getInnerAccount(addr types.Address) (*innerAccount, bool) {
	if ia, ok := ac.innerAccountCache.Get(addr); ok {
		return ia.(*innerAccount), true
	}

	return nil, false
}

func (ac *AccountCache) getState(addr types.Address, key string) ([]byte, bool) {
	if value, ok := ac.stateCache.Get(addr); ok {
		if val, ok := value.(*lru.Cache).Get(key); ok {
			return val.([]byte), true
		}
	}

	return nil, false
}

func (ac *AccountCache) getCode(addr types.Address) ([]byte, bool) {
	if code, ok := ac.codeCache.Get(addr); ok {
		return code.([]byte), true
	}

	return nil, false
}

func (ac *AccountCache) query(addr types.Address, prefix string) map[string][]byte {
	ac.rwLock.RLock()
	defer ac.rwLock.RUnlock()

	ret := make(map[string][]byte)

	if stateMap, ok := ac.states[addr]; ok {
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

	ac.innerAccounts = make(map[types.Address]*innerAccount)
	ac.states = make(map[types.Address]map[string][]byte)
	ac.codes = make(map[types.Address][]byte)
	ac.innerAccountCache.Purge()
	ac.stateCache.Purge()
	ac.codeCache.Purge()
}
