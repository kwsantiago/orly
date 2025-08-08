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

## Quick Start

```bash
# Build the benchmark tool
cd cmd/benchmark
CGO_LDFLAGS="-L/usr/local/lib" PKG_CONFIG_PATH="/usr/local/lib/pkgconfig" go build -o benchmark .

# Run simple benchmark
./benchmark --relay ws://localhost:7447 --events 1000 --queries 50

# Run full comparison benchmark  
./setup_relays.sh  # Setup all relay implementations
./run_all_benchmarks.sh  # Run benchmarks on all relays
```

## Latest Benchmark Results

| Relay | Publishing (events/sec) | Querying (queries/sec) | Backend |
|-------|------------------------|------------------------|---------|
| **Khatru** | 9,570 | 4.77 | SQLite |
| **Strfry** | 1,338 | 266.16 | LMDB |
| **Relayer** | 1,122 | 623.36 | PostgreSQL |
| **Orly** | 668 | 4.92 | Badger |

See [RELAY_COMPARISON_RESULTS.md](RELAY_COMPARISON_RESULTS.md) for detailed analysis.

## Core Benchmarking

### Basic Usage

```bash
# Run a full benchmark (publish and query)
./benchmark --relay ws://localhost:7447 --events 10000 --queries 100

# Benchmark only publishing
./benchmark --relay ws://localhost:7447 --events 50000 --concurrency 20 --skip-query

# Benchmark only querying
./benchmark --relay ws://localhost:7447 --queries 500 --skip-publish

# Use custom event sizes
./benchmark --relay ws://localhost:7447 --events 10000 --size 2048
```

### Advanced Features

```bash
# Query profiling with subscription testing
./benchmark --profile --profile-subs --sub-count 100 --sub-duration 30s

# Load pattern simulation
./benchmark --load --load-pattern spike --load-duration 60s --load-base 50 --load-peak 200

# Full load test suite
./benchmark --load-suite --load-constraints

# Timing instrumentation
./benchmark --timing --timing-events 100 --timing-subs --timing-duration 10s

# Generate comparative reports
./benchmark --report --report-format markdown --report-title "Production Benchmark"
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

### Multi-Relay Options
- `--multi-relay`: Use multi-relay harness
- `--relay-bin`: Path to relay binary
- `--install`: Install relay dependencies and binaries
- `--install-secp`: Install only secp256k1 library
- `--work-dir`: Working directory for builds (default: /tmp/relay-build)
- `--install-dir`: Installation directory for binaries (default: /usr/local/bin)

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

## Output

The tool provides detailed metrics including:

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
Publishing 1000 events to ws://localhost:7447...
  Published 1000 events...

Querying events from ws://localhost:7447...
  Executed 20 queries...
  Executed 40 queries...

=== Benchmark Results ===

Publish Performance:
  Events Published: 1000
  Total Data: 0.81 MB
  Duration: 890.91ms
  Rate: 1122.45 events/second
  Bandwidth: 0.91 MB/second

Query Performance:
  Queries Executed: 50
  Events Returned: 800
  Duration: 80.21ms
  Rate: 623.36 queries/second
  Avg Events/Query: 16.00
```

## Relay Setup

First run `./setup_relays.sh` to build all relay binaries, then start individual relays:

### Khatru (SQLite)
```bash
cd /tmp/relay-benchmark/khatru/examples/basic-sqlite3
./khatru-relay
```

### Strfry (LMDB)
```bash
cd /tmp/relay-benchmark/strfry
./strfry --config strfry.conf relay
```

### Relayer (PostgreSQL)
```bash
# Start PostgreSQL
docker run -d --name relay-postgres -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=nostr -p 5433:5432 postgres:15-alpine

# Run relayer
cd /tmp/relay-benchmark/relayer/examples/basic
POSTGRESQL_DATABASE="postgres://postgres:postgres@localhost:5433/nostr?sslmode=disable" \
  ./relayer-bin
```

### Orly (Badger)
```bash
cd /tmp/relay-benchmark
ORLY_PORT=7448 ORLY_DATA_DIR=/tmp/orly-benchmark ORLY_SPIDER_TYPE=none ./orly-relay
```

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
