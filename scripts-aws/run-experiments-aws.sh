#!/bin/bash
SCRIPT_DIR=$(dirname "$(realpath "$0")")
export BASE_DIR=$(readlink -m $SCRIPT_DIR/..)
unset MSYS_NO_PATHCONV
blueback="\0033[1;37;44m"
resetvid="\0033[0m"

declare -A parsedConfigMeFirst
declare -A parsedTestCfg
declare -A hosts

testNumber=$(ls -v -1 $BASE_DIR/test-reports | tail -n 1)
testNumber=$((testNumber+1))
testFolder=$BASE_DIR/test-reports/$testNumber
echo -e $blueback \## "Creating test folder $testFolder "   $resetvid
mkdir $testFolder

echo -e $blueback  "Copying 'CONFIG-ME-FIRST.yaml' to $testFolder" $resetvid 
cp $BASE_DIR/CONFIG-ME-FIRST.yaml $testFolder/
echo -e $blueback \## "Reading network params from 'CONFIG-ME-FIRST.yaml' "   $resetvid
configMeFirstText=$(cat CONFIG-ME-FIRST.yaml)
numberOfOrgs=$(echo "$configMeFirstText" | shyaml get-length organizations)
for  ((org=0; org<$numberOfOrgs; org+=1)); do
    keyValues=( $(echo "$configMeFirstText" | shyaml key-values organizations.$org | dos2unix) )
    keyValueIndex=0
    for keyValue in ${keyValues[@]}; do
        if [ $(($keyValueIndex%2)) == 0 ]; then
            key=$keyValue
        else
            value=$keyValue
            parsedConfigMeFirst[$org,${key}]=${value}
        fi

        keyValueIndex=$((keyValueIndex+1))
    done
    parsedConfigMeFirst[$org,${key}]=${value}
done

echo -e $blueback  "Loading AWS hosts from $BASE_DIR/aws-hosts.yaml" $resetvid 
hostsText=$(cat aws-hosts.yaml)
unset keyValues
keyValues=$(echo "$hostsText" | shyaml key-values | dos2unix)
for keyValue in ${keyValues[@]}; do
    keyValueIndex=0
    for keyValue in ${keyValues[@]}; do
        if [ $(($keyValueIndex%2)) == 0 ]; then
            key=$keyValue
        else
            value=$keyValue
            hosts[${key}]=${value}
        fi

        keyValueIndex=$((keyValueIndex+1))
    done
    hosts[${key}]=${value}
done

parseTestConfigMap() {
    local preKey=$1
    shift
    local keyValues=$@
    keyValueIndex=0
    for keyValue in ${keyValues[@]}; do
        if [ $(($keyValueIndex%2)) == 0 ]; then
            key=$keyValue
            #echo "KEY:"$keyValue
        else
            value=$keyValue
            #echo "VALUE:"$keyValue
            parsedTestCfg[$preKey,${key}]=${value}
        fi

        keyValueIndex=$((keyValueIndex+1))
    done
    parsedTestCfg[$preKey,${key}]=${value}
}

echo -e $blueback  "Copying 'test-configuration.yaml' to $testFolder" $resetvid 
cp $BASE_DIR/test-configuration.yaml $testFolder/
echo -e $blueback \## "Reading configuration params in test-configuration.yaml"   $resetvid 
testParamsText=`cat test-configuration.yaml`
sensorsKeyValues=( $(echo "$testParamsText" | shyaml key-values sensors | dos2unix) )
sellersKeyValues=( $(echo "$testParamsText" | shyaml key-values sellers | dos2unix) )
buyersKeyValues=( $(echo "$testParamsText" | shyaml key-values buyers | dos2unix) )
utilityUrl=( $(echo "$testParamsText" | shyaml get-value utilityurl | dos2unix) )
paymentUrl=( $(echo "$testParamsText" | shyaml get-value paymentcompanyurl | dos2unix) )
auctionInterval=( $(echo "$testParamsText" | shyaml get-value auctioninterval | dos2unix) )
applicationInstancesNumber=( $(echo "$configMeFirstText" | shyaml get-value applications-quantity | dos2unix) )

parseTestConfigMap sensors "${sensorsKeyValues[@]}" 
parseTestConfigMap sellers "${sellersKeyValues[@]}" 
parseTestConfigMap buyers "${buyersKeyValues[@]}" 

echo -e $blueback  "Copying 'docker-compose-aws.yml' to $testFolder" $resetvid 
cp $BASE_DIR/docker-compose-aws.yml $testFolder/
echo -e $blueback  "Copying 'aws-hosts-instances.yaml' to $testFolder" $resetvid 
cp $BASE_DIR/aws-hosts-instances.yaml $testFolder/
echo -e $blueback  "Copying 'generated-config-aws/configtx.yaml' to $testFolder to store the Batch (Block) configurations" $resetvid 
cp $BASE_DIR/generated-config-aws/configtx.yaml $testFolder/


