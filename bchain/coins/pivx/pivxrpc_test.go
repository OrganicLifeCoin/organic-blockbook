//go:build unittest
// +build unittest

package pivx

import (
	"blockbook/bchain"
	"blockbook/bchain/coins/btc"
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
