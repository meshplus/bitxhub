package ledger

import (
	"bytes"
	"fmt"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/axiomesh/axiom-kit/types"
)

const (
	accCacheSize      = 1024        // innerAccountCache,stateCache,codeCache layer1 size(1KB)
	accStateCacheSize = 1024 * 1024 // stateCache layer2 size(1MB)
)

type AccountCache struct {
	innerAccountCache *lru.Cache[string, *InnerAccount]

	// 2 layer cache: accountAddr -> stateKey -> stateValue
	stateCache *lru.Cache[string, *lru.Cache[string, []byte]]

	codeCache *lru.Cache[string, []byte]
}

func NewAccountCache() (*AccountCache, error) {
	innerAccountCache, err := lru.New[string, *InnerAccount](accCacheSize)
	if err != nil {
		return nil, fmt.Errorf("init innerAccountCache failed: %w", err)
	}

	stateCache, err := lru.New[string, *lru.Cache[string, []byte]](accCacheSize)
	if err != nil {
		return nil, fmt.Errorf("init stateCache failed: %w", err)
	}

	codeCache, err := lru.New[string, []byte](accCacheSize)
	if err != nil {
		return nil, fmt.Errorf("init codeCache failed: %w", err)
	}

	return &AccountCache{
		innerAccountCache: innerAccountCache,
		stateCache:        stateCache,
		codeCache:         codeCache,
	}, nil
}

func (ac *AccountCache) add(accounts map[string]IAccount) error {
	for addr, acc := range accounts {
		account := acc.(*SimpleAccount)
		var stateCache *lru.Cache[string, []byte]

		if account.dirtyAccount != nil {
			ac.innerAccountCache.Add(addr, account.dirtyAccount)
		}

		if account.dirtyState.Count() != 0 {
			value, ok := ac.stateCache.Get(addr)
			if ok {
				stateCache = value
			} else {
				cache, err := lru.New[string, []byte](accStateCacheSize)
				if err != nil {
					return fmt.Errorf("init lru cache failed: %w", err)
				}
				stateCache = cache
				ac.stateCache.Add(addr, stateCache)
			}

			for key, value := range account.dirtyState.Items() {
				stateCache.Add(key, value)
			}
		}

		if !bytes.Equal(account.originCode, account.dirtyCode) {
			ac.codeCache.Add(addr, account.dirtyCode)
		}
	}
	return nil
}

func (ac *AccountCache) rmAccount(addr *types.Address) {
	ac.innerAccountCache.Remove(addr.String())
	ac.codeCache.Remove(addr.String())
	ac.stateCache.Remove(addr.String())
}

func (ac *AccountCache) getInnerAccount(addr *types.Address) (*InnerAccount, bool) {
	return ac.innerAccountCache.Get(addr.String())
}

func (ac *AccountCache) getState(addr *types.Address, key string) ([]byte, bool) {
	if value, ok := ac.stateCache.Get(addr.String()); ok {
		if val, ok := value.Get(key); ok {
			return val, true
		}
	}

	return nil, false
}

func (ac *AccountCache) getCode(addr *types.Address) ([]byte, bool) {
	return ac.codeCache.Get(addr.String())
}

func (ac *AccountCache) clear() {
	ac.innerAccountCache.Purge()
	ac.stateCache.Purge()
	ac.codeCache.Purge()
}
