#!/usr/bin/env bash
set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repository_root"

obsolete_files=(
  .gitlab-ci.yml
  launch.sh
  build/blockchaincfg.json
  build/tnblockchaincfg.json
  configs/coins/pivx.json
  configs/coins/pivx_testnet.json
  static/pivx-32.png
  static/pivx-og.jpg
  static/pivx-sm.png
  static/pivx.png
  static/test-socketio.html
  static/test-websocket.html
)

for file in "${obsolete_files[@]}"; do
  if [[ -e "$file" ]]; then
    echo "Obsolete public file remains: $file" >&2
    exit 1
  fi
done

olc_only_files=(
  bchain/coins/eth
  bchain/mempool_ethereum_type.go
  db/rocksdb_ethereumtype.go
  db/rocksdb_ethereumtype_test.go
  tests/dbtestdata/dbtestdata_ethereumtype.go
  static/templates/txdetail_ethereumtype.html
)

for file in "${olc_only_files[@]}"; do
  if [[ -e "$file" ]]; then
    echo "An unrelated Ethereum implementation remains: $file" >&2
    exit 1
  fi
done

if rg -n 'go-ethereum|ChainEthereumType|bchain/coins/eth' go.mod api bchain db server; then
  echo "The OLC build still compiles an unrelated Ethereum path." >&2
  exit 1
fi

legacy_coin_pattern='177''6cash|free''dombuzz'
if rg -n -i "$legacy_coin_pattern" \
  --glob '!tests/repository-hygiene.test.sh' \
  --glob '!go.sum' .; then
  echo "Obsolete coin branding remains." >&2
  exit 1
fi

if rg -n 'test-socketio\.html|test-websocket\.html' server; then
  echo "A public diagnostic frontend route remains." >&2
  exit 1
fi

if [[ -e server/socketio.go ]] || rg -n -i 'socket\.io|socketio' go.mod server common docs README.md; then
  echo "The unsupported legacy Socket.IO surface remains." >&2
  exit 1
fi

if rg -n 'net/http/pprof|flag\.String\("prof"' blockbook.go; then
  echo "The production binary still includes the optional profiling server." >&2
  exit 1
fi

if rg -n 'charts/(supply|network|github)|plot_data' server static/templates; then
  echo "The explorer exposes a chart route without shipped data." >&2
  exit 1
fi

if rg -n -i 'trezor|bitcoin|ethereum' docs; then
  echo "Public OLC documentation still contains unrelated coin examples." >&2
  exit 1
fi

rg -q 'txindex=1' README.md
rg -q 'SetReadLimit' server/websocket.go
rg -q 'SetReadDeadline' server/websocket.go
if rg -q 'go s\.onRequest' server/websocket.go; then
  echo "WebSocket requests can bypass per-connection serialization." >&2
  exit 1
fi
rg -q 'done.*chan struct' server/websocket.go
rg -q 'enqueueResponse' server/websocket.go
if rg -q 'close\(c\.out\)' server/websocket.go; then
  echo "WebSocket output channel can still be closed while producers use it." >&2
  exit 1
fi
if rg -q '"status": err\.Error\(\)' server/websocket.go; then
  echo "WebSocket metrics contain unbounded error labels." >&2
  exit 1
fi
if rg -q 'RPCLatency\.With\(common\.Labels\{[^}]*"error"' bchain/coins/blockchain.go; then
  echo "RPC metrics contain unbounded error labels." >&2
  exit 1
fi
rg -q 'RPCLatency\.With\(common\.Labels\{[^}]*"status": status' bchain/coins/blockchain.go
if rg -q 'IndexResyncErrors\.With\(common\.Labels\{[^}]*err\.Error\(\)' db/sync.go; then
  echo "Index metrics contain unbounded error labels." >&2
  exit 1
fi
rg -q 'IndexResyncErrors\.With\(common\.Labels\{"status": "error"\}\)' db/sync.go
rg -q '^const maxAddressesGap = 100$' api/xpub.go
rg -q '^const outChannelSize = 16$' server/websocket.go
rg -q 'validateWebsocketRequestMetadata' server/websocket.go
rg -q 'ValidateAddressLength' api/worker.go server/websocket.go bchain/coins/btc/bitcoinparser.go
rg -q 'MaxBytesReader.*maxSignedTransactionFormBytes' server/public.go
rg -q 'validateSignedTransactionHex' server/public.go
rg -q 'unlock := lockXpub' api/xpub.go

rg -q 'OrganicLifeCoin logo' static/templates/base.html
rg -q '/static/organiclifecoin.png' static/templates/base.html
rg -q 'BlockbookAbout: "OrganicLifeCoin Blockbook indexer and explorer\."' api/text.go
rg -q 'TOSLink:.*"https://github.com/OrganicLifeCoin/organic-blockbook"' api/text.go

if rg -q 'trezor-logo' static/templates/base.html static/css/main.css; then
  echo "An obsolete frontend class remains." >&2
  exit 1
fi

vendor_files=(
  static/vendor/bootstrap-4.3.1.min.css
  static/vendor/bootstrap-4.3.1.min.js
  static/vendor/jquery-3.5.1.slim.min.js
  static/vendor/popper-1.14.7.min.js
  static/vendor/THIRD_PARTY_NOTICES.txt
)

for file in "${vendor_files[@]}"; do
  test -s "$file"
done

test "$(stat -c '%a' static/js/qrcode.min.js 2>/dev/null || stat -f '%Lp' static/js/qrcode.min.js)" = "644"
rg -q 'QRCode.js' static/vendor/THIRD_PARTY_NOTICES.txt

if command -v sha256sum >/dev/null 2>&1; then
  favicon_hash="$(sha256sum static/favicon.ico | cut -d ' ' -f 1)"
else
  favicon_hash="$(shasum -a 256 static/favicon.ico | cut -d ' ' -f 1)"
fi
test "$favicon_hash" = "8adc9e946659dc0a050e1592a42374549c15e0e9af7e0c040e2439eccfc1455f"

if rg -n '<(script|link)[^>]+https://|@import url\("https://' \
  static/templates/base.html static/css/main.css; then
  echo "The explorer loads a remote executable or stylesheet." >&2
  exit 1
fi

echo "Repository hygiene checks passed."
