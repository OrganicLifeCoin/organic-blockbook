package bchain

import (
	"encoding/json"
	"testing"
)

func TestChainInfoMasternodeFields(t *testing.T) {
	encoded, err := json.Marshal(ChainInfo{
		MasternodeCount: 42,
		NextSuperBlock:  9001,
	})
	if err != nil {
		t.Fatal(err)
	}

	var fields map[string]interface{}
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if fields["masternodecount"] != float64(42) {
		t.Errorf("masternodecount = %v, want 42", fields["masternodecount"])
	}
	if fields["nextsuperblock"] != float64(9001) {
		t.Errorf("nextsuperblock = %v, want 9001", fields["nextsuperblock"])
	}
}
