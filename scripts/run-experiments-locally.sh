#!/bin/bash
SCRIPT_DIR=$(dirname "$(realpath "$0")")
export BASE_DIR=$(readlink -m $SCRIPT_DIR/..)
unset MSYS_NO_PATHCONV
blueback="\0033[1;37;44m"
resetvid="\0033[0m"

declare -A parsedConfigMeFirst
declare -A parsedTestCfg


testNumber=$(ls -v -1 $BASE_DIR/test-reports | tail -n 1)
testNumber=$((testNumber+1))
testFolder=$BASE_DIR/test-reports/$testNumber
echo -e $blueback \## "Creating test folder $testFolder "   $resetvid
mkdir $testFolder

echo -e $blueback  "Copying 'CONFIG-ME-FIRST.yaml' to $testFolder" $resetvid 
cp $BASE_DIR/CONFIG-ME-FIRST.yaml $testFolder/
echo -e $blueback \## "Reading network params from 'CONFIG-ME-FIRST.yaml' "   $resetvid
configMeFirstText=`cat CONFIG-ME-FIRST.yaml`
numberOfOrgs=`echo "$configMeFirstText" | shyaml get-length organizations`
for  ((org=0; org<$numberOfOrgs; org+=1)); do
    keyValues=( $(echo "$configMeFirstText" | shyaml key-values organizations.$org) )
    keyValueIndex=0
    for keyValue in ${keyValues[@]}; do
        if [ $(($keyValueIndex%2)) == 0 ]; then
            key=$keyValue
        else
            value=$keyValue
            parsedConfigMeFirst[$org,${key%?}]=${value%?}
        fi

        keyValueIndex=$((keyValueIndex+1))
    done
    parsedConfigMeFirst[$org,${key%?}]=${value}
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
            parsedTestCfg[$preKey,${key%?}]=${value%?}
        fi

        keyValueIndex=$((keyValueIndex+1))
    done
    parsedTestCfg[$preKey,${key%?}]=${value}
}

echo -e $blueback  "Copying 'test-configuration.yaml' to $testFolder" $resetvid 
cp $BASE_DIR/test-configuration.yaml $testFolder/
echo -e $blueback \## "Reading configuration params in test-configuration.yaml"   $resetvid 
testParamsText=`cat test-configuration.yaml`
sensorsKeyValues=( $(echo "$testParamsText" | shyaml key-values sensors) )
sellersKeyValues=( $(echo "$testParamsText" | shyaml key-values sellers) )
buyersKeyValues=( $(echo "$testParamsText" | shyaml key-values buyers) )
utilityUrl=( $(echo "$testParamsText" | shyaml get-value utilityurl) )
paymentUrl=( $(echo "$testParamsText" | shyaml get-value paymentcompanyurl) )
auctionInterval=( $(echo "$testParamsText" | shyaml get-value auctioninterval) )
applicationInstancesNumber=( $(echo "$configMeFirstText" | shyaml get-value applications-quantity) )

parseTestConfigMap sensors "${sensorsKeyValues[@]}" 
parseTestConfigMap sellers "${sellersKeyValues[@]}" 
parseTestConfigMap buyers "${buyersKeyValues[@]}" 


#initialize peers and orderers
echo -e $blueback \## "Restarting peers and orderers and measuring their files size "   $resetvid 
echo 'Files size sum (du -hs) result from each container:' > $testFolder/initial-containers-filesystem-sizes.txt
for  ((org=0; org<$numberOfOrgs; org+=1)); do
    orgName=${parsedConfigMeFirst[$org,name]}

    nOrds=${parsedConfigMeFirst[$org,orderer-quantity]}
    for ((i=1; i<=$nOrds; i+=1)); do
        docker restart orderer$i-$orgName 
        docker stats --format "{{.CPUPerc}}:{{.MemUsage}}:{{.NetIO}}:{{.BlockIO}}" orderer$i-$orgName > $testFolder/stats-orderer$i-$orgName.txt &
        echo -n "orderer$i-$orgName: "  >> $testFolder/initial-containers-filesystem-sizes.txt
        docker exec orderer$i-$orgName du -s >> $testFolder/initial-containers-filesystem-sizes.txt
    done

    nPeers=${parsedConfigMeFirst[$org,peer-quantity]}
    for ((i=1; i<=$nPeers; i+=1)); do
        docker restart peer$i-$orgName
        docker stats --format "{{.CPUPerc}}:{{.MemUsage}}:{{.NetIO}}:{{.BlockIO}}" peer$i-$orgName > $testFolder/stats-peer$i-$orgName.txt &
        chaincodeContainerName=$(docker container ls --format "{{.Names}}" | grep "dev-peer$i-$orgName")
        docker stats --format "{{.CPUPerc}}:{{.MemUsage}}:{{.NetIO}}:{{.BlockIO}}" $chaincodeContainerName > $testFolder/stats-chaincode-peer$i-$orgName.txt &
        echo -n "peer$i-$orgName: "  >> $testFolder/initial-containers-filesystem-sizes.txt
        docker exec peer$i-$orgName du -s >> $testFolder/initial-containers-filesystem-sizes.txt
    done
