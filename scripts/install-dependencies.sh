unset MSYS_NO_PATHCONV
blueback="\0033[1;37;44m"
resetvid="\0033[0m"
SCRIPT_DIR=$(dirname "$(realpath "$0")")

git config --global core.autocrlf false
git config --global core.longpaths true

echo -e $blueback \## Making the EnergyNetowork hyperledger binaries with transaction priority  $resetvid
echo -e $blueback \## Cloning fabric repository $resetvid
git clone https://github.com/hyperledger/fabric.git
cd fabric/
echo -e $blueback \## checkout fabric-v2.3.0 $resetvid
git checkout tags/v2.3.0
echo -e $blueback \## Applying the EnergyNetowork patch on fabric-v2.3.0 $resetvid
git apply $SCRIPT_DIR/../patches/fabric-energy-network.diff
echo -e $blueback \## Installing gotools to make fabric $resetvid
#make gotools
echo -e $blueback \## "Compiling fabric-v2.3.0 with support for TRANSACTION PRIORITY" $resetvid
#make docker
cd ..


echo -e $blueback \## Showing how EnergyNetowrk compiled a new '"proposal_response.pb.go"' for the fabric and chaincode $resetvid
echo -e $blueback \## Cloning fabric-protos repository $resetvid
git clone https://github.com/hyperledger/fabric-protos.git
cd fabric-protos/
echo -e $blueback \## checkout fabric-protos release-2.1 $resetvid
git checkout release-2.1
echo -e $blueback \## applying EnergyNetowork patch fabric-protos-energy-network.diff $resetvid
git apply $SCRIPT_DIR/../patches/fabric-protos-energy-network.diff
echo -e $blueback \## Building docker that compiles the PROTOS $resetvid
docker build - < ./ci/Dockerfile -t fabric-protos-compilation
echo -e $blueback \## Trying to stop and remove '"fabric-protos-compilation"' container if exists $resetvid
docker stop fabric-protos-compilation
docker container rm fabric-protos-compilation
echo -e $blueback \## Compiling protos to be available at $PWD/build/fabric-protos-go $resetvid
docker run -itd --name fabric-protos-compilation --mount type=bind,source=$PWD,target=/go fabric-protos-compilation sh
docker exec fabric-protos-compilation mkdir -p build/fabric-protos-go
docker exec fabric-protos-compilation ./ci/compile_go_protos.sh
echo -e $blueback \## Stopping and removing container '"fabric-protos-compilation"' $resetvid
docker stop fabric-protos-compilation
docker container rm fabric-protos-compilation
cd ..

echo -e $blueback \## "Installing JAVA dependencies for this machine" $resetvid
$SCRIPT_DIR/install-java-dependencies.sh

echo -e $blueback \## "Building image ubuntu CLI 'energy-network-ubuntu' for applications" $resetvid
$SCRIPT_DIR/build-energy-network-ubuntu-image.sh


echo -e $blueback \## Making the EnergyNetowork "'applications'" jars $resetvid
#git clone https://github.com/johannww/applications.
#cd fabric-gateway-java/
#git checkout tags/v2.2.1
#mvn clean
#mvn install -Dcheckstyle.skip -DskipTests -P release
#cd ..