//go:build unittest

package pivx

import (
	"blockbook/bchain/coins/btc"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

// Golden address from Core pqkey_tests; raw transaction and txid independently
// decoded by OrganicLifeCoin 1.1.3 decoderawtransaction (synthetic FUND).
const pqAddress = "olcpqtest1ppf0kwsev98fsffnp9hpe0chp509e7mvpgpzyd5erfslqtgsxkrls25pmyv"
const legacyPQAddress = "olcpqtest1ppf0kwsev98fsffnp9hpe0chp509e7mvpgpzyd5erfslqtgsxkrlslg3hpw"
const pqScript = "ff51200a5f67432c29d304a6612dc397e2e1a3cb9f6d81404446d3234c3e05a206b0ff"
const pqRaw = "030008000101010101010101010101010101010101010101010101010101010101010101010000000000ffffffff0100e1f5050000000023ff51200a5f67432c29d304a6612dc397e2e1a3cb9f6d81404446d3234c3e05a206b0ff00000000000103010000"
const pqTxID = "c8c6d4c604e38027188b7d63daf3d1b721dfeef11d2b11f711f44d971cb8d275"

func TestBech32mChecksumVariant(t *testing.T) {
	hr, data, err := decodeBech32m("A1LQFN3A")
	if err != nil || hr != "a" || len(data) != 0 {
		t.Fatalf("Bech32m vector failed: %q %x %v", hr, data, err)
	}
	encoded, err := encodeBech32m(hr, data)
	if err != nil || encoded != "a1lqfn3a" {
		t.Fatalf("Bech32m encoding failed: %q %v", encoded, err)
	}
	if _, _, err := decodeBech32m("A12UEL5L"); err == nil {
		t.Fatal("accepted legacy Bech32 checksum as Bech32m")
	}
}

func TestPQAddressRoundTrip(t *testing.T) {
	p := NewPivXParser(GetChainParams("test"), &btc.Configuration{})
	if uint32(p.Params.Net) != 0x1102c46b {
		t.Errorf("old testnet identity: %x", p.Params.Net)
	}
	script, _ := hex.DecodeString(pqScript)
	desc, err := p.GetAddrDescFromAddress(pqAddress)
	if err != nil || !reflect.DeepEqual([]byte(desc), script) {
		t.Errorf("address decode: %x, %v", desc, err)
	}
	addresses, searchable, err := p.GetAddressesFromAddrDesc(script)
	if err != nil || !searchable || !reflect.DeepEqual(addresses, []string{pqAddress}) {
		t.Errorf("script decode: %v %v %v", addresses, searchable, err)
	}
	if !p.IsAddrDescIndexable(script) {
		t.Error("PQ output not indexed")
	}
	for _, address := range []string{strings.ToUpper(pqAddress), pqAddress[:68] + "q", legacyPQAddress, "olcpqregtest1ptg5w9rz7lg3wfm0kpkqcucf8k7egr24c68pve32tqklksmw08h8syk3x80", pqAddress + "q"} {
		if _, err := p.GetAddrDescFromAddress(address); err == nil {
			t.Errorf("accepted invalid address %s", address)
		}
	}
	_, data, _ := decodeBech32m(pqAddress)
	for _, invalid := range []struct {
		name   string
		change func([]byte)
	}{
		{"version", func(d []byte) { d[0] = 2 }},
		{"padding", func(d []byte) { d[len(d)-1] |= 1 }},
	} {
		d := append([]byte(nil), data...)
		invalid.change(d)
		address, _ := encodeBech32m("olcpqtest", d)
		if _, err := p.GetAddrDescFromAddress(address); err == nil {
			t.Errorf("accepted invalid %s", invalid.name)
		}
	}
	main := NewPivXParser(GetChainParams("main"), &btc.Configuration{})
	if _, err := main.GetAddrDescFromAddress(pqAddress); err == nil {
		t.Error("mainnet accepted PQ address")
	}
	addresses, _, _ = main.GetAddressesFromAddrDesc(script)
	if len(addresses) != 0 {
		t.Errorf("mainnet exposed PQ address: %v", addresses)
	}
}

func TestPQMempoolUsesCoreTransaction(t *testing.T) {
	p := NewPivXParser(GetChainParams("test"), &btc.Configuration{})
	legacy, err := hex.DecodeString(testTx1.Hex)
	if err != nil {
		t.Fatal(err)
	}
	ordinary, err := p.ParseTx(legacy)
	if err != nil || ordinary.Txid != testTx1.Txid {
		t.Fatalf("ordinary raw transaction regressed: %v %v", ordinary, err)
	}
	raw, _ := hex.DecodeString(pqRaw)
	if tx, err := p.ParseTx(raw); err == nil {
		t.Errorf("legacy parser silently truncated special tx: %+v", tx)
	}
	var invalidAmount atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		if req.Method != "getrawtransaction" || len(req.Params) != 2 || string(req.Params[1]) != "1" {
			t.Errorf("not a verbose transaction request: %+v", req)
			json.NewEncoder(w).Encode(map[string]any{"result": pqRaw})
			return
		}
		value := json.Number("1")
		if invalidAmount.Load() {
			value = "1e9999"
		}
		json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{
			"txid": pqTxID, "version": 3, "type": 8, "hex": pqRaw,
			"vin": []any{}, "vout": []any{map[string]any{"value": value, "n": 0, "scriptPubKey": map[string]any{"hex": pqScript}}},
		}})
	}))
	defer server.Close()
	config, _ := json.Marshal(map[string]any{"rpc_url": server.URL, "rpc_timeout": 5})
	chain, err := NewPivXRPC(config, nil)
	if err != nil {
		t.Fatal(err)
	}
	rpc := chain.(*PivXRPC)
	rpc.Parser = p
	tx, err := rpc.GetTransactionForMempool(pqTxID)
	if err != nil {
		t.Fatal(err)
	}
	if tx.Txid != pqTxID || tx.Version != 3 || tx.Hex != pqRaw {
		t.Errorf("lost Core transaction identity: %+v", tx)
	}
	if len(tx.Vout) != 1 || tx.Vout[0].ValueSat.Int64() != 100000000 || !reflect.DeepEqual(tx.Vout[0].ScriptPubKey.Addresses, []string{pqAddress}) {
		t.Errorf("lost PQ output: %+v", tx.Vout)
	}
	invalidAmount.Store(true)
	if _, err := rpc.GetTransactionForMempool(pqTxID); err == nil {
		t.Error("accepted unrepresentable amount")
	}
}
