#!/usr/bin/env bash

printf "### multi-relay installer\n\nthis is written for ubuntu 24.04, may work with debian and other debian based distros. REQUIRES SUDO.\n"
printf "\nthis script must be run as '. %s/install.sh', if not, stop and invoke again with the . in front.\n" "`pwd`"
printf "\nnote that the c++ and rust builds are run single threaded to limit memory usage; go builds do not use more than one CPU thread. also all go builds are done after cleaning the module cache to be fair\n\n"
read -p "Press Enter to continue, ctrl-C to stop."
PREVPATH=`pwd`
printf "\n>>> updating apt\n"
sudo apt update > /dev/null 2>&1
printf "\n>>> installing prerequisite deb packages\n"
sudo apt install -y \
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
  libtool \
  libflatbuffers-dev \
  libzstd-dev \
  zlib1g-dev \
  protobuf-compiler \
  pkg-config \
  libssl-dev \
  > /dev/null 2>&1
printf "\n>>> installing go environment script\n"
cp .goenv $HOME/
chmod +x $HOME/.goenv
cd $HOME || exit1
printf "\n>>> downloading Go\n"
wget -nc https://go.dev/dl/go1.25.0.linux-amd64.tar.gz > /dev/null 2>&1
printf "\n>>> removing previous Go installation\n"
sudo rm -rf $HOME/go
printf "\n>>> unpacking Go install archive\n"
tar xf go1.25.0.linux-amd64.tar.gz
printf "\n>>> setting environment for Go\n"
. $HOME/.goenv
printf "\ninstalling benchmark tool\n"
cd $PREVPATH
go build && mv benchmark $HOME/.local/bin/relay-benchmark
printf "\n>>> installing rust using rustup (just press enter for default version)\n"
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
printf "\n>>> setting rust environment variables to use cargo\n"
. "$HOME/.cargo/env"
cd $PREVPATH
printf "\n>>> installing ORLY\n"
go clean -modcache
cd ../..
time go build && \
  mv orly.dev $HOME/.local/bin/orly
cd $PREVPATH
printf "\n>>> installing khatru basic-badger\n"
go clean -modcache
git clone https://github.com/fiatjaf/khatru.git && \
  cd khatru/examples/basic-badger && \
  time go build && \
  mv basic-badger $HOME/.local/bin/khatru
cd $PREVPATH
rm -rf khatru
printf "\n>>> installing relayer\n"
go clean -modcache
git clone https://github.com/mleku/relayer.git && \
  cd relayer && \
  cd examples/basic && \
  time go build && \
  mv basic $HOME/.local/bin/relayer
cd $PREVPATH
rm -rf relayer
printf "\n>>> installing strfry\n"
git clone https://github.com/hoytech/strfry.git && \
  cd strfry && \
  git submodule update --init && \
  make setup-golpe && \
  time make -j1 && \
  mv strfry $HOME/.local/bin/
cd $PREVPATH
rm -rf strfry
printf "\n>>> installing nostr-rs-relay\n"
git clone -q https://git.sr.ht/\~gheartsfield/nostr-rs-relay && \
  cd nostr-rs-relay && \
  time cargo build -q -r --jobs 1 && \
  mv target/release/nostr-rs-relay $HOME/.local/bin/
cd $PREVPATH
rm -rf nostr-rs-relay
printf "\nrun '. %s/.goenv' to configure environment for running Go, optionally add this to your .bashrc (already active now)\n" "$HOME"

