// Package nativechain is an isolated nonpayable integration harness, not a precompile registration.
package nativechain

/*
#cgo CFLAGS: -I${SRCDIR}/../../packages/vm/lithovm-ffi/include
#cgo !lithovm_release LDFLAGS: -L${SRCDIR}/../../target/debug -llithovm_ffi -Wl,-rpath,${SRCDIR}/../../target/debug
#cgo lithovm_release LDFLAGS: -L${SRCDIR}/../../target/release -llithovm_ffi -Wl,-rpath,${SRCDIR}/../../target/release
#include "lithovm.h"
extern intptr_t lithovmGoRead(uintptr_t, uint8_t *, size_t, uint8_t *, size_t);
static int32_t executeGo(const uint8_t *input, size_t len, uintptr_t handle, LithoBuffer *output) {
    return lithovm_execute_v1(input, len, handle, (LithoRead)lithovmGoRead, output);
}
*/
import "C"

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/cgo"
	"strconv"
	"strings"
	"unsafe"
)

const MaxBytes = 2 * 1024 * 1024

type Request struct {
	Version        uint8   `json:"version"`
	Operation      string  `json:"operation"`
	Caller         string  `json:"caller"`
	Contract       *string `json:"contract,omitempty"`
	Bytecode       *string `json:"bytecode,omitempty"`
	Function       *string `json:"function,omitempty"`
	Selector       *string `json:"selector,omitempty"`
	Arguments      string  `json:"arguments"`
	Nonce          uint64  `json:"nonce"`
	GasLimit       uint64  `json:"gas_limit"`
	ChainID        uint64  `json:"chain_id"`
	BlockHeight    uint64  `json:"block_height"`
	BlockTimestamp uint64  `json:"block_timestamp"`
}
type Write struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
type Response struct {
	Success     bool              `json:"success"`
	GasUsed     string            `json:"gasUsed"`
	Kind        string            `json:"kind"`
	Message     string            `json:"message"`
	Result      string            `json:"result"`
	Writes      []Write           `json:"writes"`
	Deployments []json.RawMessage `json:"deployments"`
	Events      []json.RawMessage `json:"events"`
}
type Reader func(string) ([]byte, error)

func validStateKey(key string) bool {
	const prefix = "lithovm/v1/contracts/"
	suffix := strings.TrimPrefix(key, prefix)
	if !strings.HasPrefix(key, prefix) || len(suffix) != 40 || strings.ToLower(suffix) != suffix {
		return false
	}
	_, err := hex.DecodeString(suffix)
	return err == nil
}

//export lithovmGoRead
func lithovmGoRead(handle C.uintptr_t, key *C.uint8_t, keyLen C.size_t, output *C.uint8_t, capacity C.size_t) (result C.intptr_t) {
	result = -2
	defer func() {
		if recover() != nil {
			result = -2
		}
	}()
	if keyLen == 0 || keyLen > 128 || capacity > MaxBytes {
		return -2
	}
	stateKey := string(unsafe.Slice((*byte)(unsafe.Pointer(key)), int(keyLen)))
	if !validStateKey(stateKey) {
		return -2
	}
	read := cgo.Handle(handle).Value().(Reader)
	value, err := read(stateKey)
	if err != nil || len(value) > MaxBytes {
		return -2
	}
	if value == nil {
		return -1
	}
	if capacity == 0 {
		return C.intptr_t(len(value))
	}
	if output == nil || len(value) != int(capacity) {
		return -2
	}
	copy(unsafe.Slice((*byte)(unsafe.Pointer(output)), int(capacity)), value)
	return C.intptr_t(len(value))
}

// Execute never writes external state. The embedding EVM cache owns applying and reverting the batch.
func Execute(request Request, read Reader) (Response, error) {
	var response Response
	if read == nil {
		return response, errors.New("state reader required")
	}
	input, err := json.Marshal(request)
	if err != nil || len(input) > MaxBytes {
		return response, errors.New("invalid or oversized request")
	}
	handle := cgo.NewHandle(read)
	defer handle.Delete()
	var output C.LithoBuffer
	code := C.executeGo((*C.uint8_t)(unsafe.Pointer(&input[0])), C.size_t(len(input)), C.uintptr_t(handle), &output)
	if code != 0 {
		return response, fmt.Errorf("native FFI rejected request: %d", code)
	}
	defer C.lithovm_free_v1(output)
	if output.data == nil || output.len > MaxBytes {
		return response, errors.New("invalid native buffer")
	}
	if err := json.Unmarshal(C.GoBytes(unsafe.Pointer(output.data), C.int(output.len)), &response); err != nil {
		return response, err
	}
	gas, err := strconv.ParseUint(response.GasUsed, 10, 64)
	if err != nil || gas > request.GasLimit {
		return Response{}, errors.New("invalid native gas")
	}
	if !response.Success && (len(response.Writes) != 0 || len(response.Deployments) != 0 || len(response.Events) != 0) {
		return Response{}, errors.New("failed execution leaked effects")
	}
	_, err = response.DecodedWrites()
	return response, err
}

// DecodedWrites validates the complete batch before the first external write.
func (response Response) DecodedWrites() (map[string][]byte, error) {
	if len(response.Writes) > 128 {
		return nil, errors.New("too many writes")
	}
	writes := make(map[string][]byte)
	for _, write := range response.Writes {
		if !validStateKey(write.Key) {
			return nil, errors.New("invalid state key")
		}
		if _, exists := writes[write.Key]; exists {
			return nil, errors.New("duplicate state key")
		}
		if !strings.HasPrefix(write.Value, "0x") {
			return nil, errors.New("invalid state value")
		}
		value, err := hex.DecodeString(write.Value[2:])
		if err != nil || len(value) > MaxBytes {
			return nil, errors.New("invalid state value")
		}
		writes[write.Key] = value
	}
	return writes, nil
}
