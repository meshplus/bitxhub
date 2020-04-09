package validator

// Engine runs for validation
type Engine interface {
	Validate(address, from string, proof []byte, validators string) (bool, error)
}

// Validator chooses specific method to verify transaction
type Validator interface {
	Verify(address, from string, proof []byte, validators string) (bool, error)
}
