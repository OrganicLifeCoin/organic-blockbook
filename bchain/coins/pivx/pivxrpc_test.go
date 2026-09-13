//go:build unittest
// +build unittest

package pivx

import (
	"blockbook/bchain"
	"blockbook/bchain/coins/btc"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetChainInfoUsesCurrentOLCSupplyRPC(t *testing.T) {
	t.Helper()

	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		methods = append(methods, request.Method)
		w.Header().Set("Content-Type", "application/json")

		switch request.Method {
		case "getsupplyinfo":
			if len(request.Params) != 1 || string(request.Params[0]) != "false" {
				t.Fatalf("getsupplyinfo params = %s, want [false]", request.Params)
			}
			fmt.Fprint(w, `{"result":{"transparentsupply":123.5,"shieldsupply":0,"totalsupply":123.5},"error":null}`)
		case "listpqmasternodes":
			fmt.Fprint(w, `{"result":[{},{}],"error":null}`)
		default:
			fmt.Fprintf(w, `{"result":null,"error":{"code":-32601,"message":"Method not found: %s"}}`, request.Method)
		}
	}))
	defer server.Close()

	config := json.RawMessage(fmt.Sprintf(`{"rpc_url":%q,"rpc_user":"user","rpc_pass":"password","rpc_timeout":5}`, server.URL))
	chain, err := NewPivXRPC(config, nil)
	if err != nil {
		t.Fatalf("NewPivXRPC: %v", err)
	}
	rpc := chain.(*PivXRPC)
	rpc.BitcoinGetChainInfo = func() (*bchain.ChainInfo, error) {
		return &bchain.ChainInfo{Chain: "test", Headers: 10647}, nil
	}

	info, err := rpc.GetChainInfo()
	if err != nil {
		t.Fatalf("GetChainInfo: %v", err)
	}
	if got, want := fmt.Sprint(methods), "[getsupplyinfo listpqmasternodes]"; got != want {
		t.Fatalf("RPC methods = %s, want %s", got, want)
	}
	if info.TransparentSupply.String() != "123.5" || info.ShieldSupply.String() != "0" || info.MoneySupply.String() != "123.5" {
		t.Fatalf("unexpected supply values: transparent=%s shield=%s total=%s", info.TransparentSupply, info.ShieldSupply, info.MoneySupply)
	}
	if info.MasternodeCount != 2 {
		t.Fatalf("masternode count = %d, want 2", info.MasternodeCount)
	}
}

func TestPQMasternodeCount(t *testing.T) {
	count, err := pqMasternodeCount([]json.RawMessage{{}, {}}, nil)
	if err != nil || count != 2 {
		t.Fatalf("count = %d, err = %v", count, err)
	}
	count, err = pqMasternodeCount(nil, &bchain.RPCError{Code: -1, Message: "PQ masternodes are not active on this network"})
	if err != nil || count != 0 {
		t.Fatalf("inactive count = %d, err = %v", count, err)
	}
	if _, err = pqMasternodeCount(nil, &bchain.RPCError{Code: -1, Message: "stale registry"}); err == nil {
		t.Fatal("unexpected registry errors must fail")
	}
}

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