done

echo -e $blueback \## "Restarting 'cli-applications' "   $resetvid
for  ((i=1; i<=$applicationInstancesNumber; i+=1)); do docker restart cli-applications-$i; done

echo -e $blueback \## "measuring ecdsap256 speed - ONLY POSSIBLE IN cli-applications "   $resetvid
docker exec cli-applications-1 openssl speed ecdsap256 > $testFolder/ecdsap256-speed-cli-applications.txt

for  ((i=1; i<=$applicationInstancesNumber; i+=1)); do
    docker stats --format "{{.CPUPerc}}:{{.MemUsage}}:{{.NetIO}}:{{.BlockIO}}" cli-applications-$i > $testFolder/stats-cli-applications-$i.txt &
done
#loggingFlag1="-Djava.util.logging.config.file=commons-logging.properties"
#loggingFlag2="-Dlog4j.configuration=log4j.properties"

echo -e $blueback \## "Starting Utility and PaymentCompany applications "   $resetvid 
#cd energy-applications
#nohup mvn exec:java@utility -Dexec.mainClass="applications.AppUtility" -Dexec.args="-msp UFSC -port 80 --certificate $BASE_DIR/hyperledger/ufsc/admin1/msp/signcerts/cert.pem --privatekey $BASE_DIR/hyperledger/ufsc/admin1/msp/keystore/key.pem"  > $BASE_DIR/test-reports/AppUtility.out 2>&1 &
#nohup mvn exec:java@payment -Dexec.mainClass="applications.AppPaymentCompany" -Dexec.args="-msp UFSC -port 81 --certificate $BASE_DIR/hyperledger/ufsc/admin1/msp/signcerts/cert.pem --privatekey $BASE_DIR/hyperledger/ufsc/admin1/msp/keystore/key.pem" > $BASE_DIR/test-reports/AppPaymentCompany.out 2>&1 &
export MSYS_NO_PATHCONV=1
#docker exec cli-applications bash -c 'nohup mvn exec:java@utility -Dexec.mainClass="applications.AppUtility" -Dexec.args="-msp UFSC -port 80 --certificate /EnergyNetwork/hyperledger/ufsc/admin1/msp/signcerts/cert.pem --privatekey /EnergyNetwork/hyperledger/ufsc/admin1/msp/keystore/key.pem --dockernetwork" '$loggingFlag1' '$loggingFlag2' > /EnergyNetwork/test-reports/'$testNumber'/AppUtility.out 2>&1 &'
#docker exec cli-applications bash -c 'nohup mvn exec:java@payment -Dexec.mainClass="applications.AppPaymentCompany" -Dexec.args="-msp UFSC -port 81 --certificate /EnergyNetwork/hyperledger/ufsc/admin1/msp/signcerts/cert.pem --privatekey /EnergyNetwork/hyperledger/ufsc/admin1/msp/keystore/key.pem --dockernetwork" '$loggingFlag1' '$loggingFlag2' > /EnergyNetwork/test-reports/'$testNumber'/AppPaymentCompany.out 2>&1 &'
unset MSYS_NO_PATHCONV

