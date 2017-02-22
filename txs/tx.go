// Copyright 2017 Monax Industries Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package txs

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"

	"golang.org/x/crypto/ripemd160"

	acm "github.com/eris-ltd/eris-db/account"
	ptypes "github.com/eris-ltd/eris-db/permission/types"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-wire"

	"github.com/tendermint/go-crypto"
	tendermint_types "github.com/tendermint/tendermint/types" // votes for dupeout ..
)

var (
	ErrTxInvalidAddress       = errors.New("Error invalid address")
	ErrTxDuplicateAddress     = errors.New("Error duplicate address")
	ErrTxInvalidAmount        = errors.New("Error invalid amount")
	ErrTxInsufficientFunds    = errors.New("Error insufficient funds")
	ErrTxInsufficientGasPrice = errors.New("Error insufficient gas price")
	ErrTxUnknownPubKey        = errors.New("Error unknown pubkey")
	ErrTxInvalidPubKey        = errors.New("Error invalid pubkey")
	ErrTxInvalidSignature     = errors.New("Error invalid signature")
	ErrTxPermissionDenied     = errors.New("Error permission denied")
)

type ErrTxInvalidString struct {
	Msg string
}

func (e ErrTxInvalidString) Error() string {
	return e.Msg
}

type ErrTxInvalidSequence struct {
	Got      int
	Expected int
}

func (e ErrTxInvalidSequence) Error() string {
	return Fmt("Error invalid sequence. Got %d, expected %d", e.Got, e.Expected)
}

/*
Tx (Transaction) is an atomic operation on the ledger state.

Account Txs:
 - SendTx         Send coins to address
 - CallTx         Send a msg to a contract that runs in the vm
 - NameTx	  Store some value under a name in the global namereg

Validation Txs:
 - BondTx         New validator posts a bond
 - UnbondTx       Validator leaves
 - DupeoutTx      Validator dupes out (equivocates)

Admin Txs:
 - PermissionsTx
*/

// Types of Tx implementations
const (
	// Account transactions
	TxTypeSend = byte(0x01)
	TxTypeCall = byte(0x02)
	TxTypeName = byte(0x03)

	// Validation transactions
	TxTypeBond    = byte(0x11)
	TxTypeUnbond  = byte(0x12)
	TxTypeRebond  = byte(0x13)
	TxTypeDupeout = byte(0x14)

	// Admin transactions
	TxTypePermissions = byte(0x20)
)

// for wire.readReflect
var _ = wire.RegisterInterface(
	struct{ Tx }{},
	wire.ConcreteType{&SendTx{}, TxTypeSend},
	wire.ConcreteType{&CallTx{}, TxTypeCall},
	wire.ConcreteType{&NameTx{}, TxTypeName},
	wire.ConcreteType{&BondTx{}, TxTypeBond},
	wire.ConcreteType{&UnbondTx{}, TxTypeUnbond},
	wire.ConcreteType{&RebondTx{}, TxTypeRebond},
	wire.ConcreteType{&DupeoutTx{}, TxTypeDupeout},
	wire.ConcreteType{&PermissionsTx{}, TxTypePermissions},
)

//-----------------------------------------------------------------------------

type (
	Tx interface {
		WriteSignBytes(chainID string, w io.Writer, n *int, err *error)
	}

	// UnconfirmedTxs
	UnconfirmedTxs struct {
		Txs []Tx `json:"txs"`
	}

	SendTx struct {
		Inputs  []*TxInput  `json:"inputs"`
		Outputs []*TxOutput `json:"outputs"`
	}

	// BroadcastTx or Transact
	Receipt struct {
		TxHash          []byte `json:"tx_hash"`
		CreatesContract uint8  `json:"creates_contract"`
		ContractAddr    []byte `json:"contract_addr"`
	}

	NameTx struct {
		Input *TxInput `json:"input"`
		Name  string   `json:"name"`
		Data  string   `json:"data"`
		Fee   int64    `json:"fee"`
	}

	CallTx struct {
		Input    *TxInput `json:"input"`
		Address  []byte   `json:"address"`
		GasLimit int64    `json:"gas_limit"`
		Fee      int64    `json:"fee"`
		Data     []byte   `json:"data"`
	}

	TxInput struct {
		Address   []byte           `json:"address"`   // Hash of the PubKey
		Amount    int64            `json:"amount"`    // Must not exceed account balance
		Sequence  int              `json:"sequence"`  // Must be 1 greater than the last committed TxInput
		Signature crypto.Signature `json:"signature"` // Depends on the PubKey type and the whole Tx
		PubKey    crypto.PubKey    `json:"pub_key"`   // Must not be nil, may be nil
	}

	TxOutput struct {
		Address []byte `json:"address"` // Hash of the PubKey
		Amount  int64  `json:"amount"`  // The sum of all outputs must not exceed the inputs.
	}
)

