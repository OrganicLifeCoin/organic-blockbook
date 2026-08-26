//go:build unittest
// +build unittest

package coins

import (
	"reflect"
	"sort"
	"testing"
)

func TestOrganicLifeCoinFactoriesAreTheOnlyPublicFactories(t *testing.T) {
	got := make([]string, 0, len(BlockChainFactories))
	for name := range BlockChainFactories {
		got = append(got, name)
	}
	sort.Strings(got)
	want := []string{"OrganicLifeCoin", "OrganicLifeCoin Testnet"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("registered factories = %v, want %v", got, want)
	}
}
