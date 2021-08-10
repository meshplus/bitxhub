package vm

// VM is the basic interface for an implementation of the VM.
type VM interface {
	// Run should execute the given contract with the given input
	// and return the contract execution return bytes or an error if it
	// failed.
	Run(input []byte, gasLimit uint64) ([]byte, uint64, error)
}
