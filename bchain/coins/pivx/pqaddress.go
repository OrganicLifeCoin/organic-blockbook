package pivx

import (
	"errors"
	"strings"
)

const (
	pqBech32mConstant = uint32(0x2bc830a3)
	pqCharset         = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"
)

func (p *PivXParser) pqHRP() string {
	if p.Params.Net == TestnetMagic {
		return "olcpqtest"
	}
	return "olcpq"
}

func isPQScript(script []byte) bool {
	return len(script) == 35 && script[0] == 0xff && script[1] == 0x51 && script[2] == 0x20
}

func pqPolymod(values []byte) uint32 {
	const generators = 5
	generator := [generators]uint32{0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3}
	checksum := uint32(1)
	for _, value := range values {
		top := checksum >> 25
		checksum = (checksum&0x1ffffff)<<5 ^ uint32(value)
		for i := 0; i < generators; i++ {
			if (top>>i)&1 != 0 {
				checksum ^= generator[i]
			}
		}
	}
	return checksum
}

func pqHRPExpand(hrp string) []byte {
	values := make([]byte, 0, len(hrp)*2+1)
	for i := range hrp {
		values = append(values, hrp[i]>>5)
	}
	values = append(values, 0)
	for i := range hrp {
		values = append(values, hrp[i]&31)
	}
	return values
}

func pqConvertBits(data []byte, fromBits, toBits uint, pad bool) ([]byte, error) {
	var accumulator uint32
	var bits uint
	maxValue := uint32(1<<toBits) - 1
	maxAccumulator := uint32(1<<(fromBits+toBits-1)) - 1
	result := make([]byte, 0, (len(data)*int(fromBits)+int(toBits)-1)/int(toBits))
	for _, value := range data {
		if uint32(value)>>fromBits != 0 {
			return nil, errors.New("invalid PQ address data")
		}
		accumulator = ((accumulator << fromBits) | uint32(value)) & maxAccumulator
		bits += fromBits
		for bits >= toBits {
			bits -= toBits
			result = append(result, byte((accumulator>>bits)&maxValue))
		}
	}
	if pad {
		if bits > 0 {
			result = append(result, byte((accumulator<<(toBits-bits))&maxValue))
		}
	} else if bits >= fromBits || ((accumulator<<(toBits-bits))&maxValue) != 0 {
		return nil, errors.New("invalid PQ address padding")
	}
	return result, nil
}

func pqAddressFromScript(script []byte, hrp string) (string, bool) {
	if !isPQScript(script) {
		return "", false
	}
	converted, err := pqConvertBits(script[3:], 8, 5, true)
	if err != nil {
		return "", false
	}
	data := append([]byte{1}, converted...)
	values := append(pqHRPExpand(hrp), data...)
	values = append(values, make([]byte, 6)...)
	polymod := pqPolymod(values) ^ pqBech32mConstant
	var address strings.Builder
	address.Grow(len(hrp) + 1 + len(data) + 6)
	address.WriteString(hrp)
	address.WriteByte('1')
	for _, value := range data {
		address.WriteByte(pqCharset[value])
	}
	for i := 0; i < 6; i++ {
		shift := uint(5 * (5 - i))
		address.WriteByte(pqCharset[(polymod>>shift)&31])
	}
	return address.String(), true
}

func pqScriptFromAddress(address, hrp string) ([]byte, error) {
	if address != strings.ToLower(address) || len(address) != len(hrp)+1+53+6 ||
		!strings.HasPrefix(address, hrp+"1") {
		return nil, errors.New("invalid PQ address")
	}
	encoded := address[len(hrp)+1:]
	values := make([]byte, len(encoded))
	for i := range encoded {
		index := strings.IndexByte(pqCharset, encoded[i])
		if index < 0 {
			return nil, errors.New("invalid PQ address")
		}
		values[i] = byte(index)
	}
	checksumInput := append(pqHRPExpand(hrp), values...)
	if pqPolymod(checksumInput) != pqBech32mConstant {
		return nil, errors.New("invalid PQ address checksum")
	}
	payload := values[:len(values)-6]
	if len(payload) != 53 || payload[0] != 1 {
		return nil, errors.New("invalid PQ address payload")
	}
	id, err := pqConvertBits(payload[1:], 5, 8, false)
	if err != nil || len(id) != 32 {
		return nil, errors.New("invalid PQ address payload")
	}
	script := append([]byte{0xff, 0x51, 0x20}, id...)
	canonical, ok := pqAddressFromScript(script, hrp)
	if !ok || canonical != address {
		return nil, errors.New("invalid PQ address")
	}
	return script, nil
}
