# OrganicLifeCoin Blockbook API

The public service provides JSON over HTTP. New integrations must use API V2. The unversioned `/api/` route returns service status. Other unversioned API routes use the legacy V1 response format.

Amounts in API V2 responses are strings in the smallest OLC unit. This format prevents number rounding in clients.

## Errors

An error response uses this form:

```json
{
  "error": "Error description"
}
```

The HTTP status is `400` for an invalid public request and `500` for an internal failure.

## Service status

`GET /api/`

Returns the Blockbook state and the connected OrganicLifeCoin node state. Check these fields before you use indexed data:

- `blockbook.inSync`
- `blockbook.inSyncMempool`
- `blockbook.bestHeight`
- `backend.blocks`
- `backend.headers`

## Block hash

`GET /api/v2/block-index/{height}`

Returns the block hash for a height.

```json
{
  "blockHash": "..."
}
```

## Block

`GET /api/v2/block/{hash-or-height}?page=1`

Returns the block header and a page of normalized transactions. The default transaction page size is 1000.

## Transaction

`GET /api/v2/tx/{txid}`

Returns transparent inputs, outputs, confirmations, fees, and shielded value counts when present.

Use `?spending=true` to include the transaction that spends each known output. This query can require more database work than the default request.

## Node-specific transaction data

`GET /api/v2/tx-specific/{txid}`

Returns the transaction data that comes directly from the OrganicLifeCoin node. The exact fields follow the node version.

## Address

`GET /api/v2/address/{address}`

Supported query values:

| Name | Meaning |
| --- | --- |
| `page` | Select a result page. The first page is 1. |
| `pageSize` | Set the number of transactions on a page. |
| `from` | Include transactions from this block height. |
| `to` | Include transactions up to this block height. |
| `filter` | Use `inputs`, `outputs`, or an output index. |
| `details` | Use `basic`, `txids`, or `txs`. |

The response can contain the confirmed balance, total received, total sent, unconfirmed balance, transaction count, transaction identifiers, and transaction objects. The selected `details` value controls the transaction data.

## Extended public key

`GET /api/v2/xpub/{xpub}`

This route accepts the same paging, height, filter, and detail values as the address route. Use `gap` to set the address discovery gap. The service caps this value at 100.

## Unspent outputs

`GET /api/v2/utxo/{address-or-xpub}`

Use `?confirmed=true` to exclude mempool outputs. For an extended public key, the response also includes the derived address and path when available.

## Send a signed transaction

`POST /api/v2/sendtx/`

Send the signed raw transaction as the plain request body. The service accepts at most 4 MiB. It does not create or sign transactions.

Example:

```bash
curl --fail \
  --request POST \
  --header 'Content-Type: text/plain' \
  --data-binary 'SIGNED_TRANSACTION_HEX' \
  https://explorer.example.com/api/v2/sendtx/
```

A successful response has this form:

```json
{
  "result": "transaction-id"
}
```

The path form `/api/v2/sendtx/{signed-hex}` remains available for compatibility. Use `POST` for new integrations.

## Fee estimate

`GET /api/v2/estimatefee/{blocks}?conservative=true`

Returns the estimated fee rate for the requested confirmation target. The response uses the decimal OLC unit.

## Shielded serial lookup

`GET /api/v2/findzcserial/{serial-hex}`

Returns the transaction identifier that contains the supplied shielded serial, when the node supports this lookup.

## Live updates

Connect to `/websocket` for JSON requests and live subscriptions. Each message must be no larger than 4 MiB. One subscription request can contain no more than 1000 addresses. Idle or unresponsive connections are closed.

## Public deployment

Place the service behind HTTPS. Apply gateway rate limits to address history, extended-public-key discovery, spending-output lookup, and transaction submission. Do not expose the node RPC port.
