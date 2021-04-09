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
	"github.com/meshplus/bitxhub-kit/storage/blockfile"
	"github.com/meshplus/bitxhub-kit/storage/leveldb"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/bitxhub/pkg/vm"
	"github.com/meshplus/bitxhub/pkg/vm/wasm/vmledger"
	libp2pcert "github.com/meshplus/go-libp2p-cert"
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

	bytes, err := ioutil.ReadFile("./testdata/ledger_test_gc.wasm")
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

	accountCache, err := ledger.NewAccountCache()
	require.Nil(t, err)
	logger := log.NewWithModule("account_test")
	blockFile, err := blockfile.NewBlockFile(dir, logger)
	assert.Nil(t, err)
	ldg, err := ledger.New(createMockRepo(t), store, ldb, blockFile, accountCache, log.NewWithModule("executor"))
	assert.Nil(t, err)

	return &vm.Context{
		Caller:          caller,
		Callee:          &types.Address{},
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

	accountCache, err := ledger.NewAccountCache()
	require.Nil(t, err)
	logger := log.NewWithModule("account_test")
	blockFile, err := blockfile.NewBlockFile(dir, logger)
	assert.Nil(t, err)
	ldg, err := ledger.New(createMockRepo(t), store, ldb, blockFile, accountCache, log.NewWithModule("executor"))
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

	accountCache, err := ledger.NewAccountCache()
	require.Nil(t, err)
	logger := log.NewWithModule("account_test")
	blockFile, err := blockfile.NewBlockFile(dir, logger)
	assert.Nil(t, err)
	ldg, err := ledger.New(createMockRepo(t), store, ldb, blockFile, accountCache, log.NewWithModule("executor"))
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
		Method: "state_test_set",
		Args: []*pb.Arg{
			{Type: pb.Arg_Bytes, Value: []byte("alice")},
			{Type: pb.Arg_Bytes, Value: []byte("111")},
		},
	}
	payload, err := invokePayload.Marshal()
	require.Nil(t, err)
	data := &pb.TransactionData{
		Payload: payload,
	}
	ctx1 := &vm.Context{
		Caller:          ctx.Caller,
		Callee:          types.NewAddress(ret),
		TransactionData: data,
		Ledger:          ctx.Ledger,
	}
	imports1, err := vmledger.New()
	require.Nil(t, err)
	fmt.Println(imports1)
	wasm1, err := New(ctx1, imports1, instances)
	require.Nil(t, err)
	fmt.Println(wasm1.w.Instance.Exports)

	result, err := wasm1.Run(payload)
	require.Nil(t, err)
	require.Equal(t, "1", string(result))

	invokePayload1 := &pb.InvokePayload{
		Method: "state_test_get",
		Args: []*pb.Arg{
			{Type: pb.Arg_Bytes, Value: []byte("alice")},
			{Type: pb.Arg_Bytes, Value: []byte("111")},
		},
	}
	payload1, err := invokePayload1.Marshal()
	require.Nil(t, err)
	// data1 := &pb.TransactionData{
	// 	Payload: payload1,
	// }
	// ctx2 := &vm.Context{
	// 	Caller:          ctx.Caller,
	// 	Callee:          types.NewAddress(ret),
	// 	TransactionData: data1,
	// 	Ledger:          ctx.Ledger,
	// }

	result1, err := wasm1.Run(payload1)
	require.Nil(t, err)
	require.Equal(t, "1", string(result1))
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
		Callee:          types.NewAddress(ret),
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

	accountCache, err := ledger.NewAccountCache()
	assert.Nil(b, err)
	logger := log.NewWithModule("account_test")
	blockFile, err := blockfile.NewBlockFile(dir, logger)
	assert.Nil(b, err)
	ldg, err := ledger.New(&repo.Repo{Key: &repo.Key{PrivKey: privKey}}, store, ldb, blockFile, accountCache, log.NewWithModule("executor"))
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
		Callee:          types.NewAddress(ret),
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

// TestWasm_RunValidation has been deprecated. See test in BitXHub-Core instead
func TestWasm_RunValidation(t *testing.T) {}

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
		Callee:          types.NewAddress(ret),
		TransactionData: data,
		Ledger:          ctx.Ledger,
	}
	imports1, err := vmledger.New()
	require.Nil(t, err)
	wasm1, err := New(ctx1, imports1, instances)
	require.Nil(t, err)

	_, err = wasm1.Run(payload)
	assert.Equal(t, errorLackOfMethod, err)
}

func createMockRepo(t *testing.T) *repo.Repo {
	key := `-----BEGIN EC PRIVATE KEY-----
BcNwjTDCxyxLNjFKQfMAc6sY6iJs+Ma59WZyC/4uhjE=
-----END EC PRIVATE KEY-----`

	privKey, err := libp2pcert.ParsePrivateKey([]byte(key), crypto.Secp256k1)
	require.Nil(t, err)

	address, err := privKey.PublicKey().Address()
	require.Nil(t, err)

	return &repo.Repo{
		Key: &repo.Key{
			PrivKey: privKey,
			Address: address.String(),
		},
	}
}

func TestContext(t *testing.T) {
	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	assert.Nil(t, err)
	bytes, err := ioutil.ReadFile("./testdata/fabric_policy.wasm")
	require.Nil(t, err)

	data := &pb.TransactionData{
		Payload: bytes,
	}

	addr, err := privKey.PublicKey().Address()
	require.Nil(t, err)

	dir := filepath.Join(os.TempDir(), "ctx_test")
	store, err := leveldb.New(filepath.Join(dir, "validation"))
	assert.Nil(t, err)
	ldb, err := leveldb.New(filepath.Join(dir, "ledger"))
	assert.Nil(t, err)

	accountCache, err := ledger.NewAccountCache()
	require.Nil(t, err)
	logger := log.NewWithModule("ctx_test")
	blockFile, err := blockfile.NewBlockFile(dir, logger)
	assert.Nil(t, err)
	ldg, err := ledger.New(createMockRepo(t), store, ldb, blockFile, accountCache, log.NewWithModule("executor"))
	require.Nil(t, err)

	tx := &pb.Transaction{
		From: addr,
		To:   addr,
	}

	ctx := NewContext(tx, data, ldg, logger)

	require.Equal(t, ctx.Caller(), addr.String())
	require.Equal(t, ctx.Callee(), addr.String())
	require.NotNil(t, ctx.Logger())
}
