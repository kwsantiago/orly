# Nostr Relay Performance Comparison

Benchmark results for Khatru, Strfry, Relayer, and Orly relay implementations.

## Test Configuration

- **Events Published**: 1000 per relay
- **Event Size**: 512 bytes content
- **Queries Executed**: 50 per relay  
- **Concurrency**: 5 simultaneous publishers
- **Platform**: Linux 5.15.0-151-generic
- **Date**: 2025-08-08

## Performance Results

### Publishing Performance

| Relay | Events Published | Data Size | Duration | Events/sec | Bandwidth |
|-------|-----------------|-----------|----------|------------|-----------|
| **Khatru** | 1,000 | 0.81 MB | 104.49ms | **9,569.94** | **7.79 MB/s** |
| **Strfry** | 1,000 | 0.81 MB | 747.41ms | 1,337.95 | 1.09 MB/s |
| **Relayer** | 1,000 | 0.81 MB | 890.91ms | 1,122.45 | 0.91 MB/s |
| **Orly** | 1,000 | 0.81 MB | 1.497s | 667.91 | 0.54 MB/s |


### Query Performance

| Relay | Queries | Events Retrieved | Duration | Queries/sec | Avg Events/Query |
|-------|---------|-----------------|----------|-------------|------------------|
| **Relayer** | 50 | 800 | 80.21ms | **623.36** | 16.00 |
| **Strfry** | 50 | 2,000 | 187.86ms | 266.16 | 40.00 |
| **Orly** | 50 | 800 | 10.164s | 4.92 | 16.00 |
| **Khatru** | 50 | 2,000 | 10.487s | 4.77 | 40.00 |


## Implementation Details

### Khatru
- Language: Go
- Backend: SQLite (embedded)
- Dependencies: Go 1.20+, SQLite3
- Publishing: 9,570 events/sec, 104ms duration
- Querying: 4.77 queries/sec, 10.5s duration

### Strfry
- Language: C++
- Backend: LMDB (embedded)
- Dependencies: flatbuffers, lmdb, zstd, secp256k1, cmake, g++
- Publishing: 1,338 events/sec, 747ms duration
- Querying: 266 queries/sec, 188ms duration

### Relayer
- Language: Go
- Backend: PostgreSQL (external)
- Dependencies: Go 1.20+, PostgreSQL 12+
- Publishing: 1,122 events/sec, 891ms duration
- Querying: 623 queries/sec, 80ms duration

### Orly
- Language: Go
- Backend: Badger (embedded)
- Dependencies: Go 1.20+, libsecp256k1
- Publishing: 668 events/sec, 1.5s duration
- Querying: 4.92 queries/sec, 10.2s duration

## Test Environment

- Platform: Linux 5.15.0-151-generic
- Concurrency: 5 publishers
- Event size: 512 bytes
- Signature verification: secp256k1
- Content validation: UTF-8

