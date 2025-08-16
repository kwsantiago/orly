#!/usr/bin/env bash

echo multi-relay installer - this is written for ubuntu 24.04, may work with debian and other debian based distros. REQUIRES SUDO.
echo "this script must be run as '. `pwd`/install.sh', if not, stop and invoke again with the . in front."
read -p "Press Enter to continue, ctrl-C to stop."
PREVPATH=`pwd`
printf "\n>>> updating apt                            \r"
sudo apt update > /dev/null 2>&1
printf ">>> installing prerequisite deb packages    \r"
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
  libzstd-dev zlib1g-dev \
  > /dev/null 2>&1
printf ">>> installing go environment script        \r"
cp .goenv $HOME/
chmod +x $HOME/.goenv
cd $HOME || exit1
printf ">>> downloading Go                         \r"
wget -nc https://go.dev/dl/go1.25.0.linux-amd64.tar.gz > /dev/null 2>&1
printf ">>> removing previous Go installation       \r"
sudo rm -rf $HOME/go
printf ">>> unpacking Go install archive            \r"
tar xf go1.25.0.linux-amd64.tar.gz
printf ">>> setting environment for Go              \r"
printf ">>> installing strfry                       \n"
echo
cd $PREVPATH || exit
git clone https://github.com/hoytech/strfry.git && \
  cd strfry && \
  git submodule update --init && \
  make setup-golpe && \
  make -j$(nproc) && \
  mv strfry $HOME/.local/bin/ && \
  cd .. && \
  rm -rf strfry
echo
printf ">>> installing khatru basic-badger\n"
echo
. $HOME/.goenv
git clone https://github.com/fiatjaf/khatru.git && \
  cd khatru/examples/basic-badger && \
  go build && \
  mv basic-badger $HOME/.local/bin/khatru && \
  cd ../../.. && \
  rm -rf khatru
echo
printf ">>> installing relayer\n"
echo
git clone https://github.com/mleku/relayer.git && \
  cd relayer && \
  cd examples/basic && \
  go build && \
  mv basic $HOME/.local/bin/relayer && \
  cd ../.. && \
  rm -rf relayer
echo
printf ">>> installing ORLY\n"
echo
cd ../../..
go build && \
  mv orly.dev $HOME/.local/bin/orly && \
  cd $PREVPATH || exit

printf "run '. %s/.goenv' to configure environment for running Go, optionally add this to your .bashrc (already active now)\n" "$HOME"

