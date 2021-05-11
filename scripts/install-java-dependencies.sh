#!/bin/bash
unset MSYS_NO_PATHCONV
blueback="\0033[1;37;44m"
resetvid="\0033[0m"
SCRIPT_DIR=$(dirname "$(realpath "$0")")

git config --global core.autocrlf false
git config --global core.longpaths true

echo -e $blueback \## Making the fabric-java-sdk-2.2.5 with patch to ease idemix use $resetvid
rm -r fabric-sdk-java
git clone https://github.com/hyperledger/fabric-sdk-java.git
cd fabric-sdk-java/
git checkout tags/v2.2.5
git config core.eol lf
git add --renormalize .
mvn clean
git apply $SCRIPT_DIR/../patches/java-sdk-energy-network.diff
mvn install -DskipTests -P release
cd ..
rm -r fabric-sdk-java


echo -e $blueback \## Making the fabric-java-gateway-2.2.1 with patch to ease idemix use $resetvid
rm -r fabric-gateway-java
git clone https://github.com/hyperledger/fabric-gateway-java.git
cd fabric-gateway-java/
git checkout tags/v2.2.1
mvn clean
git apply $SCRIPT_DIR/../patches/java-gateway-energy-network-tracked.diff
git apply $SCRIPT_DIR/../patches/java-gateway-energy-network-untracked.diff
mvn install -Dcheckstyle.skip -DskipTests -P release
cd ..
rm -r fabric-gateway-java