echo -e $blueback \## "Starting sensors, sellers and buyers applications"   $resetvid 
#nohup mvn exec:java@sensor-test -Dexec.mainClass="applications.AppSensorForTest" -Dexec.args="-msp ${parsedTestCfg[sensors,msp]} --basedir $BASE_DIR --sensors ${parsedTestCfg[sensors,quantity]} --unit ${parsedTestCfg[sensors,unit]} --publishinterval ${parsedTestCfg[sensors,publishinterval]} --publishquantity ${parsedTestCfg[sensors,publishquantity]}" > $BASE_DIR/test-reports/AppSensorForTest.out 2>&1 &
#nohup mvn exec:java@seller-test -Dexec.mainClass="applications.AppSellerForTest" -Dexec.args="-msp ${parsedTestCfg[sellers,msp]}  --basedir $BASE_DIR --sellers ${parsedTestCfg[sellers,quantity]} --publishinterval ${parsedTestCfg[sellers,publishinterval]}  --publishquantity ${parsedTestCfg[sellers,publishquantity]} --paymentcompanyurl $paymentUrl" > $BASE_DIR/test-reports/AppSellerForTest.out 2>&1 &
#nohup mvn exec:java@buyer-test -Dexec.mainClass="applications.AppBuyerForTest" -Dexec.args="-msp ${parsedTestCfg[buyers,msp]} --basedir $BASE_DIR --buyers ${parsedTestCfg[buyers,quantity]} --publishinterval ${parsedTestCfg[buyers,publishinterval]}  --publishquantity ${parsedTestCfg[buyers,publishquantity]} --utilityurl $utilityUrl --paymentcompanyurl $paymentUrl" > $BASE_DIR/test-reports/AppBuyerForTest.out 2>&1 &
export MSYS_NO_PATHCONV=1
for  ((i=1; i<=$applicationInstancesNumber; i+=1)); do
    (docker exec cli-applications-$i bash -c 'nohup mvn exec:java@sensor-test -Dexec.mainClass="applications.AppSensorForTest" -Dexec.args="-msp '${parsedTestCfg[sensors,msp]}' --basedir /EnergyNetwork --sensors '${parsedTestCfg[sensors,quantity]}' --unit '${parsedTestCfg[sensors,unit]}' --publishinterval '${parsedTestCfg[sensors,publishinterval]}' --publishquantity '${parsedTestCfg[sensors,publishquantity]}' --dockernetwork" '$loggingFlag1' '$loggingFlag2' > /EnergyNetwork/test-reports/'$testNumber'/AppSensorForTest.out 2>&1') &
    pidsSensor[$i]=$!
    (docker exec cli-applications-$i bash -c 'nohup mvn exec:java@seller-test -Dexec.mainClass="applications.AppSellerForTest" -Dexec.args="-msp '${parsedTestCfg[sellers,msp]}'  --basedir /EnergyNetwork --sellers '${parsedTestCfg[sellers,quantity]}' --publishinterval '${parsedTestCfg[sellers,publishinterval]}'  --publishquantity '${parsedTestCfg[sellers,publishquantity]}' --paymentcompanyurl '$paymentUrl' --dockernetwork" '$loggingFlag1' '$loggingFlag2' > /EnergyNetwork/test-reports/'$testNumber'/AppSellerForTest.out 2>&1') &
    pidsSeller[$i]=$!
    (docker exec cli-applications-$i bash -c 'nohup mvn exec:java@buyer-test -Dexec.mainClass="applications.AppBuyerForTest" -Dexec.args="-msp '${parsedTestCfg[buyers,msp]}' --basedir /EnergyNetwork --buyers '${parsedTestCfg[buyers,quantity]}' --publishinterval '${parsedTestCfg[buyers,publishinterval]}'  --publishquantity '${parsedTestCfg[buyers,publishquantity]}' --utilityurl '$utilityUrl' --paymentcompanyurl '$paymentUrl' --dockernetwork" '$loggingFlag1' '$loggingFlag2' -Djava.security.egd=file:/dev/./urandom > /EnergyNetwork/test-reports/'$testNumber'/AppBuyerForTest.out 2>&1') &
    pidsBuyer[$i]=$!
done
#pidBuyer=$(docker exec cli-applications bash -c 'nohup mvn exec:java@buyer-test -Dexec.mainClass="applications.AppBuyerForTestX509" -Dexec.args="-msp '${parsedTestCfg[buyers,msp]}' --basedir /EnergyNetwork --buyers '${parsedTestCfg[buyers,quantity]}' --publishinterval '${parsedTestCfg[buyers,publishinterval]}'  --publishquantity '${parsedTestCfg[buyers,publishquantity]}' --utilityurl '$utilityUrl' --paymentcompanyurl '$paymentUrl' --dockernetwork" '$loggingFlag1' '$loggingFlag2' > /EnergyNetwork/test-reports/'$testNumber'/AppBuyerForTestX509.out 2>&1 & echo $!')
#docker exec cli-applications bash -c 'nohup mvn exec:java@buyer-test -Dexec.mainClass="applications.AppBuyerForTest" -Dexec.args="-msp 'IDEMIXORG' --basedir /EnergyNetwork --buyers '1' --publishinterval '5000'  --publishquantity '10' --utilityurl http://localhost --paymentcompanyurl http://localhost:81 --dockernetwork" -Djava.util.logging.config.file=commons-logging.properties -Dlog4j.configuration=log4j.properties'
unset MSYS_NO_PATHCONV

