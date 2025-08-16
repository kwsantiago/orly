# Nostr Relay Performance Comparison

Benchmark results for Khatru, Strfry, Relayer, and Orly relay implementations.

## Test Configuration

- **Events Published**: 10,000 per relay
- **Event Size**: ~1.3KB content
- **Queries Executed**: 100 per relay
- **Concurrency**: 10 simultaneous publishers
- **Platform**: Linux 6.8.0-71-generic
- **Date**: August 15, 2025
- **Orly Version**: v0.6.2-8-gacd2c41

## Performance Results

### Publishing Performance

| Relay | Events Published | Data Size | Duration | Events/sec | Bandwidth |
|-------|-----------------|-----------|----------|------------|-----------|
| **Orly** | 10,000 | 13.03 MB | 1.29s | **7,730.99** | **10.07 MB/s** |
| **Khatru** | 10,000 | 13.03 MB | 1.34s | 7,475.31 | 9.73 MB/s |
| **Strfry** | 10,000 | 13.03 MB | 5.45s | 1,836.17 | 2.39 MB/s |
| **Relayer** | 10,000 | 13.03 MB | 9.02s | 1,109.25 | 1.45 MB/s |


### Query Performance

| Relay | Queries | Events Retrieved | Duration | Queries/sec | Avg Events/Query |
|-------|---------|-----------------|----------|-------------|------------------|
| **Relayer** | 100 | 4,000 | 1.02s | **97.60** | 40.00 |
| **Strfry** | 100 | 4,000 | 1.48s | 67.67 | 40.00 |
| **Orly** | 100 | 4,000 | 3.57s | 28.02 | 40.00 |
| **Khatru** | 100 | 4,000 | 21.41s | 4.67 | 40.00 |


## Implementation Details

### Khatru
- Language: Go
- Backend: SQLite (embedded)
- Dependencies: Go 1.20+, SQLite3
- Publishing: 7,475 events/sec, 1.34s duration
- Querying: 4.67 queries/sec, 21.4s duration

### Strfry
- Language: C++
- Backend: LMDB (embedded)
- Dependencies: flatbuffers, lmdb, zstd, secp256k1, cmake, g++
- Publishing: 1,836 events/sec, 5.45s duration
- Querying: 67.67 queries/sec, 1.48s duration

### Relayer
- Language: Go
- Backend: PostgreSQL (external)
- Dependencies: Go 1.20+, PostgreSQL 12+
- Publishing: 1,109 events/sec, 9.02s duration
- Querying: 97.60 queries/sec, 1.02s duration

## Test Environment

- Platform: Linux 6.8.0-71-generic
- Concurrency: 10 publishers
- Event size: ~1.3KB
- Signature verification: secp256k1
- Content validation: UTF-8

## Docker Setup

All benchmarks can be run using the provided Docker setup:

```bash
# Clone and navigate to benchmark directory
git clone https://github.com/mleku/orly.git
cd orly/cmd/benchmark

# Start all relays
docker compose up -d

# Run benchmarks
docker compose run benchmark -relay ws://orly:7447 -events 10000 -queries 100
docker compose run benchmark -relay ws://khatru:7447 -events 10000 -queries 100
docker compose run benchmark -relay ws://strfry:7777 -events 10000 -queries 100
docker compose run benchmark -relay ws://relayer:7447 -events 10000 -queries 100
```

## Configuration Notes

To achieve optimal Orly performance, ensure logging is minimized:
- Use `--log-level error` flag when starting Orly
- Build with minimal logging tags if compiling from source
- Set environment variable `LOG_LEVEL=error`