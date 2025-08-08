#!/bin/bash

BENCHMARK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RELAY_DIR="/tmp/relay-benchmark"
RESULTS_FILE="$BENCHMARK_DIR/BENCHMARK_RESULTS.md"

cd "$BENCHMARK_DIR"

echo "=== Starting Relay Benchmark Suite ===" | tee "$RESULTS_FILE"
echo "Date: $(date)" | tee -a "$RESULTS_FILE"
echo "" | tee -a "$RESULTS_FILE"

# Function to start a relay and wait for it to be ready
start_relay() {
    local name=$1
    local cmd=$2
    local port=$3
    
    echo "Starting $name on port $port..."
    $cmd &
    local pid=$!
    
    # Wait for relay to be ready
    sleep 3
    
    # Check if process is still running
    if ! kill -0 $pid 2>/dev/null; then
        echo "Failed to start $name"
        return 1
    fi
    
    echo "$name started with PID $pid"
    return $pid
}

# Function to run benchmark and capture results
run_benchmark() {
    local relay_name=$1
    local relay_url=$2
    
    echo "" | tee -a "$RESULTS_FILE"
    echo "## Benchmarking $relay_name" | tee -a "$RESULTS_FILE"
    echo "URL: $relay_url" | tee -a "$RESULTS_FILE"
    echo "" | tee -a "$RESULTS_FILE"
    
    # Run standard benchmark
    echo "### Standard Benchmark" | tee -a "$RESULTS_FILE"
    ./benchmark --relay "$relay_url" --events 5000 --queries 100 --concurrency 10 --size 1024 2>&1 | tee -a "$RESULTS_FILE"
    
    # Run query profiling
    echo "" | tee -a "$RESULTS_FILE"
    echo "### Query Profiling" | tee -a "$RESULTS_FILE"
    ./benchmark --relay "$relay_url" --profile --queries 500 --concurrency 5 2>&1 | tee -a "$RESULTS_FILE"
    
    # Run timing instrumentation
    echo "" | tee -a "$RESULTS_FILE"
    echo "### Timing Instrumentation" | tee -a "$RESULTS_FILE"
    ./benchmark --relay "$relay_url" --timing --timing-events 100 2>&1 | tee -a "$RESULTS_FILE"
    
    # Run load simulation
    echo "" | tee -a "$RESULTS_FILE"
    echo "### Load Simulation (Spike Pattern)" | tee -a "$RESULTS_FILE"
    ./benchmark --relay "$relay_url" --load --load-pattern spike --load-duration 30s --load-base 50 --load-peak 200 2>&1 | tee -a "$RESULTS_FILE"
    
    echo "" | tee -a "$RESULTS_FILE"
    echo "---" | tee -a "$RESULTS_FILE"
}

# Test 1: Khatru
echo "=== Testing Khatru ===" | tee -a "$RESULTS_FILE"
cd "$RELAY_DIR"
if [ -f "khatru/examples/basic-sqlite3/khatru-relay" ]; then
    ./khatru/examples/basic-sqlite3/khatru-relay &
    KHATRU_PID=$!
    sleep 3
    
    if kill -0 $KHATRU_PID 2>/dev/null; then
        run_benchmark "Khatru" "ws://localhost:7447"
        kill $KHATRU_PID 2>/dev/null
        wait $KHATRU_PID 2>/dev/null
    else
        echo "Khatru failed to start" | tee -a "$RESULTS_FILE"
    fi
else
    echo "Khatru binary not found" | tee -a "$RESULTS_FILE"
fi

# Test 2: Strfry
echo "=== Testing Strfry ===" | tee -a "$RESULTS_FILE"
if [ -f "strfry/strfry" ]; then
    # Create minimal strfry config
    cat > /tmp/strfry.conf <<EOF
relay {
    bind = "127.0.0.1"
    port = 7447
    nofiles = 0
    realIpHeader = ""
    info {
        name = "strfry test"
        description = "benchmark test relay"
    }
}
events {
    maxEventSize = 65536
    rejectEventsNewerThanSeconds = 900
    rejectEventsOlderThanSeconds = 94608000
    rejectEphemeralEventsOlderThanSeconds = 60
    rejectFutureEventsSeconds = 900
}
db {
    path = "/tmp/strfry-db"
}
EOF
    
    rm -rf /tmp/strfry-db
    ./strfry/strfry --config /tmp/strfry.conf relay &
    STRFRY_PID=$!
    sleep 5
    
    if kill -0 $STRFRY_PID 2>/dev/null; then
        run_benchmark "Strfry" "ws://localhost:7447"
        kill $STRFRY_PID 2>/dev/null
        wait $STRFRY_PID 2>/dev/null
    else
        echo "Strfry failed to start" | tee -a "$RESULTS_FILE"
    fi
else
    echo "Strfry binary not found" | tee -a "$RESULTS_FILE"
fi

# Test 3: Relayer
echo "=== Testing Relayer ===" | tee -a "$RESULTS_FILE"
if [ -f "relayer/examples/basic/relayer-bin" ]; then
    # Start PostgreSQL container for relayer
    docker run -d --name relay-postgres-$$ -e POSTGRES_PASSWORD=postgres \
        -e POSTGRES_DB=nostr -p 5433:5432 postgres:15-alpine
    
    sleep 5
    
    # Start relayer
    cd "$RELAY_DIR/relayer/examples/basic"
    POSTGRESQL_DATABASE="postgres://postgres:postgres@localhost:5433/nostr?sslmode=disable" \
        ./relayer-bin &
    RELAYER_PID=$!
    sleep 3
    
    if kill -0 $RELAYER_PID 2>/dev/null; then
        run_benchmark "Relayer" "ws://localhost:7447"
        kill $RELAYER_PID 2>/dev/null
        wait $RELAYER_PID 2>/dev/null
    else
        echo "Relayer failed to start" | tee -a "$RESULTS_FILE"
    fi
    
    # Clean up PostgreSQL container
    docker stop relay-postgres-$$ && docker rm relay-postgres-$$
    cd "$RELAY_DIR"
else
    echo "Relayer binary not found" | tee -a "$RESULTS_FILE"
fi

# Test 4: Orly
echo "=== Testing Orly ===" | tee -a "$RESULTS_FILE"
if [ -f "orly-relay" ]; then
    # Start Orly on different port to avoid conflicts
    ORLY_PORT=7448 ORLY_DATA_DIR=/tmp/orly-benchmark ORLY_SPIDER_TYPE=none ./orly-relay &
    ORLY_PID=$!
    sleep 3
    
    if kill -0 $ORLY_PID 2>/dev/null; then
        run_benchmark "Orly" "ws://localhost:7448"
        kill $ORLY_PID 2>/dev/null
        wait $ORLY_PID 2>/dev/null
    else
        echo "Orly failed to start" | tee -a "$RESULTS_FILE"
    fi
    
    # Clean up Orly data
    rm -rf /tmp/orly-benchmark
else
    echo "Orly binary not found" | tee -a "$RESULTS_FILE"
fi

# Generate comparative report
echo "" | tee -a "$RESULTS_FILE"
echo "=== Generating Comparative Report ===" | tee -a "$RESULTS_FILE"
cd "$BENCHMARK_DIR"
./benchmark --report --report-format markdown --report-file final_comparison 2>&1 | tee -a "$RESULTS_FILE"

echo "" | tee -a "$RESULTS_FILE"
echo "=== Benchmark Suite Complete ===" | tee -a "$RESULTS_FILE"
echo "Results saved to: $RESULTS_FILE" | tee -a "$RESULTS_FILE"