echo -e $blueback \## "Starting PeriodicAuction application"  $resetvid 
#nohup mvn exec:java@auction -Dexec.mainClass="applications.AppPeriodicAuction" -Dexec.args="-msp UFSC --auctioninterval $auctionInterval --certificate $BASE_DIR/hyperledger/ufsc/admin1/msp/signcerts/cert.pem --privatekey $BASE_DIR/hyperledger/ufsc/admin1/msp/keystore/key.pem" > $BASE_DIR/test-reports/AppPeriodicAuction.out 2>&1 &

#nohup mvn exec:java@auction -Dexec.mainClass="applications.AppPeriodicAuction" -Dexec.args="-msp UFSC --auctioninterval 50000 --certificate $BASE_DIR/hyperledger/ufsc/admin1/msp/signcerts/cert.pem --privatekey $BASE_DIR/hyperledger/ufsc/admin1/msp/keystore/key.pem" &

export MSYS_NO_PATHCONV=1
(docker exec cli-applications-1 bash -c 'mvn exec:java@auction -Dexec.mainClass="applications.AppPeriodicAuction" -Dexec.args="-msp UFSC --auctioninterval '$auctionInterval' --certificate /EnergyNetwork/hyperledger/ufsc/admin1/msp/signcerts/cert.pem --privatekey /EnergyNetwork/hyperledger/ufsc/admin1/msp/keystore/key.pem --dockernetwork" '$loggingFlag1' '$loggingFlag2' > /EnergyNetwork/test-reports/'$testNumber'/AppPeriodicAuction.out 2>&1') &
unset MSYS_NO_PATHCONV
#cd ..

#get metrics from peers and orederes metric servers
#wget nos metrics servers

# print system and hardware information
echo -e $blueback \## "Environment characteristics on PHYSICAL MACHINE! "   $resetvid 
cat /proc/cpuinfo > $testFolder/cpuinfo.txt
cat /proc/meminfo > $testFolder/meminfo.txt
df -h > $testFolder/df.txt
echo '$OSTYPE:' $OSTYPE >$testFolder/operating-system.txt
#comandos que garantem preferencia de processos... 

echo -e $blueback \## "Waiting for AppSensorTest, AppSellerTest and AppBuyerTest to finish "   $resetvid 
for  ((i=1; i<=$applicationInstancesNumber; i+=1)); do
    wait ${pidsSensor[$i]}
    wait ${pidsSeller[$i]}
    wait ${pidsBuyer[$i]}
done
echo -e $blueback \## "Applications ended "   $resetvid 

echo -e $blueback \## "Killing containers logging jobs"   $resetvid 
jobs -p | xargs kill

echo -e $blueback \## "final containers sizes "   $resetvid 
for  ((org=0; org<$numberOfOrgs; org+=1)); do
    orgName=${parsedConfigMeFirst[$org,name]}

    nOrds=${parsedConfigMeFirst[$org,orderer-quantity]}
    for ((i=1; i<=$nOrds; i+=1)); do
        echo -n "orderer$i-$orgName: "  >> $testFolder/final-containers-filesystem-sizes.txt
        docker exec orderer$i-$orgName du -hs >> $testFolder/final-containers-filesystem-sizes.txt
    done

    nPeers=${parsedConfigMeFirst[$org,peer-quantity]}
    for ((i=1; i<=$nPeers; i+=1)); do
        echo -n "peer$i-$orgName: "  >> $testFolder/final-containers-filesystem-sizes.txt
        docker exec peer$i-$orgName du -hs >> $testFolder/final-containers-filesystem-sizes.txt
    done
done

echo -e $blueback \## "Plotting graphs to folder test-reports/$testNumber/plots"   $resetvid 
python $SCRIPT_DIR/experimentGraphicCreator.py $BASE_DIR/test-reports/$testNumber