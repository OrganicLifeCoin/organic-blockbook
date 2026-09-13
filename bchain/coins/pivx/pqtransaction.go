package pivx

import (
	"blockbook/bchain"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"math"
	"math/big"

	"github.com/martinboehm/btcd/chaincfg/chainhash"
)

const (
	pqTransactionVersion = int16(3)
	pqTransactionType    = int16(8)
	pqMaxTransactionSize = 24000
	pqMaxInputs          = 2
	pqMaxOutputs         = 3
)

type pqReader struct {
	*bytes.Reader
}

func (r pqReader) byte() (byte, error) {
	value, err := r.ReadByte()
	if err != nil {
		return 0, errors.New("truncated PQ transaction")
	}
	return value, nil
}

func (r pqReader) uint32() (uint32, error) {
	var value uint32
	if err := binary.Read(r, binary.LittleEndian, &value); err != nil {
		return 0, errors.New("truncated PQ transaction")
	}
	return value, nil
}

func (r pqReader) int64() (int64, error) {
	var value int64
	if err := binary.Read(r, binary.LittleEndian, &value); err != nil {
		return 0, errors.New("truncated PQ transaction")
	}
	return value, nil
}

func (r pqReader) compactSize(limit uint64) (uint64, error) {
	first, err := r.byte()
	if err != nil {
		return 0, err
	}
	var value uint64
	switch first {
	case 0xfd:
		var n uint16
		if err := binary.Read(r, binary.LittleEndian, &n); err != nil || n < 0xfd {
			return 0, errors.New("invalid PQ CompactSize")
		}
		value = uint64(n)
	case 0xfe:
		var n uint32
		if err := binary.Read(r, binary.LittleEndian, &n); err != nil || n <= math.MaxUint16 {
			return 0, errors.New("invalid PQ CompactSize")
		}
		value = uint64(n)
	case 0xff:
		if err := binary.Read(r, binary.LittleEndian, &value); err != nil || value <= math.MaxUint32 {
			return 0, errors.New("invalid PQ CompactSize")
		}
	default:
		value = uint64(first)
	}
	if value > limit {
		return 0, errors.New("PQ CompactSize exceeds limit")
	}
	return value, nil
}

func (r pqReader) bytes(limit uint64) ([]byte, error) {
	length, err := r.compactSize(limit)
	if err != nil {
		return nil, err
	}
	value := make([]byte, int(length))
	if _, err := io.ReadFull(r, value); err != nil {
		return nil, errors.New("truncated PQ transaction")
	}
	return value, nil
}

func isNullHash(hash []byte) bool {
	for _, value := range hash {
		if value != 0 {
			return false
		}
	}
	return true
}

func (p *PivXParser) parsePQTransaction(raw []byte) (*bchain.Tx, error) {
	if len(raw) > pqMaxTransactionSize {
		return nil, errors.New("PQ transaction exceeds size limit")
	}
	r := pqReader{bytes.NewReader(raw)}
	var version, transactionType int16
	if err := binary.Read(r, binary.LittleEndian, &version); err != nil {
		return nil, errors.New("truncated PQ transaction")
	}
	if err := binary.Read(r, binary.LittleEndian, &transactionType); err != nil {
		return nil, errors.New("truncated PQ transaction")
	}
	if version != pqTransactionVersion || transactionType != pqTransactionType {
		return nil, errors.New("not an OLC PQ transaction")
	}
	inputCount, err := r.compactSize(pqMaxInputs)
	if err != nil {
		return nil, err
	}
	type parsedInput struct {
		hash, script   []byte
		vout, sequence uint32
	}
	inputs := make([]parsedInput, int(inputCount))
	for i := range inputs {
		inputs[i].hash = make([]byte, 32)
		if _, err := io.ReadFull(r, inputs[i].hash); err != nil {
			return nil, errors.New("truncated PQ transaction")
		}
		if inputs[i].vout, err = r.uint32(); err != nil {
			return nil, err
		}
		if inputs[i].script, err = r.bytes(pqMaxTransactionSize); err != nil {
			return nil, err
		}
		if inputs[i].sequence, err = r.uint32(); err != nil {
			return nil, err
		}
	}
	outputCount, err := r.compactSize(pqMaxOutputs)
	if err != nil {
		return nil, err
	}
	outputs := make([]bchain.Vout, int(outputCount))
	for i := range outputs {
		value, err := r.int64()
		if err != nil {
			return nil, err
		}
		script, err := r.bytes(pqMaxTransactionSize)
		if err != nil {
			return nil, err
		}
		addresses, _, _ := p.OutputScriptToAddressesFunc(script)
		outputs[i] = bchain.Vout{
			ValueSat:     *big.NewInt(value),
			N:            uint32(i),
			ScriptPubKey: bchain.ScriptPubKey{Hex: hex.EncodeToString(script), Addresses: addresses},
		}
	}
	lockTime, err := r.uint32()
	if err != nil {
		return nil, err
	}
	saplingMarker, err := r.byte()
	if err != nil || saplingMarker != 0 {
		return nil, errors.New("invalid PQ Sapling marker")
	}
	extraPayloadMarker, err := r.byte()
	if err != nil || extraPayloadMarker != 1 {
		return nil, errors.New("missing PQ extra payload")
	}
	payload, err := r.bytes(pqMaxTransactionSize)
	if err != nil {
		return nil, err
	}
	if len(payload) < 3 || payload[0] != 1 || payload[2] > pqMaxInputs || r.Len() != 0 {
		return nil, errors.New("invalid PQ extra payload")
	}
	coinbase := len(inputs) == 1 && isNullHash(inputs[0].hash) && inputs[0].vout == math.MaxUint32
	vin := make([]bchain.Vin, len(inputs))
	for i, input := range inputs {
		if coinbase {
			vin[i] = bchain.Vin{Coinbase: hex.EncodeToString(input.script), Sequence: input.sequence}
			continue
		}
		var hash chainhash.Hash
		copy(hash[:], input.hash)
		vin[i] = bchain.Vin{
			Txid: hash.String(), Vout: input.vout, Sequence: input.sequence,
			ScriptSig: bchain.ScriptSig{Hex: hex.EncodeToString(input.script)},
		}
	}
	txid := chainhash.DoubleHashH(raw).String()
	return &bchain.Tx{
		Hex: hex.EncodeToString(raw), Txid: txid, Version: int32(version),
		LockTime: lockTime, Vin: vin, Vout: outputs,
	}, nil
}

func isPQTransaction(raw []byte) bool {
	return len(raw) >= 4 && int16(binary.LittleEndian.Uint16(raw[:2])) == pqTransactionVersion &&
		int16(binary.LittleEndian.Uint16(raw[2:4])) == pqTransactionType
}
