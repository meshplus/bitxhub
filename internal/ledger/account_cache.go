package ledger

import (
	"bytes"
	"strings"
	"sync"
)

type AccountCache struct {
	innerAccounts map[string]*innerAccount
	states        map[string]map[string][]byte
	codes         map[string][]byte
	rwLock        sync.RWMutex
}

func NewAccountCache() *AccountCache {
	return &AccountCache{
		innerAccounts: make(map[string]*innerAccount),
		states:        make(map[string]map[string][]byte),
		codes:         make(map[string][]byte),
		rwLock:        sync.RWMutex{},
	}
}

func (ac *AccountCache) add(accounts map[string]*Account) {
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

func (ac *AccountCache) remove(accounts map[string]*Account) {
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

func (ac *AccountCache) getInnerAccount(addr string) (*innerAccount, bool) {
	ac.rwLock.RLock()
	defer ac.rwLock.RUnlock()

	if innerAccount, ok := ac.innerAccounts[addr]; ok {
		return innerAccount, true
	}

	return nil, false
}

func (ac *AccountCache) getState(addr string, key string) ([]byte, bool) {
	ac.rwLock.RLock()
	defer ac.rwLock.RUnlock()

	if stateMap, ok := ac.states[addr]; ok {
		if val, ok := stateMap[key]; ok {
			return val, true
		}
	}

	return nil, false
}

func (ac *AccountCache) getCode(addr string) ([]byte, bool) {
	ac.rwLock.RLock()
	defer ac.rwLock.RUnlock()

	if code, ok := ac.codes[addr]; ok {
		return code, true
	}

	return nil, false
}

func (ac *AccountCache) query(addr string, prefix string) map[string][]byte {
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

	ac.innerAccounts = make(map[string]*innerAccount)
	ac.states = make(map[string]map[string][]byte)
	ac.codes = make(map[string][]byte)
}
