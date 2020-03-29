package validator

type Validator interface {
	Verify(address, from string, proof []byte, validators string) (bool, error)
}
