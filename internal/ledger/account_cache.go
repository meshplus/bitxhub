package ledger

import (
	"bytes"
	"fmt"

	lru "github.com/hashicorp/golang-lru"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/eth-kit/ledger"
)

const (
	innerAccountCacheSize = 1024 * 1024
	codeCacheSize         = 1024 * 1024
	stateCacheLayer1Size  = 1024        // stateCache layer1 size(1KB)
	stateCacheLayer2Size  = 1024 * 1024 // stateCache layer2 size(1MB)
)

type AccountCache struct {
	innerAccountCache *lru.Cache
	stateCache        *lru.Cache // 2 layer cache: accountAddr -> stateKey -> stateValue
	codeCache         *lru.Cache
}

func NewAccountCache() (*AccountCache, error) {
	innerAccountCache, err := lru.New(innerAccountCacheSize)
	if err != nil {
		return nil, fmt.Errorf("init innerAccountCache failed: %w", err)
	}

	stateCache, err := lru.New(stateCacheLayer1Size)
	if err != nil {
		return nil, fmt.Errorf("init stateCache failed: %w", err)
	}

	codeCache, err := lru.New(codeCacheSize)
	if err != nil {
		return nil, fmt.Errorf("init codeCache failed: %w", err)
	}

	return &AccountCache{
		innerAccountCache: innerAccountCache,
		stateCache:        stateCache,
		codeCache:         codeCache,
	}, nil
}

func (ac *AccountCache) add(accounts map[string]ledger.IAccount) error {
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
			cache, err := lru.New(stateCacheLayer2Size)
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
	return nil
}

func (ac *AccountCache) rmAccount(addr *types.Address) {
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

func (ac *AccountCache) clear() {
	ac.innerAccountCache.Purge()
	ac.stateCache.Purge()
	ac.codeCache.Purge()
}
