# Nostr Relay Benchmark

## Usage

```bash
# Build
docker build -f cmd/benchmark/Dockerfile -t relay-benchmark .

# Run all relays
docker run --rm relay-benchmark all

# Run specific relay
docker run --rm relay-benchmark orly
docker run --rm relay-benchmark strfry 1000 50
```

## Parameters

```bash
docker run --rm relay-benchmark [relay] [events] [queries]

relay:   all | orly | khatru-badger | khatru-sqlite | strfry | nostr-rs | relayer
events:  number of events (default: 10000)
queries: number of queries (default: 100)
```

## Results

**Date:** August 17, 2025  
**Test:** 5,000 events, 100 queries  
**Docker:** golang:1.24, rust:1.82

| Relay | Version | Events/sec | Queries/sec |
|-------|---------|------------|-------------|
| ORLY | 0.8.0 | 8,762 | 3-7* |
| Khatru-Badger | git HEAD | 7,859 | 3-7* |
| Khatru-SQLite | git HEAD | 206 | 3-4* |
| Strfry | git HEAD | 1,856 | 13* |
| nostr-rs-relay | 0.9.0 | 2,976 | 4-5* |
| Relayer | git HEAD | 1,452 | 6-7* |

**⚠️ Known Limitations:**
- Query rates are NOT representative of production performance
- Uses sequential queries on single connection (not concurrent)
- Tests on fresh database without optimized indexes
- Small dataset (5K events vs millions in production)
- Event publishing rates ARE accurate and representative