package validatorlib

// #include <stdlib.h>
//
// extern int32_t fabric_validate_v13(void *context, long long proof_ptr, long long validator_ptr);
import "C"
import (
	"unsafe"

	"github.com/wasmerio/go-ext-wasm/wasmer"
)

//export fabric_validate_v13
func fabric_validate_v13(context unsafe.Pointer, proof_ptr int64, validator_ptr int64) int32 {
	ctx := wasmer.IntoInstanceContext(context)
	data := ctx.Data().(map[int]int)
	memory := ctx.Memory()
	proof := memory.Data()[proof_ptr : proof_ptr+int64(data[int(proof_ptr)])]
	validator := memory.Data()[validator_ptr : validator_ptr+int64(data[int(validator_ptr)])]
	vInfo, err := UnmarshalValidatorInfo(validator)
	if err != nil {
		return 0
	}
	err = ValidateV14(proof, []byte(vInfo.Policy), vInfo.ConfByte, vInfo.Cid)
	if err != nil {
		return 0
	}

	return 1
}

func (im *Imports) importFabricV13() {
	var err error
	im.imports, err = im.imports.Append("fabric_validate_v13", fabric_validate_v13, C.fabric_validate_v13)
	if err != nil {
		return
	}
}
