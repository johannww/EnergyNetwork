#!/bin/bash
SCRIPT_DIR=$(dirname "$(realpath "$0")")
export BASE_DIR=$(readlink -m $SCRIPT_DIR/..)
mkdir -p bin
wget https://github.com/hyperledger/fabric/releases/download/v2.3.0/hyperledger-fabric-linux-amd64-2.3.0.tar.gz
wget https://github.com/hyperledger/fabric-ca/releases/download/v1.4.9/hyperledger-fabric-ca-linux-amd64-1.4.9.tar.gz

gzip --decompress hyperledger-fabric-linux-amd64-2.3.0.tar.gz
tar -C $BASE_DIR -xvf hyperledger-fabric-linux-amd64-2.3.0.tar
rm -r $BASE_DIR/config
rm hyperledger-fabric-linux-amd64-2.3.0.tar

gzip --decompress hyperledger-fabric-ca-linux-amd64-1.4.9.tar.gz
tar -C $BASE_DIR -xvf hyperledger-fabric-ca-linux-amd64-1.4.9.tar
rm hyperledger-fabric-ca-linux-amd64-1.4.9.tar