func (txIn *TxInput) ValidateBasic() error {
	if len(txIn.Address) != 20 {
		return ErrTxInvalidAddress
	}
	if txIn.Amount == 0 {
		return ErrTxInvalidAmount
	}
	return nil
}

func (txIn *TxInput) WriteSignBytes(w io.Writer, n *int, err *error) {
	wire.WriteTo([]byte(Fmt(`{"address":"%X","amount":%v,"sequence":%v}`, txIn.Address, txIn.Amount, txIn.Sequence)), w, n, err)
}

func (txIn *TxInput) String() string {
	return Fmt("TxInput{%X,%v,%v,%v,%v}", txIn.Address, txIn.Amount, txIn.Sequence, txIn.Signature, txIn.PubKey)
}

//-----------------------------------------------------------------------------

func (txOut *TxOutput) ValidateBasic() error {
	if len(txOut.Address) != 20 {
		return ErrTxInvalidAddress
	}
	if txOut.Amount == 0 {
		return ErrTxInvalidAmount
	}
	return nil
}

func (txOut *TxOutput) WriteSignBytes(w io.Writer, n *int, err *error) {
	wire.WriteTo([]byte(Fmt(`{"address":"%X","amount":%v}`, txOut.Address, txOut.Amount)), w, n, err)
}

func (txOut *TxOutput) String() string {
	return Fmt("TxOutput{%X,%v}", txOut.Address, txOut.Amount)
}

//-----------------------------------------------------------------------------

func (tx *SendTx) WriteSignBytes(chainID string, w io.Writer, n *int, err *error) {
	wire.WriteTo([]byte(Fmt(`{"chain_id":%s`, jsonEscape(chainID))), w, n, err)
	wire.WriteTo([]byte(Fmt(`,"tx":[%v,{"inputs":[`, TxTypeSend)), w, n, err)
	for i, in := range tx.Inputs {
		in.WriteSignBytes(w, n, err)
		if i != len(tx.Inputs)-1 {
			wire.WriteTo([]byte(","), w, n, err)
		}
	}
	wire.WriteTo([]byte(`],"outputs":[`), w, n, err)
	for i, out := range tx.Outputs {
		out.WriteSignBytes(w, n, err)
		if i != len(tx.Outputs)-1 {
			wire.WriteTo([]byte(","), w, n, err)
		}
	}
	wire.WriteTo([]byte(`]}]}`), w, n, err)
}

func (tx *SendTx) String() string {
	return Fmt("SendTx{%v -> %v}", tx.Inputs, tx.Outputs)
}

//-----------------------------------------------------------------------------

func (tx *CallTx) WriteSignBytes(chainID string, w io.Writer, n *int, err *error) {
	wire.WriteTo([]byte(Fmt(`{"chain_id":%s`, jsonEscape(chainID))), w, n, err)
	wire.WriteTo([]byte(Fmt(`,"tx":[%v,{"address":"%X","data":"%X"`, TxTypeCall, tx.Address, tx.Data)), w, n, err)
	wire.WriteTo([]byte(Fmt(`,"fee":%v,"gas_limit":%v,"input":`, tx.Fee, tx.GasLimit)), w, n, err)
	tx.Input.WriteSignBytes(w, n, err)
	wire.WriteTo([]byte(`}]}`), w, n, err)
}

func (tx *CallTx) String() string {
	return Fmt("CallTx{%v -> %x: %x}", tx.Input, tx.Address, tx.Data)
}

func NewContractAddress(caller []byte, nonce int) []byte {
	temp := make([]byte, 32+8)
	copy(temp, caller)
	PutInt64BE(temp[32:], int64(nonce))
	hasher := ripemd160.New()
	hasher.Write(temp) // does not error
	return hasher.Sum(nil)
}

