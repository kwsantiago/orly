# Orly Relay Benchmark Tool

A performance benchmarking tool for Nostr relays that tests both event ingestion speed and query performance.

## Quick Start (Simple Version)

The repository includes a simple standalone benchmark tool that doesn't require the full Orly dependencies:

```bash
# Build the simple benchmark
go build -o benchmark-simple ./benchmark_simple.go

# Run with default settings
./benchmark-simple

# Or use the convenience script
chmod +x run_benchmark.sh
./run_benchmark.sh --relay ws://localhost:7447 --events 10000
```

## Features

- **Event Publishing Benchmark**: Tests how fast a relay can accept and store events
- **Query Performance Benchmark**: Tests various filter types and query speeds
- **Concurrent Publishing**: Supports multiple concurrent publishers to stress test the relay
- **Detailed Metrics**: Reports events/second, bandwidth usage, and query performance

## Usage

```bash
# Build the tool
go build -o benchmark ./cmd/benchmark

# Run a full benchmark (publish and query)
./benchmark -relay ws://localhost:7447 -events 10000 -queries 100

# Benchmark only publishing
./benchmark -relay ws://localhost:7447 -events 50000 -concurrency 20 -skip-query

# Benchmark only querying
./benchmark -relay ws://localhost:7447 -queries 500 -skip-publish

# Use custom event sizes
./benchmark -relay ws://localhost:7447 -events 10000 -size 2048
```

## Options

- `-relay`: Relay URL to benchmark (default: ws://localhost:7447)
- `-events`: Number of events to publish (default: 10000)
- `-size`: Average size of event content in bytes (default: 1024)
- `-concurrency`: Number of concurrent publishers (default: 10)
- `-queries`: Number of queries to execute (default: 100)
- `-query-limit`: Limit for each query (default: 100)
- `-skip-publish`: Skip the publishing phase
- `-skip-query`: Skip the query phase
- `-v`: Enable verbose output

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
Publishing 10000 events to ws://localhost:7447...
  Published 1000 events...
  Published 2000 events...
  ...

Querying events from ws://localhost:7447...
  Executed 20 queries...
  Executed 40 queries...
  ...

=== Benchmark Results ===

Publish Performance:
  Events Published: 10000
  Total Data: 12.34 MB
  Duration: 5.2s
  Rate: 1923.08 events/second
  Bandwidth: 2.37 MB/second

Query Performance:
  Queries Executed: 100
  Events Returned: 4523
  Duration: 2.1s
  Rate: 47.62 queries/second
  Avg Events/Query: 45.23
```