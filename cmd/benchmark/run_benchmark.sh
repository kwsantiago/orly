#!/bin/bash

# Simple Nostr Relay Benchmark Script

# Default values
RELAY_URL="ws://localhost:7447"
EVENTS=10000
SIZE=1024
CONCURRENCY=10
QUERIES=100
QUERY_LIMIT=100

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --relay)
      RELAY_URL="$2"
      shift 2
      ;;
    --events)
      EVENTS="$2"
      shift 2
      ;;
    --size)
      SIZE="$2"
      shift 2
      ;;
    --concurrency)
      CONCURRENCY="$2"
      shift 2
      ;;
    --queries)
      QUERIES="$2"
      shift 2
      ;;
    --query-limit)
      QUERY_LIMIT="$2"
      shift 2
      ;;
    --skip-publish)
      SKIP_PUBLISH="-skip-publish"
      shift
      ;;
    --skip-query)
      SKIP_QUERY="-skip-query"
      shift
      ;;
    *)
      echo "Unknown option: $1"
      echo "Usage: $0 [--relay URL] [--events N] [--size N] [--concurrency N] [--queries N] [--query-limit N] [--skip-publish] [--skip-query]"
      exit 1
      ;;
  esac
done

# Build the benchmark tool if it doesn't exist
if [ ! -f benchmark-simple ]; then
  echo "Building benchmark tool..."
  go build -o benchmark-simple ./benchmark_simple.go
  if [ $? -ne 0 ]; then
    echo "Failed to build benchmark tool"
    exit 1
  fi
fi

# Run the benchmark
echo "Running Nostr relay benchmark..."
echo "Relay: $RELAY_URL"
echo "Events: $EVENTS (size: $SIZE bytes)"
echo "Concurrency: $CONCURRENCY"
echo "Queries: $QUERIES (limit: $QUERY_LIMIT)"
echo ""

./benchmark-simple \
  -relay "$RELAY_URL" \
  -events $EVENTS \
  -size $SIZE \
  -concurrency $CONCURRENCY \
  -queries $QUERIES \
  -query-limit $QUERY_LIMIT \
  $SKIP_PUBLISH \
  $SKIP_QUERY