//-----------------------------------------------------------------------------

func (tx *NameTx) WriteSignBytes(chainID string, w io.Writer, n *int, err *error) {
	wire.WriteTo([]byte(Fmt(`{"chain_id":%s`, jsonEscape(chainID))), w, n, err)
	wire.WriteTo([]byte(Fmt(`,"tx":[%v,{"data":%s,"fee":%v`, TxTypeName, jsonEscape(tx.Data), tx.Fee)), w, n, err)
	wire.WriteTo([]byte(`,"input":`), w, n, err)
	tx.Input.WriteSignBytes(w, n, err)
	wire.WriteTo([]byte(Fmt(`,"name":%s`, jsonEscape(tx.Name))), w, n, err)
	wire.WriteTo([]byte(`}]}`), w, n, err)
}

func (tx *NameTx) ValidateStrings() error {
	if len(tx.Name) == 0 {
		return ErrTxInvalidString{"Name must not be empty"}
	}
	if len(tx.Name) > MaxNameLength {
		return ErrTxInvalidString{Fmt("Name is too long. Max %d bytes", MaxNameLength)}
	}
	if len(tx.Data) > MaxDataLength {
		return ErrTxInvalidString{Fmt("Data is too long. Max %d bytes", MaxDataLength)}
	}

	if !validateNameRegEntryName(tx.Name) {
		return ErrTxInvalidString{Fmt("Invalid characters found in NameTx.Name (%s). Only alphanumeric, underscores, dashes, forward slashes, and @ are allowed", tx.Name)}
	}

	if !validateNameRegEntryData(tx.Data) {
		return ErrTxInvalidString{Fmt("Invalid characters found in NameTx.Data (%s). Only the kind of things found in a JSON file are allowed", tx.Data)}
	}

	return nil
}

func (tx *NameTx) String() string {
	return Fmt("NameTx{%v -> %s: %s}", tx.Input, tx.Name, tx.Data)
}

//-----------------------------------------------------------------------------

type BondTx struct {
	PubKey    crypto.PubKeyEd25519    `json:"pub_key"` // NOTE: these don't have type byte
	Signature crypto.SignatureEd25519 `json:"signature"`
	Inputs    []*TxInput              `json:"inputs"`
	UnbondTo  []*TxOutput             `json:"unbond_to"`
}

func (tx *BondTx) WriteSignBytes(chainID string, w io.Writer, n *int, err *error) {
	wire.WriteTo([]byte(Fmt(`{"chain_id":%s`, jsonEscape(chainID))), w, n, err)
	wire.WriteTo([]byte(Fmt(`,"tx":[%v,{"inputs":[`, TxTypeBond)), w, n, err)
	for i, in := range tx.Inputs {
		in.WriteSignBytes(w, n, err)
		if i != len(tx.Inputs)-1 {
			wire.WriteTo([]byte(","), w, n, err)
		}
	}
	wire.WriteTo([]byte(Fmt(`],"pub_key":`)), w, n, err)
	wire.WriteTo(wire.JSONBytes(tx.PubKey), w, n, err)
	wire.WriteTo([]byte(`,"unbond_to":[`), w, n, err)
	for i, out := range tx.UnbondTo {
		out.WriteSignBytes(w, n, err)
		if i != len(tx.UnbondTo)-1 {
			wire.WriteTo([]byte(","), w, n, err)
		}
	}
	wire.WriteTo([]byte(`]}]}`), w, n, err)
}

func (tx *BondTx) String() string {
	return Fmt("BondTx{%v: %v -> %v}", tx.PubKey, tx.Inputs, tx.UnbondTo)
}

//-----------------------------------------------------------------------------

type UnbondTx struct {
	Address   []byte                  `json:"address"`
	Height    int                     `json:"height"`
	Signature crypto.SignatureEd25519 `json:"signature"`
}

func (tx *UnbondTx) WriteSignBytes(chainID string, w io.Writer, n *int, err *error) {
	wire.WriteTo([]byte(Fmt(`{"chain_id":%s`, jsonEscape(chainID))), w, n, err)
	wire.WriteTo([]byte(Fmt(`,"tx":[%v,{"address":"%X","height":%v}]}`, TxTypeUnbond, tx.Address, tx.Height)), w, n, err)
}

