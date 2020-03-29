package wasm

// SetString set the string type arg for wasm
func (w *Wasm) SetString(str string) (int32, error) {
	alloc := w.Instance.Exports["allocate"]
	lengthOfStr := len(str)

	allocResult, err := alloc(lengthOfStr)
	if err != nil {
		return 0, err
	}
	inputPointer := allocResult.ToI32()

	memory := w.Instance.Memory.Data()[inputPointer:]

	var i int
	for i = 0; i < lengthOfStr; i++ {
		memory[i] = str[i]
	}

	memory[i] = 0
	w.argMap[int(inputPointer)] = len(str)

	return inputPointer, nil
}

// SetBytes set bytes type arg for wasm
func (w *Wasm) SetBytes(b []byte) (int32, error) {
	alloc := w.Instance.Exports["allocate"]
	lengthOfBytes := len(b)

	allocResult, err := alloc(lengthOfBytes)
	if err != nil {
		return 0, err
	}
	inputPointer := allocResult.ToI32()

	memory := w.Instance.Memory.Data()[inputPointer:]

	var i int
	for i = 0; i < lengthOfBytes; i++ {
		memory[i] = b[i]
	}

	memory[i] = 0
	w.argMap[int(inputPointer)] = len(b)

	return inputPointer, nil
}
