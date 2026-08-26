# OrganicLifeCoin Blockbook

OrganicLifeCoin Blockbook indexes the OLC blockchain and serves the public explorer frontend.

The repository derives from Trezor Blockbook and the PIVX-compatible parser implementation. OrganicLifeCoin contributors maintain the OLC network integration and explorer.

## Included components

- A RocksDB blockchain indexer
- OrganicLifeCoin mainnet and testnet parser support
- REST API V1 and V2
- WebSocket live updates
- An integrated server-rendered explorer
- OLC templates, styles, scripts, and images
- A testnet-first container deployment

The frontend and indexer run in one process. This design keeps page data, API behavior, and live updates on one origin.

## Security boundary

Blockbook reads public blockchain data from a private OrganicLifeCoin node. It does not store wallet seeds, private keys, passwords, or wallet files.

The transaction API accepts a signed raw transaction and sends it to the node. Blockbook cannot create the signature or spend collateral.

Keep the node RPC port and the internal Blockbook port on a private network. Publish only the public explorer port through an HTTPS gateway.

The container applies these controls:

- An unprivileged user with a fixed user ID
- A read-only root filesystem in Compose
- No Linux capabilities
- No privilege escalation
- A private runtime directory
- A persistent volume for the index database
- JSON-safe runtime configuration

## Data flow

1. The OrganicLifeCoin node provides blocks, transactions, mempool data, and optional ZeroMQ notifications.
2. Blockbook parses the node data and writes the index to RocksDB.
3. The public API reads normalized records from the index.
4. The integrated explorer renders the same records as HTML pages.

## Explorer frontend

The integrated frontend provides these primary routes:

| Route | Purpose |
| --- | --- |
| `/blocks` | Show recent blocks. |
| `/block/{hash-or-height}` | Show one block. |
| `/tx/{txid}` | Show one transaction. |
| `/address/{address}` | Show address activity. |
| `/xpub/{xpub}` | Show extended-public-key activity. |
| `/mempool` | Show mempool activity. |
| `/status` | Show indexer and node status. |
| `/sendtx` | Send a signed raw transaction. |

The frontend files are in `static/`. The Go server reads the templates and assets at runtime.

## Public API

Use API V2 for new integrations.

| Endpoint | Purpose |
| --- | --- |
| `/api/` | Return service and node status. |
| `/api/v2/block-index/{height}` | Return a block hash. |
| `/api/v2/block/{hash-or-height}` | Return block data. |
| `/api/v2/tx/{txid}` | Return normalized transaction data. |
| `/api/v2/tx-specific/{txid}` | Return node-specific transaction data. |
| `/api/v2/address/{address}` | Return address data. |
| `/api/v2/xpub/{xpub}` | Return extended-public-key data. |
| `/api/v2/utxo/{address-or-xpub}` | Return unspent outputs. |
| `/api/v2/sendtx/{signed-hex}` | Send a signed transaction. |
| `/websocket` | Provide live API updates. |

See [docs/api.md](docs/api.md) for the full API format.

## Network defaults

| Value | Mainnet | Testnet |
| --- | ---: | ---: |
| Parser name | `OrganicLifeCoin` | `OrganicLifeCoin Testnet` |
| Node RPC port | `43723` | `49718` |
| Node ZeroMQ port | `38349` | `38349` |
| Extended-public-key magic | `69274625` | `69274641` |
| SLIP-0044 value | `5150` | `1` |
| Governance cycle | `10,080` blocks | `40,320` blocks |

The supplied environment example enables testnet.

## Requirements

- Docker Engine with the Compose plugin
- An OrganicLifeCoin node with private RPC access
- Persistent storage for the Blockbook database
- An HTTPS reverse proxy for public service

Local Go development uses Go 1.27. The container build installs the pinned Go toolchain and RocksDB version automatically.

The first index synchronization can use significant CPU, storage, and memory. The required capacity depends on the current chain size.

## Configure the node

Enable the node RPC server and the full transaction index. Create one RPC user for Blockbook. The node configuration must include:

```ini
server=1
txindex=1
rpcuser=replace-with-a-dedicated-user
rpcpassword=replace-with-a-long-random-password
```

If the node previously ran without `txindex=1`, stop it, add this setting, and start it once with `-reindex`. Wait for the node to finish the reindex before you start Blockbook. Blockbook needs the full transaction index to read historical transaction inputs.

If you use live notifications, publish block and transaction hashes on the configured ZeroMQ endpoint.

Permit connections only from the Blockbook host or container network. Do not publish the node RPC port to the internet.

## Start with Docker Compose

1. Clone the repository.

```bash
git clone git@github.com:OrganicLifeCoin/organic-blockbook.git
cd organic-blockbook
```

