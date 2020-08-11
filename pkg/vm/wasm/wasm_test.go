package wasm

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/meshplus/bitxhub-core/validator/validatorlib"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/pkg/storage/leveldb"
	"github.com/meshplus/bitxhub/pkg/vm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wasmerio/go-ext-wasm/wasmer"
)

const cert1 = `-----BEGIN CERTIFICATE-----
MIICKDCCAc+gAwIBAgIRAIvRdkwS+++KkoPliLaqSF0wCgYIKoZIzj0EAwIwczEL
MAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExFjAUBgNVBAcTDVNhbiBG
cmFuY2lzY28xGTAXBgNVBAoTEG9yZzIuZXhhbXBsZS5jb20xHDAaBgNVBAMTE2Nh
Lm9yZzIuZXhhbXBsZS5jb20wHhcNMTkwODI3MDgwOTAwWhcNMjkwODI0MDgwOTAw
WjBqMQswCQYDVQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTEWMBQGA1UEBxMN
U2FuIEZyYW5jaXNjbzENMAsGA1UECxMEcGVlcjEfMB0GA1UEAxMWcGVlcjEub3Jn
Mi5leGFtcGxlLmNvbTBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABL10V0Smz2FO
HR82njo9H0HNyNWt/JUKWMt8Olx425u5y2kqs5RjAMRdyz3U3N0Tve1znDbCjoDL
WtrXW0WdKSSjTTBLMA4GA1UdDwEB/wQEAwIHgDAMBgNVHRMBAf8EAjAAMCsGA1Ud
IwQkMCKAIG4CB58dq07RqnQt9GKmMABv6hWRK0oDO4FBHFtpDlXUMAoGCCqGSM49
BAMCA0cAMEQCIFuh8p+nbtjQEZEFg03BN58//9VRsukQXj0xP1eHnrD4AiBwI1jq
L6FMy96mi64g37R0i/I+T4MC5p2mzZIHvRJ8Rg==
-----END CERTIFICATE-----`

func initCreateContext(t *testing.T, name string) *vm.Context {
	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	assert.Nil(t, err)
	dir := filepath.Join(os.TempDir(), "wasm", name)

	bytes, err := ioutil.ReadFile("./testdata/wasm_test.wasm")
	assert.Nil(t, err)

	data := &pb.TransactionData{
		Payload: bytes,
	}

	caller, err := privKey.PublicKey().Address()
	assert.Nil(t, err)

	store, err := leveldb.New(filepath.Join(dir, "wasm"))
	assert.Nil(t, err)

	ldb, err := leveldb.New(filepath.Join(dir, "ledger"))
	assert.Nil(t, err)

	ldg, err := ledger.New(store, ldb, ledger.NewAccountCache(), log.NewWithModule("executor"))
	assert.Nil(t, err)

	return &vm.Context{
		Caller:          caller,
		TransactionData: data,
		Ledger:          ldg,
	}
}

func initValidationContext(t *testing.T, name string) *vm.Context {
	dir := filepath.Join(os.TempDir(), "validation", name)

	bytes, err := ioutil.ReadFile("./testdata/validation_test.wasm")
	require.Nil(t, err)
	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	require.Nil(t, err)

	data := &pb.TransactionData{
		Payload: bytes,
	}

	caller, err := privKey.PublicKey().Address()
	require.Nil(t, err)

	store, err := leveldb.New(filepath.Join(dir, "validation"))
	assert.Nil(t, err)
	ldb, err := leveldb.New(filepath.Join(dir, "ledger"))
	assert.Nil(t, err)

	ldg, err := ledger.New(store, ldb, ledger.NewAccountCache(), log.NewWithModule("executor"))
	require.Nil(t, err)

	return &vm.Context{
		Caller:          caller,
		TransactionData: data,
		Ledger:          ldg,
	}
}

func initFabricContext(t *testing.T, name string) *vm.Context {
	dir := filepath.Join(os.TempDir(), "fabric_policy", name)

	bytes, err := ioutil.ReadFile("./testdata/fabric_policy.wasm")
	require.Nil(t, err)
	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	require.Nil(t, err)

	data := &pb.TransactionData{
		Payload: bytes,
	}

	caller, err := privKey.PublicKey().Address()
	require.Nil(t, err)

	store, err := leveldb.New(filepath.Join(dir, "validation_farbic"))
	assert.Nil(t, err)
	ldb, err := leveldb.New(filepath.Join(dir, "ledger"))
	assert.Nil(t, err)

	ldg, err := ledger.New(store, ldb, ledger.NewAccountCache(), log.NewWithModule("executor"))
	require.Nil(t, err)

	return &vm.Context{
		Caller:          caller,
		TransactionData: data,
		Ledger:          ldg,
	}
}

