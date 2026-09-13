package pivx

import (
	"blockbook/bchain"
	"blockbook/bchain/coins/btc"
	"encoding/json"
	"strings"

	"github.com/golang/glog"
	"github.com/juju/errors"
)

const (
	mainnetBudgetCycleBlocks = 10080
	testnetBudgetCycleBlocks = 40320
)

// PivXRPC is an interface to JSON-RPC bitcoind service.
type PivXRPC struct {
	*btc.BitcoinRPC
	BitcoinGetChainInfo func() (*bchain.ChainInfo, error)
}

// NewPivXRPC returns new PivXRPC instance.
func NewPivXRPC(config json.RawMessage, pushHandler func(bchain.NotificationType)) (bchain.BlockChain, error) {
	b, err := btc.NewBitcoinRPC(config, pushHandler)
	if err != nil {
		return nil, err
	}

	s := &PivXRPC{
		b.(*btc.BitcoinRPC),
		b.GetChainInfo,
	}
	s.RPCMarshaler = btc.JSONMarshalerV1{}
	s.ChainConfig.SupportsEstimateFee = true
	s.ChainConfig.SupportsEstimateSmartFee = false

	return s, nil
}

// Initialize initializes PivXRPC instance.
func (b *PivXRPC) Initialize() error {
	ci, err := b.GetChainInfo()
	if err != nil {
		return err
	}
	chainName := ci.Chain

	glog.Info("Chain name ", chainName)
	params := GetChainParams(chainName)

	// always create parser
	b.Parser = NewPivXParser(params, b.ChainConfig)

	// parameters for getInfo request
	if params.Net == MainnetMagic {
		b.Testnet = false
		b.Network = "livenet"
	} else {
		b.Testnet = true
		b.Network = "testnet"
	}

	glog.Info("rpc: block chain ", params.Name)

	return nil
}

// GetBlock returns block with given hash.
func (z *PivXRPC) GetBlock(hash string, height uint32) (*bchain.Block, error) {
	var err error
	if hash == "" && height > 0 {
		hash, err = z.GetBlockHash(height)
		if err != nil {
			return nil, err
		}
	}

	glog.V(1).Info("rpc: getblock (verbosity=1) ", hash)

	res := btc.ResGetBlockThin{}
	req := btc.CmdGetBlock{Method: "getblock"}
	req.Params.BlockHash = hash
	req.Params.Verbosity = 1
	err = z.Call(&req, &res)

	if err != nil {
		return nil, errors.Annotatef(err, "hash %v", hash)
	}
	if res.Error != nil {
		return nil, errors.Annotatef(res.Error, "hash %v", hash)
	}

	txs := make([]bchain.Tx, 0, len(res.Result.Txids))
	for _, txid := range res.Result.Txids {
		tx, err := z.GetTransaction(txid)
		if err != nil {
			if err == bchain.ErrTxNotFound {
				glog.Errorf("rpc: getblock: skipping transanction in block %s due error: %s", hash, err)
				continue
			}
			return nil, err
		}
		txs = append(txs, *tx)
	}
	block := &bchain.Block{
		BlockHeader: res.Result.BlockHeader,
		Txs:         txs,
	}
	return block, nil
}

// getsupplyinfo

type CmdGetSupplyInfo struct {
	Method string `json:"method"`
	Params []bool `json:"params"`
}

type ResGetSupplyInfo struct {
	Error  *bchain.RPCError `json:"error"`
	Result struct {
		TransparentSupply json.Number `json:"transparentsupply"`
		ShieldSupply      json.Number `json:"shieldsupply"`
		TotalSupply       json.Number `json:"totalsupply"`
	} `json:"result"`
}

// getmasternodecount

type CmdListPQMasternodes struct {
	Method string `json:"method"`
}

type ResListPQMasternodes struct {
	Error  *bchain.RPCError  `json:"error"`
	Result []json.RawMessage `json:"result"`
}

func pqMasternodeCount(records []json.RawMessage, rpcError *bchain.RPCError) (int, error) {
	if rpcError == nil {
		return len(records), nil
	}
	if strings.Contains(rpcError.Message, "PQ masternodes are not active") {
		return 0, nil
	}
	return 0, rpcError
}

// GetNextSuperBlock returns the next superblock height after nHeight
func (b *PivXRPC) GetNextSuperBlock(nHeight int) int {
	nBlocksPerPeriod := mainnetBudgetCycleBlocks
	if b.Testnet {
		nBlocksPerPeriod = testnetBudgetCycleBlocks
	}
	return nHeight - nHeight%nBlocksPerPeriod + nBlocksPerPeriod
}

// GetChainInfo returns information about the connected backend
// PIVX adds Money Supply to btc implementation
func (b *PivXRPC) GetChainInfo() (*bchain.ChainInfo, error) {
	rv, err := b.BitcoinGetChainInfo()
	if err != nil {
		return nil, err
	}

	glog.V(1).Info("rpc: getsupplyinfo")

	resSi := ResGetSupplyInfo{}
	err = b.Call(&CmdGetSupplyInfo{Method: "getsupplyinfo", Params: []bool{false}}, &resSi)
	if err != nil {
		return nil, err
	}
	if resSi.Error != nil {
		return nil, resSi.Error
	}
	rv.TransparentSupply = resSi.Result.TransparentSupply
	rv.ShieldSupply = resSi.Result.ShieldSupply
	rv.MoneySupply = resSi.Result.TotalSupply

	glog.V(1).Info("rpc: listpqmasternodes")

	resMc := ResListPQMasternodes{}
	err = b.Call(&CmdListPQMasternodes{Method: "listpqmasternodes"}, &resMc)
	if err != nil {
		return nil, err
	}
	masternodeCount, err := pqMasternodeCount(resMc.Result, resMc.Error)
	if err != nil {
		return nil, err
	}
	rv.MasternodeCount = masternodeCount

	rv.NextSuperBlock = b.GetNextSuperBlock(rv.Headers)

	return rv, nil
}

// findserial
type CmdFindSerial struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
}

type ResFindSerial struct {
	Error  *bchain.RPCError `json:"error"`
	Result struct {
		Success bool   `json:"success"`
		Txid    string `json:"txid"`
	} `json:"result"`
}

func (b *PivXRPC) Findzcserial(serialHex string) (string, error) {
	glog.V(1).Info("rpc: findserial")

	res := ResFindSerial{}
	req := CmdFindSerial{Method: "findserial"}
	req.Params = []string{serialHex}
	err := b.Call(&req, &res)

	if err != nil {
		return "", err
	}
	if res.Error != nil {
		return "", res.Error
	}
	if !res.Result.Success {
		return "Serial not found in blockchain", nil
	}
	return res.Result.Txid, nil
}