2. Create the local environment file.

```bash
cp .env.example .env
```

3. Set `RPC_USER` and `RPC_PASSWORD` in `.env`.

4. Make sure that `NODE_RPC_HOST` resolves from the container.

5. Build and start Blockbook.

```bash
docker compose up -d --build
```

6. Read the service status.

```bash
curl --fail http://127.0.0.1:9130/api/
```

7. Read the container health state.

```bash
docker compose ps
```

The Compose file binds the public port to `127.0.0.1`. A local gateway can publish this port through HTTPS.

## Environment values

| Name | Required | Purpose |
| --- | --- | --- |
| `NETWORK` | No | Select `testnet` or `mainnet`. The default is `testnet`. |
| `NODE_RPC_HOST` | Yes | Set the private node host. |
| `RPC_USER` | Yes | Set the node RPC user. |
| `RPC_PASSWORD` | Yes | Set the node RPC password. |
| `NODE_RPC_PORT` | No | Override the selected network RPC port. |
| `NODE_ZMQ_HOST` | No | Enable live notifications from this node host. |
| `NODE_ZMQ_PORT` | No | Override the selected network ZeroMQ port. |
| `PUBLIC_PORT` | No | Set the local published port. The default is `9130`. |
| `RPC_TIMEOUT` | No | Set the node timeout in seconds. The default is `25`. |
| `MEMPOOL_WORKERS` | No | Set the mempool worker count. The default is `8`. |
| `MEMPOOL_SUB_WORKERS` | No | Set the mempool sub-worker count. The default is `2`. |
| `BLOCK_ADDRESSES_TO_KEEP` | No | Set the recent address-cache depth. The default is `300`. |

Do not commit `.env`. The startup script writes credentials to a private runtime directory inside the container.

## Reverse proxy

Publish the local Blockbook port through an HTTPS reverse proxy.

Example Caddy configuration:

```caddyfile
explorer.example.com {
    encode zstd gzip
    reverse_proxy 127.0.0.1:9130
}
```

Add gateway request limits for the transaction-send route and expensive address queries. Keep the gateway and Blockbook on the same trusted host or network.

## Database backup

Stop Blockbook before you copy the database volume. This action creates a consistent filesystem backup.

```bash
docker compose stop blockbook
docker run --rm \
  --volume organic-blockbook_blockbook-data:/data:ro \
  --volume "$PWD:/backup" \
  ubuntu:22.04 \
  tar -C /data -czf /backup/blockbook-data.tar.gz .
docker compose start blockbook
```

Blockbook can rebuild the database from the node. A backup reduces recovery time.

## Update the service

1. Deploy and index testnet first.
2. Pull the reviewed source revision.
3. Build a new image.
4. Stop the old container.
5. Start the new container with the same data volume.
6. Make sure that `/api/` reports an in-sync state.
7. Promote the reviewed image to mainnet.

## Development checks

Run the repository checks:

```bash
bash tests/repository-config.test.sh
bash tests/repository-hygiene.test.sh
bash tests/container-layout.test.sh
```

Run parser tests:

```bash
go test -tags unittest ./bchain/...
```

The complete test suite requires RocksDB `5.18.3` and its development headers. GitHub Actions runs the complete suite in the pinned build environment.

## Repository layout

| Path | Purpose |
| --- | --- |
| `api/` | Normalize indexed data for public APIs. |
| `bchain/` | Connect to nodes and parse blockchain data. |
| `db/` | Store and read the RocksDB index. |
| `server/` | Serve APIs, WebSockets, and explorer pages. |
| `static/` | Store the integrated explorer frontend. |
| `configs/` | Store public OLC network values. |
| `deploy/` | Render private runtime configuration. |
| `docs/` | Store upstream Blockbook technical references. |

## Release checklist

1. Make sure that the release uses the reviewed OLC network configuration.
2. Make sure that the node RPC port is private.
3. Make sure that the public gateway uses HTTPS and request limits.
4. Run all repository checks.
5. Run all available Go tests.
6. Build the container from a clean checkout.
7. Start with a new testnet database.
8. Make sure that blocks, transactions, addresses, and live updates work.
9. Make sure that signed transaction submission works.
10. Promote the same image digest after review.

## Upstream work and license

This repository is based on [Trezor Blockbook](https://github.com/trezor/blockbook). The OLC parser derives from PIVX-compatible Blockbook work.

OrganicLifeCoin contributors added OLC network parameters, shield-transaction support, deployment controls, and the integrated OLC explorer branding.

The project uses the GNU Affero General Public License version 3. See [COPYING](COPYING) and [NOTICE](NOTICE).