sshCmd() {
    local host=$1
    shift 
    ssh -o "StrictHostKeyChecking=no" -o "UserKnownHostsFile=/dev/null" -o "ServerAliveInterval 60" -i $SCRIPT_DIR/EnergyNetworkAwsKeyPair.pem ubuntu@$host "$@"  
}

sshCmdBg() {
    local host=$1
    shift 
    ssh -o "StrictHostKeyChecking=no" -o "UserKnownHostsFile=/dev/null" -o "ServerAliveInterval 60" -i $SCRIPT_DIR/EnergyNetworkAwsKeyPair.pem ubuntu@$host "$@" &
}

#initialize peers and orderers
echo -e $blueback \## "Restarting peers and orderers and measuring their files size "   $resetvid 
echo 'Files size sum (du / -s) result from each container:' > $testFolder/initial-containers-filesystem-sizes.txt
for  ((org=0; org<$numberOfOrgs; org+=1)); do
    orgName=${parsedConfigMeFirst[$org,name]}

    nOrds=${parsedConfigMeFirst[$org,orderer-quantity]}
    for ((i=1; i<=$nOrds; i+=1)); do
        echo -n "orderer$i-$orgName: "  >> $testFolder/initial-containers-filesystem-sizes.txt
        (sshCmd ${hosts[orderer$i-$orgName]} 'bash' << EOF
            docker restart orderer$i-$orgName 
            docker exec orderer$i-$orgName du / -s
EOF
) >> $testFolder/initial-containers-filesystem-sizes.txt

        sshCmdBg ${hosts[orderer$i-$orgName]} docker stats --format "{{.CPUPerc}}:{{.MemUsage}}:{{.NetIO}}:{{.BlockIO}}" orderer$i-$orgName > $testFolder/stats-orderer$i-$orgName.txt
        if [[ $org == 0 && $i == 1 ]]; then 
            echo 'start-time: '$(date +"%T.%N") > $testFolder/test-start-and-finish.txt
            startTimestamp=$(date +%s)
        fi
    done

    nPeers=${parsedConfigMeFirst[$org,peer-quantity]}
    for ((i=1; i<=$nPeers; i+=1)); do
        echo -n "peer$i-$orgName: "  >> $testFolder/initial-containers-filesystem-sizes.txt
        (sshCmd ${hosts[peer$i-$orgName]} 'bash' << EOF
            docker restart peer$i-$orgName
            docker exec peer$i-$orgName du / -s
EOF
) >> $testFolder/initial-containers-filesystem-sizes.txt

        sshCmdBg ${hosts[peer$i-$orgName]} docker stats --format "{{.CPUPerc}}:{{.MemUsage}}:{{.NetIO}}:{{.BlockIO}}" peer$i-$orgName > $testFolder/stats-peer$i-$orgName.txt
        while true; do
            echo -e $blueback "Waiting for peer$i-$orgName chaincode init" $resetvid
            chaincodeContainerName=$(sshCmd ${hosts[peer$i-$orgName]} docker container ls --format "{{.Names}}" | grep "dev-peer$i-$orgName" )
            [[ $chaincodeContainerName == "" ]] || break
            sleep 1s
        done
        sshCmdBg ${hosts[peer$i-$orgName]} docker stats --format "{{.CPUPerc}}:{{.MemUsage}}:{{.NetIO}}:{{.BlockIO}}" $chaincodeContainerName > $testFolder/stats-chaincode-peer$i-$orgName.txt
    done
done

echo -e $blueback \## "Restarting 'cli-applications' "   $resetvid
for  ((i=1; i<=$applicationInstancesNumber; i+=1)); do 
    sshCmd ${hosts[application$i]} << EOF
        docker restart cli-applications
        mkdir /home/ubuntu/EnergyNetwork/test-reports/$testNumber
EOF
done

echo -e $blueback \## "measuring ecdsap256 speed - ONLY POSSIBLE IN cli-applications "   $resetvid
sshCmd ${hosts[application1]} docker exec cli-applications openssl speed ecdsap256 > $testFolder/ecdsap256-speed-cli-applications.txt

for  ((i=1; i<=$applicationInstancesNumber; i+=1)); do
    sshCmdBg ${hosts[application$i]} 'docker stats --format "{{.CPUPerc}}:{{.MemUsage}}:{{.NetIO}}:{{.BlockIO}}" cli-applications' > $testFolder/stats-cli-applications-$i.txt
done

