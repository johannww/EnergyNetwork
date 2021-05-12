#!/bin/bash
SCRIPT_DIR=$(dirname "$(realpath "$0")")
ARCH="$(uname -m)"
if [[ $ARCH == "x86_64" ]]; then
    ARCH_JAVA_HOME="//usr/lib/jvm/java-13-openjdk-amd64"
else
    ARCH_JAVA_HOME="//usr/lib/jvm/java-13-openjdk-arm64"
fi
export ARCH_JAVA_HOME
envsubst '$ARCH_JAVA_HOME' < $SCRIPT_DIR/../energy-applications/Dockerfile/Dockerfile > $SCRIPT_DIR/Dockerfile
docker build $SCRIPT_DIR/.. -f $SCRIPT_DIR/Dockerfile -t energy-network-ubuntu
rm $SCRIPT_DIR/Dockerfile