package executor

func (exec *BlockExecutor) isDemandNumber(num uint64) bool {
	return exec.currentHeight+1 == num
}

func (exec *BlockExecutor) getDemandNumber() uint64 {
	return exec.currentHeight + 1
}
