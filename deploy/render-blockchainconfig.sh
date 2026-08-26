#!/usr/bin/env bash
set -euo pipefail

required() {
  local name="$1"
  if [[ -z "${!name:-}" ]]; then
    echo "$name is required." >&2
    exit 1
  fi
}

valid_host() {
  [[ "$1" =~ ^[A-Za-z0-9][A-Za-z0-9.-]{0,252}$ ]] && [[ "$1" != *..* ]]
}

integer() {
  local name="$1"
  local value="$2"
  local minimum="$3"
  local maximum="$4"
  if [[ ! "$value" =~ ^[0-9]+$ ]] || (( value < minimum || value > maximum )); then
    echo "$name is outside the allowed range." >&2
    exit 1
  fi
}

required NODE_RPC_HOST
required RPC_USER
required RPC_PASSWORD

if ! valid_host "$NODE_RPC_HOST"; then
  echo "NODE_RPC_HOST is invalid." >&2
  exit 1
fi

network="${NETWORK:-testnet}"
network_config_directory="${NETWORK_CONFIG_DIR:-/etc/blockbook/networks}"
case "$network" in
  mainnet) network_config="$network_config_directory/organiclifecoin.json" ;;
  testnet) network_config="$network_config_directory/organiclifecoin-testnet.json" ;;
  *)
    echo "NETWORK must be mainnet or testnet." >&2
    exit 1
    ;;
esac

if ! jq -e '
  (.coin_name | type == "string" and length > 0) and
  (.coin_shortcut | type == "string" and length > 0) and
  (.coin_label | type == "string" and length > 0) and
  (.rpc_port | type == "number") and
  (.zmq_port | type == "number") and
  (.xpub_magic | type == "number") and
  (.slip44 | type == "number")
' "$network_config" >/dev/null; then
  echo "The network configuration is invalid." >&2
  exit 1
fi

coin_name="$(jq -r '.coin_name' "$network_config")"
coin_shortcut="$(jq -r '.coin_shortcut' "$network_config")"
coin_label="$(jq -r '.coin_label' "$network_config")"
rpc_port="${NODE_RPC_PORT:-$(jq -r '.rpc_port' "$network_config")}"
zmq_port="${NODE_ZMQ_PORT:-$(jq -r '.zmq_port' "$network_config")}"
xpub_magic="$(jq -r '.xpub_magic' "$network_config")"
slip44="$(jq -r '.slip44' "$network_config")"
rpc_timeout="${RPC_TIMEOUT:-25}"
mempool_workers="${MEMPOOL_WORKERS:-8}"
mempool_sub_workers="${MEMPOOL_SUB_WORKERS:-2}"
block_addresses_to_keep="${BLOCK_ADDRESSES_TO_KEEP:-300}"

integer NODE_RPC_PORT "$rpc_port" 1 65535
integer NODE_ZMQ_PORT "$zmq_port" 1 65535
integer RPC_TIMEOUT "$rpc_timeout" 1 300
integer MEMPOOL_WORKERS "$mempool_workers" 1 128
integer MEMPOOL_SUB_WORKERS "$mempool_sub_workers" 1 128
integer BLOCK_ADDRESSES_TO_KEEP "$block_addresses_to_keep" 1 1000000

message_queue_binding=""
if [[ -n "${NODE_ZMQ_HOST:-}" ]]; then
  if ! valid_host "$NODE_ZMQ_HOST"; then
    echo "NODE_ZMQ_HOST is invalid." >&2
    exit 1
  fi
  message_queue_binding="tcp://${NODE_ZMQ_HOST}:${zmq_port}"
fi

runtime_directory="${RUNTIME_DIR:-/runtime}"
data_directory="${DATA_DIR:-/data}"
umask 077
mkdir -p "$runtime_directory" "$data_directory"

jq -n \
  --arg coin_name "$coin_name" \
  --arg coin_shortcut "$coin_shortcut" \
  --arg coin_label "$coin_label" \
  --arg rpc_url "http://${NODE_RPC_HOST}:${rpc_port}/" \
  --arg rpc_user "$RPC_USER" \
  --arg rpc_pass "$RPC_PASSWORD" \
  --arg message_queue_binding "$message_queue_binding" \
  --argjson rpc_timeout "$rpc_timeout" \
  --argjson xpub_magic "$xpub_magic" \
  --argjson slip44 "$slip44" \
  --argjson mempool_workers "$mempool_workers" \
  --argjson mempool_sub_workers "$mempool_sub_workers" \
  --argjson block_addresses_to_keep "$block_addresses_to_keep" \
  '{
    coin_name: $coin_name,
    coin_shortcut: $coin_shortcut,
    coin_label: $coin_label,
    rpc_url: $rpc_url,
    rpc_user: $rpc_user,
    rpc_pass: $rpc_pass,
    rpc_timeout: $rpc_timeout,
    parse: true,
    message_queue_binding: $message_queue_binding,
    subversion: "",
    address_format: "",
    xpub_magic: $xpub_magic,
    slip44: $slip44,
    mempool_workers: $mempool_workers,
    mempool_sub_workers: $mempool_sub_workers,
    block_addresses_to_keep: $block_addresses_to_keep
  }' > "$runtime_directory/blockchainconfig.json"

unset RPC_USER RPC_PASSWORD
exec "$@"