func TestDeploy(t *testing.T) {
	ctx := initCreateContext(t, "create")
	instances := make(map[string]wasmer.Instance)
	imports, err := EmptyImports()
	require.Nil(t, err)
	wasm, err := New(ctx, imports, instances)
	require.Nil(t, err)

	_, err = wasm.deploy()
	require.Nil(t, err)
}

func TestExecute(t *testing.T) {
	ctx := initCreateContext(t, "execute")
	instances := make(map[string]wasmer.Instance)
	imports, err := EmptyImports()
	require.Nil(t, err)
	wasm, err := New(ctx, imports, instances)
	require.Nil(t, err)

	ret, err := wasm.deploy()
	require.Nil(t, err)

	invokePayload := &pb.InvokePayload{
		Method: "a",
		Args: []*pb.Arg{
			{Type: pb.Arg_I32, Value: []byte(fmt.Sprintf("%d", 1))},
			{Type: pb.Arg_I32, Value: []byte(fmt.Sprintf("%d", 2))},
		},
	}
	payload, err := invokePayload.Marshal()
	require.Nil(t, err)
	data := &pb.TransactionData{
		Payload: payload,
	}
	ctx1 := &vm.Context{
		Caller:          ctx.Caller,
		Callee:          types.Bytes2Address(ret),
		TransactionData: data,
		Ledger:          ctx.Ledger,
	}
	imports1, err := validatorlib.New()
	require.Nil(t, err)
	wasm1, err := New(ctx1, imports1, instances)
	require.Nil(t, err)

	result, err := wasm1.Run(payload)
	require.Nil(t, err)
	require.Equal(t, "336", string(result))
}

func TestWasm_RunFabValidation(t *testing.T) {
	ctx := initFabricContext(t, "execute")
	instances := make(map[string]wasmer.Instance)
	imports, err := EmptyImports()
	require.Nil(t, err)
	wasm, err := New(ctx, imports, instances)
	require.Nil(t, err)

	ret, err := wasm.deploy()
	require.Nil(t, err)

	ibtpBytes, err := ioutil.ReadFile("./testdata/ibtp")
	require.Nil(t, err)
	ibtp := &pb.IBTP{}
	err = proto.Unmarshal(ibtpBytes, ibtp)
	require.Nil(t, err)

	validator, err := ioutil.ReadFile("./testdata/validator")
	require.Nil(t, err)
	invokePayload := &pb.InvokePayload{
		Method: "start_verify",
		Args: []*pb.Arg{
			{Type: pb.Arg_Bytes, Value: ibtp.Proof},
			{Type: pb.Arg_Bytes, Value: validator},
			{Type: pb.Arg_Bytes, Value: ibtp.Payload},
		},
	}
	payload, err := invokePayload.Marshal()
	require.Nil(t, err)
	data := &pb.TransactionData{
		Payload: payload,
	}
	ctx1 := &vm.Context{
		Caller:          ctx.Caller,
		Callee:          types.Bytes2Address(ret),
		TransactionData: data,
		Ledger:          ctx.Ledger,
	}
	imports1, err := validatorlib.New()
	require.Nil(t, err)
	wasm1, err := New(ctx1, imports1, instances)
	require.Nil(t, err)

	result, err := wasm1.Run(payload)
	require.Nil(t, err)
	require.Equal(t, "0", string(result))
}

