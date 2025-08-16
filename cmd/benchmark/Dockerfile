FROM golang:1.23-bookworm AS builder

RUN apt-get update && apt-get install -y \
    git \
    build-essential \
    cmake \
    pkg-config \
    libssl-dev \
    liblmdb-dev \
    libsqlite3-dev \
    flatbuffers-compiler \
    flatbuffers-compiler-dev \
    libflatbuffers2 \
    libsecp256k1-1 \
    libsecp256k1-dev \
    lmdb-doc \
    autoconf \
    automake \
    libtool

WORKDIR /build

COPY go.mod go.sum ./
RUN GOTOOLCHAIN=auto go mod download

COPY . .

RUN GOTOOLCHAIN=auto CGO_ENABLED=1 go build -o orly-benchmark ./cmd/benchmark
RUN GOTOOLCHAIN=auto CGO_ENABLED=1 go build -tags minimal_log -o orly-relay .

WORKDIR /relays

RUN git clone https://github.com/fiatjaf/khatru.git && \
    cd khatru/examples/basic-sqlite3 && \
    GOTOOLCHAIN=auto go build -o /relays/khatru-relay

RUN git clone https://github.com/fiatjaf/relayer.git && \
    cd relayer && \
    git checkout 125f2a8 && \
    cd examples/basic && \
    GOTOOLCHAIN=auto go build -o /relays/relayer-bin

RUN apt-get update && apt-get install -y libflatbuffers-dev libzstd-dev && \
    git clone https://github.com/hoytech/strfry.git && \
    cd strfry && \
    git submodule update --init && \
    make setup-golpe && \
    make -j$(nproc) && \
    cp strfry /relays/strfry

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y \
    ca-certificates \
    libsqlite3-0 \
    liblmdb0 \
    libssl3 \
    flatbuffers-compiler \
    libflatbuffers2 \
    libsecp256k1-1 \
    lmdb-utils

RUN mkdir -p /app/bin /app/data

COPY --from=builder /build/orly-benchmark /app/bin/
COPY --from=builder /build/orly-relay /app/bin/
COPY --from=builder /relays/khatru-relay /app/bin/
COPY --from=builder /relays/relayer-bin /app/bin/
COPY --from=builder /relays/strfry /app/bin/

WORKDIR /app

ENV LOG_LEVEL=error

EXPOSE 7447

ENTRYPOINT ["/app/bin/orly-benchmark"]