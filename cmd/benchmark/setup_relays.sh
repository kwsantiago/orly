#!/bin/bash

# Store script directory before changing directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

WORK_DIR="/tmp/relay-benchmark"
mkdir -p "$WORK_DIR"
cd "$WORK_DIR"

echo "=== Setting up relay implementations ==="

# Clone khatru
echo "Cloning khatru..."
if [ ! -d "khatru" ]; then
    git clone https://github.com/fiatjaf/khatru.git
fi

# Clone relayer
echo "Cloning relayer..."
if [ ! -d "relayer" ]; then
    git clone https://github.com/fiatjaf/relayer.git
fi

# Clone strfry
echo "Cloning strfry..."
if [ ! -d "strfry" ]; then
    git clone https://github.com/hoytech/strfry.git
fi

# Build khatru example
echo "Building khatru..."
cd "$WORK_DIR/khatru"
if [ -f "examples/basic-sqlite3/main.go" ]; then
    cd examples/basic-sqlite3
    go build -o khatru-relay
    echo "Khatru built: $WORK_DIR/khatru/examples/basic-sqlite3/khatru-relay"
else
    echo "No basic-sqlite3 example found in khatru"
fi

# Build relayer
echo "Building relayer..."
cd "$WORK_DIR/relayer"
if [ -f "examples/basic/main.go" ]; then
    cd examples/basic
    go build -o relayer-bin
    echo "Relayer built: $WORK_DIR/relayer/examples/basic/relayer-bin"
else
    echo "Could not find relayer basic example"
fi

# Build strfry (requires cmake and dependencies)
echo "Building strfry..."
cd "$WORK_DIR/strfry"
if command -v cmake &> /dev/null; then
    git submodule update --init
    make setup
    make -j4
    echo "Strfry built: $WORK_DIR/strfry/strfry"
else
    echo "cmake not found, skipping strfry build"
fi

# Build Orly 
echo "Building Orly..."
# Find Orly project root by looking for both .git and main.go in same directory
ORLY_ROOT="$SCRIPT_DIR"
while [[ "$ORLY_ROOT" != "/" ]]; do
    if [[ -d "$ORLY_ROOT/.git" && -f "$ORLY_ROOT/main.go" ]]; then
        break
    fi
    ORLY_ROOT="$(dirname "$ORLY_ROOT")"
done

echo "Building Orly at: $ORLY_ROOT"
if [[ -f "$ORLY_ROOT/main.go" && -d "$ORLY_ROOT/.git" ]]; then
    CURRENT_DIR="$(pwd)"
    cd "$ORLY_ROOT"
    CGO_LDFLAGS="-L/usr/local/lib" PKG_CONFIG_PATH="/usr/local/lib/pkgconfig" go build -o "$WORK_DIR/orly-relay" .
    echo "Orly built: $WORK_DIR/orly-relay"
    cd "$CURRENT_DIR"
else
    echo "Could not find Orly project root with both .git and main.go"
    echo "Searched up from: $SCRIPT_DIR"
fi

echo "=== Setup complete ==="
ls -la "$WORK_DIR"