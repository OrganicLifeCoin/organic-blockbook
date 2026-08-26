package bchain

import (
	"errors"
)

// BaseChain is base type for bchain.BlockChain
type BaseChain struct {
	Parser  BlockChainParser
	Testnet bool
	Network string
}

// TODO more bchain.BlockChain methods

// GetChainParser returns BlockChainParser
func (b *BaseChain) GetChainParser() BlockChainParser {
	return b.Parser
}

// IsTestnet returns true if the blockchain is testnet
func (b *BaseChain) IsTestnet() bool {
	return b.Testnet
}

// GetNetworkName returns network name
func (b *BaseChain) GetNetworkName() string {
	return b.Network
}

// GetMempoolEntry is not supported by default
func (b *BaseChain) GetMempoolEntry(txid string) (*MempoolEntry, error) {
	return nil, errors.New("GetMempoolEntry: not supported")
}

// PIVX specific
func (b *BaseChain) Findzcserial(serialHex string) (string, error) {
	return "", errors.New("Findserial not supported")
}
