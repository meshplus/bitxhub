// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package filters

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	types2 "github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
)

// Filter can be used to retrieve and filter logs.
type Filter struct {
	api       api.CoreAPI
	addresses []*types2.Address
	topics    [][]*types2.Hash
	block     *types2.Hash // Block hash if filtering a single block
	begin     int64
	end       int64 // Range interval if filtering multiple blocks
}

type bytesBacked interface {
	Bytes() []byte
}

// NewRangeFilter creates a new filter which uses a bloom filter on blocks to
// figure out whether a particular block is interesting or not.
func NewRangeFilter(api api.CoreAPI, begin, end int64, addresses []*types2.Address, topics [][]*types2.Hash) *Filter {
	// Flatten the address and topic filter clauses into a single bloombits filter
	// system. Since the bloombits are not positional, nil topics are permitted,
	// which get flattened into a nil byte slice.
	var filters [][][]byte
	if len(addresses) > 0 {
		filter := make([][]byte, len(addresses))
		for i, address := range addresses {
			filter[i] = address.Bytes()
		}
		filters = append(filters, filter)
	}
	for _, topicList := range topics {
		filter := make([][]byte, len(topicList))
		for i, topic := range topicList {
			filter[i] = topic.Bytes()
		}
		filters = append(filters, filter)
	}

	// Create a generic filter and convert it into a range filter
	filter := newFilter(api, addresses, topics)

	filter.begin = begin
	filter.end = end

	return filter
}

// NewBlockFilter creates a new filter which directly inspects the contents of
// a block to figure out whether it is interesting or not.
func NewBlockFilter(api api.CoreAPI, block *types2.Hash, addresses []*types2.Address, topics [][]*types2.Hash) *Filter {
	// Create a generic filter and convert it into a block filter
	filter := newFilter(api, addresses, topics)
	filter.block = block
	return filter
}

// newFilter creates a generic filter that can either filter based on a block hash,
// or based on range queries. The search criteria needs to be explicitly set.
func newFilter(api api.CoreAPI, addresses []*types2.Address, topics [][]*types2.Hash) *Filter {
	return &Filter{
		api:       api,
		addresses: addresses,
		topics:    topics,
	}
}

// Logs searches the blockchain for matching log entries, returning all from the
// first block that contains matches, updating the start of the filter accordingly.
func (f *Filter) Logs(ctx context.Context) ([]*pb.EvmLog, error) {
	// If we're doing singleton block filtering, execute and return
	if f.block != nil {
		block, err := f.api.Broker().GetBlock("HASH", f.block.String())
		if err != nil {
			return nil, err
		}
		if block == nil {
			return nil, errors.New("unknown block")
		}
		return f.blockLogs(ctx, block.BlockHeader)
	}
	// Figure out the limits of the filter range
	meta, err := f.api.Chain().Meta()
	if err != nil {
		return nil, err
	}

	head := meta.Height
	if f.begin == -1 {
		f.begin = int64(head)
	}

	end := uint64(f.end)
	if f.end == -1 {
		end = head
	}

	return f.unindexedLogs(ctx, end)
}

// unindexedLogs returns the logs matching the filter criteria based on raw block
// iteration and bloom matching.
func (f *Filter) unindexedLogs(ctx context.Context, end uint64) ([]*pb.EvmLog, error) {
	var logs []*pb.EvmLog

	for ; f.begin <= int64(end); f.begin++ {
		headers, err := f.api.Broker().GetBlockHeaders(uint64(f.begin), uint64(f.begin))
		if headers == nil || err != nil {
			return logs, err
		}

		found, err := f.blockLogs(ctx, headers[0])
		if err != nil {
			return logs, err
		}
		logs = append(logs, found...)
	}
	return logs, nil
}

// blockLogs returns the logs matching the filter criteria within a single block.
func (f *Filter) blockLogs(ctx context.Context, header *pb.BlockHeader) (logs []*pb.EvmLog, err error) {
	if bloomFilter(header.Bloom, f.addresses, f.topics) {
		found, err := f.checkMatches(ctx, header.Number)
		if err != nil {
			return logs, err
		}
		logs = append(logs, found...)
	}
	return logs, nil
}

// checkMatches checks if the receipts belonging to the given header contain any log events that
// match the filter criteria. This function is called when the bloom filter signals a potential match.
func (f *Filter) checkMatches(ctx context.Context, blockNum uint64) (logs []*pb.EvmLog, err error) {
	// Get the logs of the block
	receipts, err := f.getBlockReceipts(blockNum)
	if err != nil {
		return nil, err
	}

	var unfiltered []*pb.EvmLog
	for _, receipt := range receipts {
		unfiltered = append(unfiltered, receipt.EvmLogs...)
	}
	return FilterLogs(unfiltered, nil, nil, f.addresses, f.topics), nil
}

func (f *Filter) getBlockReceipts(blockNum uint64) ([]*pb.Receipt, error) {
	var receipts []*pb.Receipt

	block, err := f.api.Broker().GetBlock("HEIGHT", fmt.Sprintf("%d", blockNum))
	if err != nil {
		return nil, err
	}

	for _, tx := range block.Transactions.Transactions {
		receipt, err := f.api.Broker().GetReceipt(tx.GetHash())
		if err != nil {
			return nil, err
		}

		receipts = append(receipts, receipt)
	}

	return receipts, nil
}

func includes(addresses []*types2.Address, a *types2.Address) bool {
	for _, addr := range addresses {
		if addr.String() == a.String() {
			return true
		}
	}

	return false
}

// FilterLogs creates a slice of logs matching the given criteria.
func FilterLogs(logs []*pb.EvmLog, fromBlock, toBlock *big.Int, addresses []*types2.Address, topics [][]*types2.Hash) []*pb.EvmLog {
	var ret []*pb.EvmLog
Logs:
	for _, log := range logs {
		if fromBlock != nil && fromBlock.Int64() >= 0 && fromBlock.Uint64() > log.BlockNumber {
			continue
		}
		if toBlock != nil && toBlock.Int64() >= 0 && toBlock.Uint64() < log.BlockNumber {
			continue
		}

		if len(addresses) > 0 && !includes(addresses, log.Address) {
			continue
		}
		// If the to filtered topics is greater than the amount of topics in logs, skip.
		if len(topics) > len(log.Topics) {
			continue Logs
		}
		for i, sub := range topics {
			match := len(sub) == 0 // empty rule set == wildcard
			for _, topic := range sub {
				if log.Topics[i].String() == topic.String() {
					match = true
					break
				}
			}
			if !match {
				continue Logs
			}
		}
		ret = append(ret, log)
	}
	return ret
}

func bloomFilter(bloom *types2.Bloom, addresses []*types2.Address, topics [][]*types2.Hash) bool {
	if len(addresses) > 0 {
		var included bool
		for _, addr := range addresses {
			if BloomLookup(bloom, addr) {
				included = true
				break
			}
		}
		if !included {
			return false
		}
	}

	for _, sub := range topics {
		included := len(sub) == 0 // empty rule set == wildcard
		for _, topic := range sub {
			if BloomLookup(bloom, topic) {
				included = true
				break
			}
		}
		if !included {
			return false
		}
	}
	return true
}

// BloomLookup is a convenience-method to check presence int he bloom filter
func BloomLookup(bin *types2.Bloom, topic bytesBacked) bool {
	return bin.Test(topic.Bytes())
}
