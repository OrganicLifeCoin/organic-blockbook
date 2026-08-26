package server

import (
	"errors"
	"io"
	"io/ioutil"
	"math"
)

const (
	maxSignedTransactionHexBytes     = 4 << 20
	maxSignedTransactionFormBytes    = maxSignedTransactionHexBytes + (1 << 10)
	maxWebsocketMessageBytes         = maxSignedTransactionHexBytes
	maxWebsocketAddressSubscriptions = 1000
	maxWebsocketFeeTargets           = 20
	maxFeeTargetBlocks               = 1008
	maxWebsocketRequestIDBytes       = 128
	maxWebsocketMethodBytes          = 64
	maxWebsocketPageSize             = 100
	maxAddressHistoryResults         = 100000
)

var (
	errSignedTransactionTooLarge  = errors.New("signed transaction exceeds the 4 MiB limit")
	errTooManyWebsocketAddresses  = errors.New("address subscription exceeds the 1000-address limit")
	errTooManyFeeTargets          = errors.New("fee estimate exceeds the 20-target limit")
	errInvalidFeeTarget           = errors.New("fee target must be between 1 and 1008 blocks")
	errWebsocketRequestIDTooLarge = errors.New("websocket request id exceeds the 128-byte limit")
	errWebsocketMethodTooLarge    = errors.New("websocket method exceeds the 64-byte limit")
)

func validateSignedTransactionHex(tx string) error {
	if len(tx) > maxSignedTransactionHexBytes {
		return errSignedTransactionTooLarge
	}
	return nil
}

func readSignedTransactionBody(r io.Reader) (string, error) {
	b, err := ioutil.ReadAll(io.LimitReader(r, maxSignedTransactionHexBytes+1))
	if err != nil {
		return "", err
	}
	tx := string(b)
	if err := validateSignedTransactionHex(tx); err != nil {
		return "", err
	}
	return tx, nil
}

func validateWebsocketAddressCount(count int) error {
	if count > maxWebsocketAddressSubscriptions {
		return errTooManyWebsocketAddresses
	}
	return nil
}

func validateWebsocketFeeTargets(targets []int) error {
	if len(targets) > maxWebsocketFeeTargets {
		return errTooManyFeeTargets
	}
	for _, target := range targets {
		if target < 1 || target > maxFeeTargetBlocks {
			return errInvalidFeeTarget
		}
	}
	return nil
}

func validateWebsocketRequestMetadata(id, method string) error {
	if len(id) > maxWebsocketRequestIDBytes {
		return errWebsocketRequestIDTooLarge
	}
	if len(method) > maxWebsocketMethodBytes {
		return errWebsocketMethodTooLarge
	}
	return nil
}

func normalizeAddressPaging(page, pageSize, defaultPageSize, maxPageSize int) (int, int) {
	if pageSize <= 0 {
		pageSize = defaultPageSize
	} else if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	if page < 1 {
		page = 1
	}
	maxPage := maxAddressHistoryResults / pageSize
	if maxPage < 1 {
		maxPage = 1
	}
	if page > maxPage {
		page = maxPage
	}
	return page, pageSize
}

func normalizeBlockHeight(height int) uint32 {
	if height <= 0 {
		return 0
	}
	if uint64(height) > math.MaxUint32 {
		return math.MaxUint32
	}
	return uint32(height)
}
