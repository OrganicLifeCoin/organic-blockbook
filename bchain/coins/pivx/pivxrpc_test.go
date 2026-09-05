//go:build unittest
// +build unittest

package pivx

import (
	"blockbook/bchain"
	"blockbook/bchain/coins/btc"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestGetNextSuperBlock(t *testing.T) {
	tests := []struct {
		name    string
		testnet bool
		height  int
		want    int
	}{
		{name: "mainnet genesis", height: 0, want: 10080},
		{name: "mainnet before boundary", height: 10079, want: 10080},
		{name: "mainnet at boundary", height: 10080, want: 20160},
		{name: "testnet genesis", testnet: true, height: 0, want: 40320},
		{name: "testnet before boundary", testnet: true, height: 40319, want: 40320},
		{name: "testnet at boundary", testnet: true, height: 40320, want: 80640},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rpc := &PivXRPC{BitcoinRPC: &btc.BitcoinRPC{BaseChain: &bchain.BaseChain{Testnet: tt.testnet}}}
			if got := rpc.GetNextSuperBlock(tt.height); got != tt.want {
				t.Fatalf("GetNextSuperBlock(%d) = %d, want %d", tt.height, got, tt.want)
			}
		})
	}
}

func TestGetChainInfoUsesPQOnlyRPCs(t *testing.T) {
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		methods = append(methods, request.Method)
		var result any
		switch request.Method {
		case "getblockchaininfo":
			result = map[string]any{"chain": "test", "blocks": 0, "headers": 0, "bestblockhash": "genesis", "difficulty": 1, "size_on_disk": 0, "warnings": ""}
		case "getnetworkinfo":
			result = map[string]any{"version": 1010400, "subversion": "/OrganicLife:1.1.4/", "protocolversion": 70929, "timeoffset": 0, "warnings": ""}
		case "getsupplyinfo":
			result = map[string]any{"transparentsupply": 12.5, "shieldsupply": 0, "totalsupply": 12.5}
		default:
			json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": -32601, "message": "Method not found"}})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"result": result})
	}))
	defer server.Close()

	config, _ := json.Marshal(map[string]any{"rpc_url": server.URL, "rpc_timeout": 5})
	chain, err := NewPivXRPC(config, nil)
	if err != nil {
		t.Fatal(err)
	}
	rpc := chain.(*PivXRPC)
	rpc.Testnet = true
	info, err := rpc.GetChainInfo()
	if err != nil {
		t.Fatal(err)
	}
	if info.TransparentSupply != "12.5" || info.ShieldSupply != "0" || info.MoneySupply != "12.5" {
		t.Fatalf("unexpected supply: %+v", info)
	}
	if info.MasternodeCount != 0 || info.NextSuperBlock != 40320 {
		t.Fatalf("unexpected PQ-only chain metadata: %+v", info)
	}
	want := []string{"getblockchaininfo", "getnetworkinfo", "getsupplyinfo"}
	if !reflect.DeepEqual(methods, want) {
		t.Fatalf("RPC methods = %v, want %v", methods, want)
	}
}
