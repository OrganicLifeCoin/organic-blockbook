package pivx

import (
	"errors"
	"strings"
)

const bech32mCharset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"
const bech32mConstant uint32 = 0x2bc830a3

func bech32mPolymod(values []byte) uint32 {
	checksum := uint32(1)
	for _, value := range values {
		top := checksum >> 25
		checksum = (checksum&0x1ffffff)<<5 ^ uint32(value)
		for i, generator := range [...]uint32{0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3} {
			if top&(1<<i) != 0 {
				checksum ^= generator
			}
		}
	}
	return checksum
}

func bech32mExpandHRP(hrp string) []byte {
	values := make([]byte, len(hrp)*2+1)
	for i := range hrp {
		values[i] = hrp[i] >> 5
		values[i+len(hrp)+1] = hrp[i] & 31
	}
	return values
}

func encodeBech32m(hrp string, data []byte) (string, error) {
	values := append(bech32mExpandHRP(hrp), data...)
	values = append(values, make([]byte, 6)...)
	checksum := bech32mPolymod(values) ^ bech32mConstant
	var result strings.Builder
	result.WriteString(hrp)
	result.WriteByte('1')
	for _, value := range data {
		if value >= 32 {
			return "", errors.New("invalid Bech32m data")
		}
		result.WriteByte(bech32mCharset[value])
	}
	for i := 0; i < 6; i++ {
		result.WriteByte(bech32mCharset[(checksum>>uint(5*(5-i)))&31])
	}
	return result.String(), nil
}

func decodeBech32m(value string) (string, []byte, error) {
	if len(value) > 1023 {
		return "", nil, errors.New("invalid Bech32m length")
	}
	lower, upper := false, false
	for i := range value {
		if value[i] < 33 || value[i] > 126 {
			return "", nil, errors.New("invalid Bech32m character")
		}
		lower = lower || value[i] >= 'a' && value[i] <= 'z'
		upper = upper || value[i] >= 'A' && value[i] <= 'Z'
	}
	if lower && upper {
		return "", nil, errors.New("mixed-case Bech32m string")
	}
	value = strings.ToLower(value)
	separator := strings.LastIndexByte(value, '1')
	if separator <= 0 || separator+7 > len(value) {
		return "", nil, errors.New("invalid Bech32m separator")
	}
	hr := value[:separator]
	data := make([]byte, len(value)-separator-1)
	for i := range data {
		index := strings.IndexByte(bech32mCharset, value[separator+1+i])
		if index < 0 {
			return "", nil, errors.New("invalid Bech32m data")
		}
		data[i] = byte(index)
	}
	if bech32mPolymod(append(bech32mExpandHRP(hr), data...)) != bech32mConstant {
		return "", nil, errors.New("invalid Bech32m checksum")
	}
	return hr, data[:len(data)-6], nil
}