loggingFlag1="-Djava.util.logging.config.file=commons-logging.properties"
loggingFlag2="-Dlog4j.configuration=log4j.properties"

#echo -e $blueback \## "Starting Utility and PaymentCompany applications "   $resetvid 

export MSYS_NO_PATHCONV=1
unset MSYS_NO_PATHCONV

echo -e $blueback \## "Starting sensors, sellers and buyers applications"   $resetvid 
export MSYS_NO_PATHCONV=1

for  ((i=1; i<=$applicationInstancesNumber; i+=1)); do

    sshCmdBg ${hosts[application$i]} docker exec cli-applications bash -c "'"'java '$loggingFlag1' '$loggingFlag2' -jar target/sensor-jar-with-dependencies.jar -msp '${parsedTestCfg[sensors,msp]}' --basedir /EnergyNetwork --sensors '${parsedTestCfg[sensors,quantity]}' --unit 0 --publishinterval '${parsedTestCfg[sensors,publishinterval]}' --publishquantity 0 --awsnetwork  > /EnergyNetwork/test-reports/'$testNumber'/AppSensorForTest'$i'.out 2>&1'"'"
    pidsSensor[$i]=$!

done

echo -e $blueback \## "Waiting for sensors declaring themselves ACTIVE"   $resetvid 
for  ((i=1; i<=$applicationInstancesNumber; i+=1)); do
    wait ${pidsSensor[$i]} && echo "Sensor from application$i declared themselves active"
done

for  ((i=1; i<=$applicationInstancesNumber; i+=1)); do

    sshCmdBg ${hosts[application$i]} docker exec cli-applications bash -c "'"'java '$loggingFlag1' '$loggingFlag2' -jar target/sensor-jar-with-dependencies.jar -msp '${parsedTestCfg[sensors,msp]}' --basedir /EnergyNetwork --sensors '${parsedTestCfg[sensors,quantity]}' --unit '${parsedTestCfg[sensors,unit]}' --publishinterval '${parsedTestCfg[sensors,publishinterval]}' --publishquantity '${parsedTestCfg[sensors,publishquantity]}' --awsnetwork --committimeout 60 > /EnergyNetwork/test-reports/'$testNumber'/AppSensorForTest'$i'.out 2>&1'"'"
    pidsSensor[$i]=$!

    sshCmdBg ${hosts[application$i]} docker exec cli-applications bash -c "'"'java '$loggingFlag1' '$loggingFlag2' -jar target/seller-jar-with-dependencies.jar -msp '${parsedTestCfg[sellers,msp]}'  --basedir /EnergyNetwork --sellers '${parsedTestCfg[sellers,quantity]}' --publishinterval '${parsedTestCfg[sellers,publishinterval]}'  --publishquantity '${parsedTestCfg[sellers,publishquantity]}' --paymentcompanyurl '$paymentUrl' --awsnetwork > /EnergyNetwork/test-reports/'$testNumber'/AppSellerForTest'$i'.out 2>&1'"'"
    pidsSeller[$i]=$!

    sshCmdBg ${hosts[application$i]} docker exec cli-applications bash -c "'"'java '$loggingFlag1' '$loggingFlag2' -Djava.security.egd=file:/dev/./urandom -jar target/buyer-jar-with-dependencies.jar -msp '${parsedTestCfg[buyers,msp]}' --basedir /EnergyNetwork --buyers '${parsedTestCfg[buyers,quantity]}' --publishinterval '${parsedTestCfg[buyers,publishinterval]}'  --publishquantity '${parsedTestCfg[buyers,publishquantity]}' --utilityurl '$utilityUrl' --paymentcompanyurl '$paymentUrl' --awsnetwork > /EnergyNetwork/test-reports/'$testNumber'/AppBuyerForTest'$i'.out 2>&1'"'"
    pidsBuyer[$i]=$!
done

unset MSYS_NO_PATHCONV

echo -e $blueback \## "Starting PeriodicAuction application"  $resetvid 

export MSYS_NO_PATHCONV=1
auctionCallerMsp=${parsedTestCfg[sensors,msp],,}
sshCmdBg ${hosts[application1]} docker exec cli-applications bash -c "'"'java '$loggingFlag1' '$loggingFlag2' -jar target/auction-jar-with-dependencies.jar -msp '${auctionCallerMsp^^}' --auctioninterval '$auctionInterval' --certificate /EnergyNetwork/hyperledger/'$auctionCallerMsp'/admin1/msp/signcerts/cert.pem --privatekey /EnergyNetwork/hyperledger/'$auctionCallerMsp'/admin1/msp/keystore/key.pem --awsnetwork > /EnergyNetwork/test-reports/'$testNumber'/AppPeriodicAuction.out 2>&1'"'"
unset MSYS_NO_PATHCONV
#cd ..
#get metrics from peers and orederes metric servers
#wget nos metrics servers

