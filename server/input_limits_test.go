package server

import (
	"strings"
	"testing"
)

func TestReadSignedTransactionBodyLimitsInput(t *testing.T) {
	valid := strings.Repeat("a", maxSignedTransactionHexBytes)
	got, err := readSignedTransactionBody(strings.NewReader(valid))
	if err != nil {
		t.Fatalf("readSignedTransactionBody(valid) error = %v", err)
	}
	if got != valid {
		t.Fatal("readSignedTransactionBody(valid) changed the body")
	}

	tooLarge := strings.Repeat("b", maxSignedTransactionHexBytes+1)
	if _, err := readSignedTransactionBody(strings.NewReader(tooLarge)); err == nil {
		t.Fatal("readSignedTransactionBody accepted an oversized body")
	}
}

func TestValidateWebsocketAddressCount(t *testing.T) {
	if err := validateWebsocketAddressCount(maxWebsocketAddressSubscriptions); err != nil {
		t.Fatalf("validateWebsocketAddressCount(valid) error = %v", err)
	}
	if err := validateWebsocketAddressCount(maxWebsocketAddressSubscriptions + 1); err == nil {
		t.Fatal("validateWebsocketAddressCount accepted too many addresses")
	}
}

func TestValidateSignedTransactionHexLength(t *testing.T) {
	if err := validateSignedTransactionHex(strings.Repeat("c", maxSignedTransactionHexBytes)); err != nil {
		t.Fatalf("validateSignedTransactionHex(valid) error = %v", err)
	}
	if err := validateSignedTransactionHex(strings.Repeat("d", maxSignedTransactionHexBytes+1)); err == nil {
		t.Fatal("validateSignedTransactionHex accepted an oversized transaction")
	}
}

func TestNormalizeAddressPaging(t *testing.T) {
	page, pageSize := normalizeAddressPaging(0, 0, 25, 1000)
	if page != 1 || pageSize != 25 {
		t.Fatalf("normalizeAddressPaging(default) = (%d, %d), want (1, 25)", page, pageSize)
	}

	page, pageSize = normalizeAddressPaging(int(^uint(0)>>1), -1, 25, 1000)
	if page != maxAddressHistoryResults/25 || pageSize != 25 {
		t.Fatalf("normalizeAddressPaging(unsafe) = (%d, %d)", page, pageSize)
	}
}

func TestValidateWebsocketFeeTargets(t *testing.T) {
	if err := validateWebsocketFeeTargets([]int{1, 6, 24}); err != nil {
		t.Fatalf("validateWebsocketFeeTargets(valid) error = %v", err)
	}
	if err := validateWebsocketFeeTargets(make([]int, maxWebsocketFeeTargets+1)); err == nil {
		t.Fatal("validateWebsocketFeeTargets accepted too many targets")
	}
	if err := validateWebsocketFeeTargets([]int{0}); err == nil {
		t.Fatal("validateWebsocketFeeTargets accepted an invalid target")
	}
}

func TestValidateWebsocketRequestMetadata(t *testing.T) {
	if err := validateWebsocketRequestMetadata("request-1", "getInfo"); err != nil {
		t.Fatalf("validateWebsocketRequestMetadata(valid) error = %v", err)
	}
	if err := validateWebsocketRequestMetadata(strings.Repeat("i", maxWebsocketRequestIDBytes+1), "getInfo"); err == nil {
		t.Fatal("validateWebsocketRequestMetadata accepted an oversized id")
	}
	if err := validateWebsocketRequestMetadata("request-1", strings.Repeat("m", maxWebsocketMethodBytes+1)); err == nil {
		t.Fatal("validateWebsocketRequestMetadata accepted an oversized method")
	}
}
