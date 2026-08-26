FROM ubuntu:22.04 AS builder

ARG DEBIAN_FRONTEND=noninteractive
ARG ROCKSDB_COMMIT=641fae60f63619ed5d0c9d9e4c4ea5a0ffa3e253
ARG GO_VERSION=1.27.0
ARG GO_AMD64_SHA256=675c26c449cbb18fc24b74650de1eabbae6e16f64326fd85a283fb3b58280685
ARG GO_ARM64_SHA256=51798d2c42d0e1c6ed7fd9f48728b4193abac9e8aad6dbac2fe96a81f5909bda
ARG TARGETARCH

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    ca-certificates \
    curl \
    gcc-10 \
    g++-10 \
    git \
    libbz2-dev \
    libgflags-dev \
    libjemalloc-dev \
    liblz4-dev \
    libsnappy-dev \
    libzstd-dev \
    libzmq3-dev \
    pkg-config \
    zlib1g-dev \
    && rm -rf /var/lib/apt/lists/*

RUN set -eu; \
    case "$TARGETARCH" in \
      amd64) go_sha256="$GO_AMD64_SHA256" ;; \
      arm64) go_sha256="$GO_ARM64_SHA256" ;; \
      *) echo "Unsupported build architecture: $TARGETARCH" >&2; exit 1 ;; \
    esac; \
    go_archive="go${GO_VERSION}.linux-${TARGETARCH}.tar.gz"; \
    curl --fail --location --silent --show-error "https://go.dev/dl/${go_archive}" --output "/tmp/${go_archive}"; \
    echo "${go_sha256}  /tmp/${go_archive}" | sha256sum --check --strict; \
    tar -C /usr/local -xzf "/tmp/${go_archive}"; \
    rm "/tmp/${go_archive}"

ENV PATH="/usr/local/go/bin:${PATH}"

RUN git clone https://github.com/facebook/rocksdb.git /opt/rocksdb \
    && cd /opt/rocksdb \
    && git checkout "$ROCKSDB_COMMIT" \
    && CC=gcc-10 CXX=g++-10 PORTABLE=1 DISABLE_WARNING_AS_ERROR=1 \
       CFLAGS=-fPIC CXXFLAGS=-fPIC make -j"$(nproc)" release \
    && cp -a /opt/rocksdb/include/rocksdb /usr/local/include/ \
    && cp -a /opt/rocksdb/librocksdb* /usr/local/lib/ \
    && ldconfig

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

ENV CGO_CFLAGS=-I/usr/local/include \
    CGO_LDFLAGS="-L/usr/local/lib -lrocksdb -lstdc++ -lm -llz4 -lbz2 -lsnappy -lzstd -ljemalloc -lz"

RUN go test -tags unittest ./...
RUN go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...

ARG VERSION=0.1.0
ARG VCS_REF=unknown
ARG BUILD_TIME=unknown

RUN go build -trimpath \
    -ldflags="-s -w -X blockbook/common.version=${VERSION} -X blockbook/common.gitcommit=${VCS_REF} -X blockbook/common.buildtime=${BUILD_TIME}" \
    -o /out/blockbook blockbook.go

FROM ubuntu:22.04

ARG DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    jq \
    libbz2-1.0 \
    libgflags2.2 \
    libjemalloc2 \
    liblz4-1 \
    libsnappy1v5 \
    libstdc++6 \
    libzmq5 \
    libzstd1 \
    zlib1g \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --system --gid 10001 blockbook \
    && useradd --system --uid 10001 --gid blockbook --home-dir /app --shell /usr/sbin/nologin blockbook

WORKDIR /app

COPY --from=builder /usr/local/lib/librocksdb* /usr/local/lib/
COPY --from=builder /out/blockbook /app/blockbook
COPY --chown=blockbook:blockbook static /app/static
COPY deploy/render-blockchainconfig.sh /usr/local/bin/render-blockchainconfig.sh
COPY configs/organiclifecoin*.json /etc/blockbook/networks/

RUN chmod 0755 /usr/local/bin/render-blockchainconfig.sh /app/blockbook \
    && mkdir -p /data /runtime \
    && chown -R blockbook:blockbook /app /data /runtime \
    && ldconfig

USER blockbook

EXPOSE 9130

HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
    CMD curl --fail --silent --show-error http://127.0.0.1:9130/api/v2 >/dev/null || exit 1

ENTRYPOINT ["/usr/local/bin/render-blockchainconfig.sh"]
CMD ["/app/blockbook", "-sync", "-datadir=/data", "-resyncindexperiod=60017", "-resyncmempoolperiod=60017", "-blockchaincfg=/runtime/blockchainconfig.json", "-internal=:19049", "-public=:9130", "-logtostderr"]