echo -e $blueback \## "Waiting for AppSensorTest, AppSellerTest and AppBuyerTest to finish "   $resetvid 
for  ((i=1; i<=$applicationInstancesNumber; i+=1)); do
    wait ${pidsSensor[$i]} && echo "Sensor application$i ended"
    wait ${pidsSeller[$i]} && echo "Seller application$i ended"
    wait ${pidsBuyer[$i]} && echo "Buyer application$i ended"
done
echo 'end-time: '$(date +"%T.%N") >> $testFolder/test-start-and-finish.txt
testDurationSec=$((`date +%s` - startTimestamp))
echo -e $blueback \## "Applications ended "   $resetvid

echo -e $blueback \## "Test start and finish times in 'test-start-and-finish.txt' "   $resetvid

echo -e $blueback \## "Killing containers logging jobs"   $resetvid 
jobs -p | xargs kill

echo -e $blueback \## "Writing the chaincode function average times to chaincode-averages.json "   $resetvid
sshCmdBg ${hosts[application1]} docker exec cli-applications bash -c "'"'java '$loggingFlag1' '$loggingFlag2' -jar target/chaincode-metrics-jar-with-dependencies.jar -msp '${auctionCallerMsp^^}' --auctioninterval '$auctionInterval' --certificate /EnergyNetwork/hyperledger/'$auctionCallerMsp'/admin1/msp/signcerts/cert.pem --privatekey /EnergyNetwork/hyperledger/'$auctionCallerMsp'/admin1/msp/keystore/key.pem --awsnetwork'"'" | sed '/WARNING*/d' > $testFolder/chaincode-averages.json

echo -e $blueback \## "Stopping periodic auction "   $resetvid 
sshCmd ${hosts[application1]} docker stop cli-applications


echo -e $blueback \## "final containers sizes "   $resetvid 
echo -e $blueback \## "Downloading peers and orderers logs "   $resetvid
mkdir -p $testFolder/logs-orderers
mkdir -p $testFolder/logs-peers
for  ((org=0; org<$numberOfOrgs; org+=1)); do
    orgName=${parsedConfigMeFirst[$org,name]}

    nOrds=${parsedConfigMeFirst[$org,orderer-quantity]}
    for ((i=1; i<=$nOrds; i+=1)); do
        sshCmd ${hosts[orderer$i-$orgName]} docker logs --since "$testDurationSec"s orderer$i-$orgName 2> $testFolder/logs-orderers/log-orderer$i-$orgName.txt &
        echo -n "orderer$i-$orgName: "  >> $testFolder/final-containers-filesystem-sizes.txt
        sshCmd ${hosts[orderer$i-$orgName]} docker exec orderer$i-$orgName du / -s >> $testFolder/final-containers-filesystem-sizes.txt
    done

    nPeers=${parsedConfigMeFirst[$org,peer-quantity]}
    for ((i=1; i<=$nPeers; i+=1)); do
        sshCmd ${hosts[peer$i-$orgName]} docker logs --since "$testDurationSec"s peer$i-$orgName 2> $testFolder/logs-peers/log-peer$i-$orgName.txt &
        echo -n "peer$i-$orgName: "  >> $testFolder/final-containers-filesystem-sizes.txt
        sshCmd ${hosts[peer$i-$orgName]} docker exec peer$i-$orgName du / -s >> $testFolder/final-containers-filesystem-sizes.txt
    done
done

echo -e $blueback \## "Downloading applications logs "   $resetvid
mkdir -p $testFolder/logs-applications
for  ((i=1; i<=$applicationInstancesNumber; i+=1)); do
    mkdir -p $testFolder/logs-applications/application$i
    scp -o "StrictHostKeyChecking=no" -o "UserKnownHostsFile=/dev/null" -i $SCRIPT_DIR/EnergyNetworkAwsKeyPair.pem ubuntu@${hosts[application$i]}:/home/ubuntu/EnergyNetwork/test-reports/$testNumber/* $testFolder/logs-applications/application$i &
done

echo -e $blueback \## "Plotting graphs to folder test-reports/$testNumber/plots"   $resetvid 
python3 $SCRIPT_DIR/../scripts/experimentGraphicCreator.py $BASE_DIR/test-reports/$testNumber

echo -e $blueback \## "Waiting for logs" $resetvid
wait

echo -e $blueback \## "Creating 'annotations.txt' if you desire to write interpretations for this experiment round" $resetvid
touch $testFolder/annotations.txt

echo -e $blueback \## "Done!" $resetvid