func BenchmarkRunFabValidation(b *testing.B) {
	dir := filepath.Join(os.TempDir(), "bmark", "execute")

	bytes, err := ioutil.ReadFile("./testdata/fabric_policy.wasm")
	require.Nil(b, err)
	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	require.Nil(b, err)

	data := &pb.TransactionData{
		Payload: bytes,
	}

	caller, err := privKey.PublicKey().Address()
	require.Nil(b, err)

	store, err := leveldb.New(filepath.Join(dir, "111"))
	assert.Nil(b, err)
	ldb, err := leveldb.New(filepath.Join(dir, "ledger"))
	assert.Nil(b, err)

	ldg, err := ledger.New(store, ldb, ledger.NewAccountCache(), log.NewWithModule("executor"))
	require.Nil(b, err)
	ctx := &vm.Context{
		Caller:          caller,
		TransactionData: data,
		Ledger:          ldg,
	}
	instances := make(map[string]wasmer.Instance)
	imports, err := EmptyImports()
	require.Nil(b, err)
	wasm, err := New(ctx, imports, instances)
	require.Nil(b, err)

	ret, err := wasm.deploy()
	require.Nil(b, err)

	proof, err := ioutil.ReadFile("./testdata/proof")
	require.Nil(b, err)
	validator, err := ioutil.ReadFile("./testdata/validator")
	require.Nil(b, err)
	invokePayload := &pb.InvokePayload{
		Method: "start_verify",
		Args: []*pb.Arg{
			{Type: pb.Arg_Bytes, Value: proof},
			{Type: pb.Arg_Bytes, Value: validator},
		},
	}
	payload, err := invokePayload.Marshal()
	require.Nil(b, err)
	ctx1 := &vm.Context{
		Caller:          ctx.Caller,
		Callee:          types.Bytes2Address(ret),
		TransactionData: data,
		Ledger:          ctx.Ledger,
	}
	for i := 0; i < b.N; i++ {
		imports1, err := validatorlib.New()
		require.Nil(b, err)
		wasm1, err := New(ctx1, imports1, instances)
		require.Nil(b, err)

		result, err := wasm1.Run(payload)
		require.Nil(b, err)
		require.Equal(b, "0", string(result))
	}
	ctx.Ledger.Close()
	store.Close()
}

func TestWasm_RunValidation(t *testing.T) {
	ctx := initValidationContext(t, "execute")
	instances := make(map[string]wasmer.Instance)
	imports, err := EmptyImports()
	require.Nil(t, err)
	wasm, err := New(ctx, imports, instances)
	require.Nil(t, err)

	ret, err := wasm.deploy()
	require.Nil(t, err)

	bytes, err := ioutil.ReadFile("./testdata/fab_test")
	require.Nil(t, err)
	invokePayload := &pb.InvokePayload{
		Method: "start_verify",
		Args: []*pb.Arg{
			{Type: pb.Arg_Bytes, Value: bytes},
			{Type: pb.Arg_Bytes, Value: []byte(cert1)},
		},
	}
	payload, err := invokePayload.Marshal()
	require.Nil(t, err)
	data := &pb.TransactionData{
		Payload: payload,
	}
	ctx1 := &vm.Context{
		Caller:          ctx.Caller,
		Callee:          types.Bytes2Address(ret),
		TransactionData: data,
		Ledger:          ctx.Ledger,
	}
	imports1, err := validatorlib.New()
	require.Nil(t, err)
	wasm1, err := New(ctx1, imports1, instances)
	require.Nil(t, err)

	result, err := wasm1.Run(payload)
	require.Nil(t, err)
	require.Equal(t, "1", string(result))
}

func TestWasm_RunWithoutMethod(t *testing.T) {
	ctx := initCreateContext(t, "execute_without_method")
	instances := make(map[string]wasmer.Instance)
	imports, err := EmptyImports()
	require.Nil(t, err)
	wasm, err := New(ctx, imports, instances)
	require.Nil(t, err)

	ret, err := wasm.deploy()
	require.Nil(t, err)

	pl := &pb.InvokePayload{
		// Method: "",
		Args: []*pb.Arg{
			{Type: pb.Arg_I32, Value: []byte(fmt.Sprintf("%d", 1))},
			{Type: pb.Arg_I32, Value: []byte(fmt.Sprintf("%d", 2))},
		},
	}
	payload, err := pl.Marshal()
	require.Nil(t, err)
	data := &pb.TransactionData{
		Payload: payload,
	}
	ctx1 := &vm.Context{
		Caller:          ctx.Caller,
		Callee:          types.Bytes2Address(ret),
		TransactionData: data,
		Ledger:          ctx.Ledger,
	}
	imports1, err := validatorlib.New()
	require.Nil(t, err)
	wasm1, err := New(ctx1, imports1, instances)
	require.Nil(t, err)

	_, err = wasm1.Run(payload)
	assert.Equal(t, errorLackOfMethod, err)
}
