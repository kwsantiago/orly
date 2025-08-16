# Nostr Relay Benchmark Suite

A comprehensive performance benchmarking suite for Nostr relay implementations, featuring event publishing tests, query profiling, load simulation, and timing instrumentation.

## Features

- **Multi-relay comparison benchmarks** - Compare Khatru, Strfry, Relayer, and Orly
- **Publishing performance testing** - Measure event ingestion rates and bandwidth
- **Query profiling** - Test various filter patterns and query speeds
- **Load pattern simulation** - Constant, spike, burst, sine, and ramp patterns
- **Timing instrumentation** - Track full event lifecycle and identify bottlenecks
- **Concurrent stress testing** - Multiple publishers with connection pooling
- **Production-grade event generation** - Proper secp256k1 signatures and UTF-8 content
- **Comparative reporting** - Markdown, JSON, and CSV format reports

## Prerequisites

- Docker 20.10 or later
- Docker Compose v2.0 or later
- Git

To install Docker and Docker Compose:
- **Ubuntu/Debian**: `sudo apt-get install docker.io docker-compose-v2`
- **macOS**: Install [Docker Desktop](https://www.docker.com/products/docker-desktop/)
- **Windows**: Install [Docker Desktop](https://www.docker.com/products/docker-desktop/)

## Quick Start

```bash
# Clone the repository
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

## Latest Benchmark Results

**Date:** August 15, 2025  
**Orly Version:** v0.6.2-8-gacd2c41

| Relay | Publishing (events/sec) | Querying (queries/sec) | Backend |
|-------|------------------------|------------------------|---------|
| **Orly** | 7,731 | 28.02 | Badger |
| **Khatru** | 7,475 | 4.67 | SQLite |
| **Strfry** | 1,836 | 67.67 | LMDB |
| **Relayer** | 1,109 | 97.60 | PostgreSQL |

*Note: Orly requires `--log-level error` flag for optimal performance.*

See [RELAY_COMPARISON_RESULTS.md](RELAY_COMPARISON_RESULTS.md) for detailed analysis.

## Docker Services

The docker-compose setup includes:

- `orly`: Orly relay on port 7447
- `khatru`: Khatru relay on port 7448
- `strfry`: Strfry relay on port 7450
- `relayer`: Relayer on port 7449 (with PostgreSQL)
- `postgres`: PostgreSQL database for Relayer
- `benchmark`: Benchmark tool

## Usage Examples

### Basic Benchmarking

```bash
# Full benchmark (publish and query)
docker compose run benchmark -relay ws://orly:7447 -events 10000 -queries 100

# Publishing only
docker compose run benchmark -relay ws://orly:7447 -events 50000 -concurrency 20 -skip-query

# Querying only
docker compose run benchmark -relay ws://orly:7447 -queries 500 -skip-publish

# Custom event sizes
docker compose run benchmark -relay ws://orly:7447 -events 10000 -size 2048
```

### Advanced Features

```bash
# Query profiling with subscription testing
docker compose run benchmark -profile -profile-subs -sub-count 100 -sub-duration 30s

# Load pattern simulation
docker compose run benchmark -load -load-pattern spike -load-duration 60s -load-base 50 -load-peak 200

# Full load test suite
docker compose run benchmark -load-suite -load-constraints

# Timing instrumentation
docker compose run benchmark -timing -timing-events 100 -timing-subs -timing-duration 10s

# Generate comparative reports
docker compose run benchmark -report -report-format markdown -report-title "Production Benchmark"
```

## Command Line Options

### Basic Options
- `--relay`: Relay URL to benchmark (default: ws://localhost:7447)
- `--events`: Number of events to publish (default: 10000)
- `--size`: Average size of event content in bytes (default: 1024)
- `--concurrency`: Number of concurrent publishers (default: 10)
- `--queries`: Number of queries to execute (default: 100)
- `--query-limit`: Limit for each query (default: 100)
- `--skip-publish`: Skip the publishing phase
- `--skip-query`: Skip the query phase
- `-v`: Enable verbose output

### Profiling Options
- `--profile`: Run query performance profiling
- `--profile-subs`: Profile subscription performance
- `--sub-count`: Number of concurrent subscriptions (default: 100)
- `--sub-duration`: Duration for subscription profiling (default: 30s)

### Load Testing Options
- `--load`: Run load pattern simulation
- `--load-pattern`: Pattern type: constant, spike, burst, sine, ramp (default: constant)
- `--load-duration`: Duration for load test (default: 60s)
- `--load-base`: Base load in events/sec (default: 50)
- `--load-peak`: Peak load in events/sec (default: 200)
- `--load-pool`: Connection pool size (default: 10)
- `--load-suite`: Run comprehensive load test suite
- `--load-constraints`: Test under resource constraints

### Timing Options
- `--timing`: Run end-to-end timing instrumentation
- `--timing-events`: Number of events for timing (default: 100)
- `--timing-subs`: Test subscription timing
- `--timing-duration`: Duration for subscription timing (default: 10s)

### Report Options
- `--report`: Generate comparative report
- `--report-format`: Output format: markdown, json, csv (default: markdown)
- `--report-file`: Output filename without extension (default: benchmark_report)
- `--report-title`: Report title (default: "Relay Benchmark Comparison")

## Query Types Tested

The benchmark tests various query patterns:
1. Query by kind
2. Query by time range (last hour)
3. Query by tag (p tags)
4. Query by author
5. Complex queries with multiple conditions

## Output Metrics

**Publish Performance:**
- Total events published
- Total data transferred
- Publishing rate (events/second)
- Bandwidth usage (MB/second)

**Query Performance:**
- Total queries executed
- Total events returned
- Query rate (queries/second)
- Average events per query

## Example Output

```
Publishing 10000 events to ws://localhost:7447...
  Published 1000 events...
  Published 2000 events...

Querying events from ws://localhost:7447...
  Executed 20 queries...
  Executed 40 queries...

=== Benchmark Results ===

Publish Performance:
  Events Published: 10000
  Total Data: 13.03 MB
  Duration: 1.29s
  Rate: 7730.99 events/second
  Bandwidth: 10.07 MB/second

Query Performance:
  Queries Executed: 100
  Events Returned: 4000
  Duration: 3.57s
  Rate: 28.02 queries/second
  Avg Events/Query: 40.00
```

## Configuration Notes

### Orly Optimization
For optimal Orly performance, ensure logging is minimized:
- Start with `--log-level error` flag
- Set environment variable `LOG_LEVEL=error`
- Build with minimal logging tags if compiling from source

### Docker Configuration
All relays are pre-configured with:
- Proper dependencies (flatbuffers, libsecp256k1, lmdb, etc.)
- Optimized build flags
- Minimal logging configurations
- Correct port mappings

## Development

The benchmark suite consists of several components:

- `main.go` - Core benchmark orchestration
- `test_signer.go` - secp256k1 event signing
- `simple_event.go` - UTF-8 safe event generation
- `query_profiler.go` - Query performance analysis
- `load_simulator.go` - Load pattern generation
- `timing_instrumentation.go` - Event lifecycle tracking
- `report_generator.go` - Comparative report generation
- `relay_harness.go` - Multi-relay management


## Notes

- All benchmarks use event generation with proper secp256k1 signatures
- Events are generated with valid UTF-8 content to ensure compatibility
- Connection pooling is used for realistic concurrent load testing
- Query patterns test real-world filter combinations
- Docker setup includes all necessary dependencies and configurations
