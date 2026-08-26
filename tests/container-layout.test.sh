#!/usr/bin/env bash
set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repository_root"

test -f Dockerfile
test -f compose.yaml
test -f .env.example
test -f .dockerignore
test -f .github/workflows/ci.yml
test ! -e deploy/source.Dockerfile
test ! -e build/text
! rg -q '^tests/?$' .dockerignore
! rg -q 'packr' api/text.go go.mod

rg -q '^FROM .+ AS builder$' Dockerfile
rg -q '^USER blockbook$' Dockerfile
rg -q 'useradd .*--uid 10001' Dockerfile
rg -q 'HEALTHCHECK' Dockerfile
rg -q -- '-datadir=/data' Dockerfile
rg -q 'COPY --from=builder .*/blockbook' Dockerfile
rg -q 'COPY --chown=blockbook:blockbook static' Dockerfile
! rg -q 'COPY .*build/text' Dockerfile
rg -q 'go test -tags unittest ./\.\.\.' Dockerfile
rg -q 'govulncheck@v1\.7\.0 ./\.\.\.' Dockerfile
rg -q 'ARG GO_VERSION=1\.27\.0' Dockerfile
rg -q 'GO_AMD64_SHA256=' Dockerfile
rg -q 'blockbook/common\.version=' Dockerfile

rg -q '^NETWORK=testnet$' .env.example
rg -q '^RPC_USER=replace-user$' .env.example
rg -q '^RPC_PASSWORD=replace-password$' .env.example

rg -q 'read_only: true' compose.yaml
rg -q 'no-new-privileges:true' compose.yaml
rg -q 'blockbook-data:/data' compose.yaml
rg -q 'uid=10001,gid=10001' compose.yaml
rg -q '127\.0\.0\.1.*9130' compose.yaml

rg -q 'bash tests/repository-config.test.sh' .github/workflows/ci.yml
rg -q 'bash tests/repository-hygiene.test.sh' .github/workflows/ci.yml
rg -q 'bash tests/container-layout.test.sh' .github/workflows/ci.yml
rg -q 'docker build' .github/workflows/ci.yml
rg -q 'docker run --rm --entrypoint /app/blockbook' .github/workflows/ci.yml

echo "Container layout checks passed."
