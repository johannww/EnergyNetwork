#!/bin/bash
SCRIPT_DIR=$(dirname "$(realpath "$0")")
docker build . -f $SCRIPT_DIR/../energy-applications/Dockerfile/Dockerfile -t energy-network-ubuntu