func (tx *UnbondTx) String() string {
	return Fmt("UnbondTx{%X,%v,%v}", tx.Address, tx.Height, tx.Signature)
}

//-----------------------------------------------------------------------------

type RebondTx struct {
	Address   []byte                  `json:"address"`
	Height    int                     `json:"height"`
	Signature crypto.SignatureEd25519 `json:"signature"`
}

func (tx *RebondTx) WriteSignBytes(chainID string, w io.Writer, n *int, err *error) {
	wire.WriteTo([]byte(Fmt(`{"chain_id":%s`, jsonEscape(chainID))), w, n, err)
	wire.WriteTo([]byte(Fmt(`,"tx":[%v,{"address":"%X","height":%v}]}`, TxTypeRebond, tx.Address, tx.Height)), w, n, err)
}

func (tx *RebondTx) String() string {
	return Fmt("RebondTx{%X,%v,%v}", tx.Address, tx.Height, tx.Signature)
}

//-----------------------------------------------------------------------------

type DupeoutTx struct {
	Address []byte                `json:"address"`
	VoteA   tendermint_types.Vote `json:"vote_a"`
	VoteB   tendermint_types.Vote `json:"vote_b"`
}

func (tx *DupeoutTx) WriteSignBytes(chainID string, w io.Writer, n *int, err *error) {
	//PanicSanity("DupeoutTx has no sign bytes")
	// TODO
	return
}

func (tx *DupeoutTx) String() string {
	return Fmt("DupeoutTx{%X,%v,%v}", tx.Address, tx.VoteA, tx.VoteB)
}

//-----------------------------------------------------------------------------

type PermissionsTx struct {
	Input    *TxInput        `json:"input"`
	PermArgs ptypes.PermArgs `json:"args"`
}

func (tx *PermissionsTx) WriteSignBytes(chainID string, w io.Writer, n *int, err *error) {
	wire.WriteTo([]byte(Fmt(`{"chain_id":%s`, jsonEscape(chainID))), w, n, err)
	wire.WriteTo([]byte(Fmt(`,"tx":[%v,{"args":"`, TxTypePermissions)), w, n, err)
	wire.WriteJSON(&tx.PermArgs, w, n, err)
	wire.WriteTo([]byte(`","input":`), w, n, err)
	tx.Input.WriteSignBytes(w, n, err)
	wire.WriteTo([]byte(`}]}`), w, n, err)
}

func (tx *PermissionsTx) String() string {
	return Fmt("PermissionsTx{%v -> %v}", tx.Input, tx.PermArgs)
}

//-----------------------------------------------------------------------------

func TxHash(chainID string, tx Tx) []byte {
	signBytes := acm.SignBytes(chainID, tx)
	hasher := ripemd160.New()
	hasher.Write(signBytes)
	// Calling Sum(nil) just gives us the digest with nothing prefixed
	return hasher.Sum(nil)
}

//-----------------------------------------------------------------------------

func EncodeTx(tx Tx) ([]byte, error) {
	var n int
	var err error
	buf := new(bytes.Buffer)
	wire.WriteBinary(struct{ Tx }{tx}, buf, &n, &err)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// panic on err
func DecodeTx(txBytes []byte) (Tx, error) {
	var n int
	var err error
	tx := new(Tx)
	buf := bytes.NewBuffer(txBytes)
	wire.ReadBinaryPtr(tx, buf, len(txBytes), &n, &err)
	if err != nil {
		return nil, err
	}
	return *tx, nil
}

func GenerateReceipt(chainId string, tx Tx) Receipt {
	receipt := Receipt{
		TxHash:          TxHash(chainId, tx),
		CreatesContract: 0,
		ContractAddr:    nil,
	}
	if callTx, ok := tx.(*CallTx); ok {
		if len(callTx.Address) == 0 {
			receipt.CreatesContract = 1
			receipt.ContractAddr = NewContractAddress(callTx.Input.Address,
				callTx.Input.Sequence)
		}
	}
	return receipt
}

//--------------------------------------------------------------------------------

// Contract: This function is deterministic and completely reversible.
func jsonEscape(str string) string {
	escapedBytes, err := json.Marshal(str)
	if err != nil {
		PanicSanity(Fmt("Error json-escaping a string", str))
	}
	return string(escapedBytes)
}
