#!/usr/bin/env bash
set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
temporary_directory="$(mktemp -d)"
trap 'rm -rf "$temporary_directory"' EXIT

mainnet_config="$repository_root/configs/organiclifecoin.json"
testnet_config="$repository_root/configs/organiclifecoin-testnet.json"

for config in "$mainnet_config" "$testnet_config"; do
  test -f "$config"
  jq -e '
    .coin_name | startswith("OrganicLifeCoin")
  ' "$config" >/dev/null
  if jq -e 'has("rpc_user") or has("rpc_password") or has("rpc_pass")' "$config" >/dev/null; then
    echo "Public network configuration contains a credential field: $config" >&2
    exit 1
  fi
done

jq -e '
  .network == "mainnet" and
  .coin_name == "OrganicLifeCoin" and
  .coin_shortcut == "OLC" and
  .rpc_port == 43723 and
  .xpub_magic == 69274625 and
  .slip44 == 5150
' "$mainnet_config" >/dev/null

jq -e '
  .network == "testnet" and
  .coin_name == "OrganicLifeCoin Testnet" and
  .coin_shortcut == "OLC" and
  .rpc_port == 49718 and
  .xpub_magic == 69274641 and
  .slip44 == 1
' "$testnet_config" >/dev/null

runtime_directory="$temporary_directory/runtime"
data_directory="$temporary_directory/data"

NETWORK=testnet \
NODE_RPC_HOST=127.0.0.1 \
RPC_USER='test"user' \
RPC_PASSWORD='test\password' \
NETWORK_CONFIG_DIR="$repository_root/configs" \
RUNTIME_DIR="$runtime_directory" \
DATA_DIR="$data_directory" \
bash "$repository_root/deploy/render-blockchainconfig.sh" \
  bash -c 'test -z "${RPC_USER+x}" && test -z "${RPC_PASSWORD+x}"'

rendered="$runtime_directory/blockchainconfig.json"
test "$(find "$runtime_directory" -maxdepth 0 -perm 700 -print)" = "$runtime_directory"
test "$(find "$rendered" -maxdepth 0 -perm 600 -print)" = "$rendered"
jq -e '
  .coin_name == "OrganicLifeCoin Testnet" and
  .rpc_url == "http://127.0.0.1:49718/" and
  .rpc_user == "test\"user" and
  .rpc_pass == "test\\password" and
  .xpub_magic == 69274641 and
  .slip44 == 1
' "$rendered" >/dev/null

if NETWORK=invalid \
  NODE_RPC_HOST=127.0.0.1 \
  RPC_USER=user \
  RPC_PASSWORD=password \
  NETWORK_CONFIG_DIR="$repository_root/configs" \
  RUNTIME_DIR="$runtime_directory" \
  DATA_DIR="$data_directory" \
  bash "$repository_root/deploy/render-blockchainconfig.sh" true 2>/dev/null; then
  echo "The renderer accepted an invalid network." >&2
  exit 1
fi

echo "Repository configuration checks passed."
