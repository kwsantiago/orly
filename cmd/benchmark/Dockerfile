FROM golang:1.24-bookworm AS go-base
RUN apt-get update && apt-get install -y git libsecp256k1-dev libsqlite3-dev

FROM go-base AS orly-builder
WORKDIR /build
COPY . .
RUN go build -tags minimal_log -o orly .
RUN go build -o benchmark ./cmd/benchmark

FROM go-base AS khatru-builder
RUN git clone --depth 1 https://github.com/fiatjaf/khatru.git /khatru
WORKDIR /khatru/examples/basic-badger
RUN go build -o khatru-badger .
WORKDIR /khatru/examples/basic-sqlite3
RUN go build -o khatru-sqlite .

FROM go-base AS relayer-builder
RUN git clone --depth 1 https://github.com/fiatjaf/relayer.git /relayer
WORKDIR /relayer/examples/basic
RUN go build -o relayer .

FROM debian:bookworm AS strfry-builder
RUN apt-get update && apt-get install -y \
    git build-essential cmake libssl-dev liblmdb-dev \
    libflatbuffers-dev libsecp256k1-dev libzstd-dev \
    zlib1g-dev libuv1-dev
RUN git clone https://github.com/hoytech/strfry.git /strfry
WORKDIR /strfry
RUN git submodule update --init
RUN make setup-golpe
RUN make -j4

FROM rust:1.82-bookworm AS rust-builder
RUN apt-get update && apt-get install -y libssl-dev pkg-config protobuf-compiler
RUN git clone --depth 1 https://git.sr.ht/~gheartsfield/nostr-rs-relay /relay
WORKDIR /relay
RUN cargo build --release

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y \
    libsecp256k1-1 libsqlite3-0 liblmdb0 libssl3 \
    libflatbuffers2 libzstd1 netcat-openbsd \
    postgresql-15 sudo \
    && rm -rf /var/lib/apt/lists/*

RUN useradd -m -s /bin/bash bench && \
    echo "bench ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers && \
    mkdir -p /var/run/postgresql && \
    chown postgres:postgres /var/run/postgresql && \
    mkdir -p /var/lib/postgresql && \
    chown -R postgres:postgres /var/lib/postgresql

WORKDIR /opt/benchmark

COPY --from=orly-builder /build/orly /build/benchmark /usr/local/bin/
COPY --from=khatru-builder /khatru/examples/basic-badger/khatru-badger /usr/local/bin/
COPY --from=khatru-builder /khatru/examples/basic-sqlite3/khatru-sqlite /usr/local/bin/
COPY --from=relayer-builder /relayer/examples/basic/relayer /usr/local/bin/
COPY --from=strfry-builder /strfry/strfry /usr/local/bin/
COPY --from=rust-builder /relay/target/release/nostr-rs-relay /usr/local/bin/

COPY cmd/benchmark/strfry.conf /opt/benchmark/
COPY cmd/benchmark/bench /usr/local/bin/
RUN chmod +x /usr/local/bin/bench && \
    chown -R bench:bench /opt/benchmark && \
    mkdir -p /etc/postgresql/15/main && \
    chown -R postgres:postgres /etc/postgresql

USER bench
ENTRYPOINT ["/usr/local/bin/bench"]