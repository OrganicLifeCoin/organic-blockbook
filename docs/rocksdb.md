# OLC index storage

OrganicLifeCoin Blockbook stores its index in RocksDB. The current internal data format version is 5.

Stop Blockbook before you copy the database directory. A live filesystem copy can be inconsistent. Blockbook can rebuild the index from an OrganicLifeCoin node that has `txindex=1`.

## Column families

The database uses these column families:

| Name | Purpose |
| --- | --- |
| `default` | Store the internal service state and database format version. |
| `height` | Map a block height to its hash, time, transaction count, and size. |
| `addresses` | Map an address descriptor and height to transaction input and output indexes. |
| `blockTxs` | Store recent rollback data. |
| `transactions` | Cache packed transactions. |
| `addressBalance` | Store transaction counts, sent values, balances, and unspent outputs. |
| `txAddresses` | Store the input and output address descriptors and values for a transaction. |

An address descriptor is the output script that the OLC parser derives from an address or transaction output.

## Internal state

The `default` column family stores the service state under the `internalState` key. The state includes:

- The indexed coin name
- The data format version
- The database state: closed, open, or inconsistent

Blockbook refuses to use a database for a different coin, a different data format, or an inconsistent bulk import. Rebuild the database if these values do not match the running service.

## Height index

The `height` key is a four-byte unsigned block height in big-endian order. Its value contains the packed block hash, block time, transaction count, and block size.

## Address transaction index

The `addresses` key combines an address descriptor with the complemented block height. The complemented height sorts newer records first. The value contains transaction identifiers and their input or output indexes.

Output indexes are non-negative. Input indexes use a complemented value. This representation lets one address record refer to both sides of a transaction.

## Address balance index

The `addressBalance` value contains:

- The number of transactions
- The total sent value
- The current balance
- Unspent transaction outputs in oldest-first order

Each unspent output contains a transaction identifier, output index, block height, and value.

## Transaction address index

The `txAddresses` value contains the block height, all indexed inputs, and all indexed outputs. Each entry stores an address descriptor and value. An output also stores whether it is spent.

## Rollback data

The `blockTxs` column family stores the transaction identifiers and input outpoints required to disconnect recent blocks. The default configuration keeps 300 blocks of rollback data.

## Transaction cache

The `transactions` column family maps a packed transaction identifier to parser-specific packed transaction data.

## Operations

Use one persistent database volume for each network. Do not point mainnet and testnet services at the same directory. Keep regular stopped-service backups if index rebuild time is operationally significant.
