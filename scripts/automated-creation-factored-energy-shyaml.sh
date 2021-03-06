#!/bin/bash
SCRIPT_DIR=$(dirname "$(realpath "$0")")
export BASE_DIR=$(readlink -m $SCRIPT_DIR/..)
export PATH=${BASE_DIR}/bin:$PATH
export COMPOSE_PROJECT_NAME="fabric"
unset MSYS_NO_PATHCONV
export MSYS_NO_PATHCONV
export FABRIC_CA_CLIENT_MSPDIR=.
export BINDABLE_PORT=7000
currentUser=$USER
blueback="\0033[1;37;44m"
resetvid="\0033[0m"

print_usage() {
 printf "Usage: pass '-e' to NOT register and enroll with the CAs, otherwise all credentials will be enrolled with the CAs"
}

registerAndEnroll='true'
while getopts 'e' flag; do
  case "${flag}" in
    e) registerAndEnroll='false' ;;
    *) print_usage
       exit 1 ;;
  esac
done


findOrgIndexByName () {
    for ((i=0; i<$numberOfOrgs; i+=1)); do
        if [ $1 = ${matrix[$i,name]} ]
        then
            echo $i
        fi
    done
}

docker-compose -f docker-compose.yml down --remove-orphans
#chown -R labsec:labsec hyperledger/
#find hyperledger/ -type f ! \( -iname "*.yaml" -or -iname "*.go" -or -iname "*.mod" -or -iname "*.sum" \) -delete
#find hyperledger/ -type l ! \( -iname "*.yaml" -or -iname "*.go" -or -iname "*.mod" -or -iname "*.sum" \) -delete
#rm -r `find hyperledger/ -name couchdb -type d`
#rm -r `find hyperledger/ -name couchdb_config -type d`


rm -r hyperledger
mkdir hyperledger


#
# Calling python script that reads CONFIG-ME-FIRST
# AND GENERATES A configtx.yaml WITH THE ORGANIZATIONS
# AND RAFT CONSENTERS INCLUDED. 
# IT ALSO READS THE 
# Located in "config-template/configtxTemplate.yaml"
#
python3 $BASE_DIR/scripts/partialConfigtxGenerator.py "$BASE_DIR"


declare -A matrix
declare -A orgsRootCAPorts
declare -A orgsOrdPorts
declare -A orgsPeerPorts

echo -e $blueback \##Parsing CONFIG-ME-FIRST.yaml with 'shyaml' $resetvid
configMeFirstText=`cat CONFIG-ME-FIRST.yaml`
numberOfOrgs=`echo "$configMeFirstText" | shyaml get-length organizations`
for  ((org=0; org<$numberOfOrgs; org+=1)); do
    keyValues=( $(echo "$configMeFirstText" | shyaml key-values organizations.$org | dos2unix) )
    keyValueIndex=0
    for keyValue in ${keyValues[@]}; do
        if [ $(($keyValueIndex%2)) == 0 ]; then
            key=$keyValue
        else
            value=$keyValue
            matrix[$org,${key}]=${value}
        fi

        keyValueIndex=$((keyValueIndex+1))
    done
    matrix[$org,${key}]=${value}
    #EXIT if an organization does not have any admin
    if (( ${matrix[$org,admin-quantity]} < 1 )); then
        echo -e $blueback \##Organization ${matrix[$org,name]} must have AT LEAST 1 admin $resetvid
        exit 1
    fi
done
applicationInstancesNumber=( $(echo "$configMeFirstText" | shyaml get-value applications-quantity) )

#
# GENERATING THE TLS CERTIFICATE AUTHORITIY
# ONE FOR EVERYONE
#
echo -e $blueback \##Configuring the CA-TLS $resetvid
echo -e $blueback \# Turning on CA-TLS$resetvid
docker-compose -f docker-compose.yml up -d ca-tls
sleep 2s
docker logs ca-tls
echo -e $blueback \# "Changing POST-INITIALIZE mounted directory file ownership " $resetvid
sudo chown -R $currentUser:$currentUser ${BASE_DIR}/hyperledger/tls-ca
echo -e $blueback \# Downloading CA-TLS admin certificate $resetvid
PATH_CERT_ADM_TLS=${BASE_DIR}/hyperledger/tls-ca/crypto/ca-cert.pem
PATH_MSP_ADM_TLS=${BASE_DIR}/hyperledger/tls-ca/admin/msp
export FABRIC_CA_CLIENT_TLS_CERTFILES=$PATH_CERT_ADM_TLS
export FABRIC_CA_CLIENT_HOME=$PATH_MSP_ADM_TLS
fabric-ca-client enroll -u https://tls-ca-admin:tls-ca-adminpw@0.0.0.0:7052 

#
# GENERATING THE ROOT CERTIFICATE AUTHORITIES
# ONE FOR EACH ORGANIZATION
#
echo -e $blueback \## Configuring organization ROOT CAs  $resetvid
for  ((l=0; l<$numberOfOrgs; l+=1)); do
    export ORG_NAME=${matrix[$l,name]}
    export BINDABLE_PORT
    echo -e $blueback \# "Changing RCA service name in file 'docker-compose.yml'" $resetvid
    perl -pi -e 's/rca:/rca-'$ORG_NAME':/g' docker-compose.yml
    sleep 1s
    echo -e $blueback \# pre-initializing RCA-$ORG_NAME $resetvid
    export INIT_OR_START="init"
    docker-compose -f docker-compose.yml up -d rca-$ORG_NAME
    sleep 1s
    echo -e $blueback \# "Changing mounted directory file ownership " $resetvid
    sudo chown -R $currentUser:$currentUser $BASE_DIR/hyperledger/$ORG_NAME
    echo -e $blueback \# Editing RCA-$ORG_NAME fabric-ca-server-config.yaml affiliations $resetvid
    python3 $BASE_DIR/scripts/editRootCaAfiiliations.py "$ORG_NAME" "$BASE_DIR/hyperledger/$ORG_NAME/ca/crypto/"
    echo -e $blueback \# Initializing RCA-$ORG_NAME $resetvid
    export INIT_OR_START="start"
    docker-compose -f docker-compose.yml up -d rca-$ORG_NAME
    perl -pi -e 's/rca-'$ORG_NAME':/rca:/g' docker-compose.yml
    sleep 1s
    docker logs rca-$ORG_NAME
    echo -e $blueback \# "Changing POST-INITIALIZE mounted directory file ownership " $resetvid
    sudo chown -R $currentUser:$currentUser $BASE_DIR/hyperledger/$ORG_NAME
    echo -e $blueback \# Configuring RCA-$ORG_NAME $resetvid
    export FABRIC_CA_CLIENT_TLS_CERTFILES=${BASE_DIR}/hyperledger/$ORG_NAME/ca/crypto/ca-cert.pem
    export FABRIC_CA_CLIENT_HOME=${BASE_DIR}/hyperledger/$ORG_NAME/ca/admin/msp
    echo -e $blueback \# Creating RCA-$ORG_NAME "admin's" certificate $resetvid
    fabric-ca-client enroll -u https://rca-$ORG_NAME-admin:rca-$ORG_NAME-adminpw@0.0.0.0:$BINDABLE_PORT
    orgsRootCAPorts[$ORG_NAME]=$BINDABLE_PORT
    BINDABLE_PORT=$(($BINDABLE_PORT+1))
done


#
# REGISTERING ADMINS, CLIENTS, ORDERERS AND PEERS
# IN THEIR RESPECTIVE CERTIFICATE AUTHORITY
#
registerByRole () {
    orgName=$1
    role=$2
    roleAndNumber=$3
    idAttrs=$4
    
    if [ $role != "admin" ] && [ $role != "peer" ] && [ $role != "orderer" ]; then
        orgUnit="client"
    else
        orgUnit=$role
    fi

    export FABRIC_CA_CLIENT_TLS_CERTFILES=$PATH_CERT_ADM_TLS
    export FABRIC_CA_CLIENT_HOME=$PATH_MSP_ADM_TLS
    fabric-ca-client register --id.name $roleAndNumber-$orgName --id.secret $roleAndNumber-$orgName  --id.type $orgUnit -u https://0.0.0.0:7052
    export FABRIC_CA_CLIENT_TLS_CERTFILES=${BASE_DIR}/hyperledger/$orgName/ca/crypto/ca-cert.pem
    export FABRIC_CA_CLIENT_HOME=${BASE_DIR}/hyperledger/$orgName/ca/admin/msp
    fabric-ca-client register --id.name $roleAndNumber-$orgName --id.secret $roleAndNumber-$orgName  --id.type $orgUnit --id.affiliation $orgName --id.attrs "$idAttrs" -u https://0.0.0.0:${orgsRootCAPorts[$orgName]}

}

if [ $registerAndEnroll != "false" ]; then
    echo -e $blueback \##Registering ALL certificates $resetvid
    for  ((l=0; l<$numberOfOrgs; l+=1)); do
        orgName=${matrix[$l,name]}
        echo -e $blueback \##Registering admins $orgName certificates $resetvid
        nAdms=${matrix[$l,admin-quantity]}
        for ((i=1; i<=$nAdms; i+=1)); do
            registerByRole $orgName "admin" "admin$i" '"hf.Registrar.Roles=client,peer,orderer",hf.Registrar.Attributes=*,hf.Revoker=true,hf.GenCRL=true,admin=true:ecert,energy.admin=true:ecert,energy.init=true:ecert,energy.paymentcompany=true:ecert,energy.utility=true:ecert,role=2' &
        done

        echo -e $blueback \##Registering clients $orgName certificates $resetvid
        nClients=${matrix[$l,client-quantity]}
        for ((i=1; i<=$nClients; i+=1)); do
        registerByRole $orgName "client" "client$i" "" &
        done

        echo -e $blueback \##Registering orderers $orgName certificates $resetvid
        nOrds=${matrix[$l,orderer-quantity]}
        for ((i=1; i<=$nOrds; i+=1)); do
            registerByRole $orgName "orderer" "orderer$i" "" &
        done

        echo -e $blueback \##Registering peers $orgName certificates $resetvid
        nPeers=${matrix[$l,peer-quantity]}
        for ((i=1; i<=$nPeers; i+=1)); do
            registerByRole $orgName "peer" "peer$i" "" &
        done

        echo -e $blueback \##Registering buyers $orgName certificates $resetvid
        nBuyers=${matrix[$l,buyer-quantity]}
        for ((i=1; i<=$nBuyers; i+=1)); do
            registerByRole $orgName "buyer" "buyer$i" 'energy.buyer=true:ecert' &
        done

        echo -e $blueback \##Registering sellers $orgName certificates $resetvid
        nSellers=${matrix[$l,seller-quantity]}
        for ((i=1; i<=$nSellers; i+=1)); do
            registerByRole $orgName "seller" "seller$i" 'energy.seller=true:ecert' &
        done

        echo -e $blueback \##Registering sensors $orgName certificates $resetvid
        nSensors=${matrix[$l,sensor-quantity]}
        for ((i=1; i<=$nSensors; i+=1)); do
            xRandom=$((1 + $RANDOM % 100))
            yRandom=$((1 + $RANDOM % 100))
            zRandom=$((1 + $RANDOM % 100))
            radius=1000 #$((40 + $RANDOM % 80))
            registerByRole $orgName "sensor" "sensor$i" "energy.sensor=true:ecert,energy.x=$xRandom:ecert,energy.y=$yRandom:ecert,energy.z=$zRandom:ecert,energy.radius=$radius:ecert" &
        done

    done
fi

echo -e $blueback \##"Waiting for certificates register" $resetvid
wait

#
# ENROLLING ADMINS, CLIENTS, ORDERERS AND PEERS
# AND SAVING THE EVERYTHING IN THEIR MSP FOLDER
#

enrollByRole () {
    orgName=$1
    orgNameUpper=$2
    role=$3
    roleAndNumber=$4
    enrollType=$5

    export FABRIC_CA_CLIENT_HOME=${BASE_DIR}/hyperledger/$orgName/$roleAndNumber/tls-msp
    export FABRIC_CA_CLIENT_TLS_CERTFILES=${BASE_DIR}/hyperledger/tls-ca/crypto/ca-cert.pem
    fabric-ca-client enroll -u https://$roleAndNumber-$orgName:$roleAndNumber-$orgName@0.0.0.0:7052 --enrollment.profile tls --csr.hosts $roleAndNumber-$orgName
    cd ${BASE_DIR}/hyperledger/$orgName/$roleAndNumber/tls-msp/keystore
    ln -s *_sk key.pem
    cd ${BASE_DIR}/
        
    export FABRIC_CA_CLIENT_HOME=${BASE_DIR}/hyperledger/$orgName/$roleAndNumber/msp
    export FABRIC_CA_CLIENT_TLS_CERTFILES=${BASE_DIR}/hyperledger/$orgName/ca/crypto/ca-cert.pem
    certNames="C=BR,ST=SC,L=Florianopolis,O=$orgNameUpper"
    fabric-ca-client enroll --csr.names $certNames -u https://$roleAndNumber-$orgName:$roleAndNumber-$orgName@0.0.0.0:${orgsRootCAPorts[$orgName]} --enrollment.type $enrollType
    cd ${BASE_DIR}/hyperledger/$orgName/$roleAndNumber/msp/keystore
    ln -s *_sk key.pem
    cd ${BASE_DIR}/

    if [ $role != "admin" ]; then
        mkdir -p ${BASE_DIR}/hyperledger/$orgName/$roleAndNumber/msp/admincerts
        #salva certificados admin no peer
        cp -a ${BASE_DIR}/hyperledger/$orgName/admin1/msp/admincerts/. ${BASE_DIR}/hyperledger/$orgName/$roleAndNumber/msp/admincerts/
    fi

    if [ $enrollType == "idemix" ]; then
        mkdir -p ${BASE_DIR}/hyperledger/$orgName/$roleAndNumber/msp/msp
        mv ${BASE_DIR}/hyperledger/$orgName/$roleAndNumber/msp/IssuerPublicKey ${BASE_DIR}/hyperledger/$orgName/$roleAndNumber/msp/msp/IssuerPublicKey
        mv ${BASE_DIR}/hyperledger/$orgName/$roleAndNumber/msp/IssuerRevocationPublicKey ${BASE_DIR}/hyperledger/$orgName/$roleAndNumber/msp/msp/RevocationPublicKey
        cp ${BASE_DIR}/hyperledger/$orgName/$roleAndNumber/msp/user/SignerConfig ${BASE_DIR}/hyperledger/$orgName/$roleAndNumber/msp/user/SignerConfig.json
        echo -e $blueback \## "converting credentials from JSON to PROTO, because the 'peer lifecycle invoke' functions require .proto" $resetvid
        java -jar idemixJsonToProto.jar -i ${BASE_DIR}/hyperledger/$orgName/$roleAndNumber/msp/user/SignerConfig
    fi
}
if [ $registerAndEnroll != "false" ]; then
    echo -e $blueback \##Downloading ALL certificates $resetvid
    for  ((l=0; l<$numberOfOrgs; l+=1)); do
        orgName=${matrix[$l,name]}
        orgNameUpper=${matrix[$l,name]^^}
        echo -e $blueback \##Downloading admins $orgName certificates $resetvid
        #Enrollment type
        enrollType=${matrix[$l,msptype]}


        nAdms=${matrix[$l,admin-quantity]}
        for ((i=1; i<=$nAdms; i+=1)); do
            enrollByRole $orgName $orgNameUpper "admin" "admin$i" $enrollType
        done

        echo -e $blueback \##Creating folder "admincerts" to admins of $orgName organization $resetvid
        nAdms=${matrix[$l,admin-quantity]}
        for ((i=1; i<=$nAdms; i+=1)); do
            mkdir -p ${BASE_DIR}/hyperledger/$orgName/admin$i/msp/admincerts
            #save admins certificates in every admin
            find hyperledger/ -type d -regex ".*/$orgName/admin[0-9]+/msp/signcerts" | while read path; do nomeAdm=${path%/msp*}; nomeAdm=${nomeAdm#*${orgName}/} ; cp ${BASE_DIR}/$path/cert.pem ${BASE_DIR}/hyperledger/$orgName/admin$i/msp/admincerts/$nomeAdm-$orgName-cert.pem; done
        done

        echo -e $blueback \##Downloading clients $orgName certificates $resetvid
        nClients=${matrix[$l,client-quantity]}
        for ((i=1; i<=$nClients; i+=1)); do
            enrollByRole $orgName $orgNameUpper "client" "client$i" $enrollType &
        done


        echo -e $blueback \##Downloading orderers $orgName certificates $resetvid
        nOrds=${matrix[$l,orderer-quantity]}
        for ((i=1; i<=$nOrds; i+=1)); do
            enrollByRole $orgName $orgNameUpper "orderer" "orderer$i" $enrollType &
        done

        echo -e $blueback \##Downloading peers $orgName certificates $resetvid
        nPeers=${matrix[$l,peer-quantity]}
        for ((i=1; i<=$nPeers; i+=1)); do
            enrollByRole $orgName $orgNameUpper "peer" "peer$i" $enrollType &
        done

        echo -e $blueback \##Downloading buyers $orgName certificates $resetvid
        nBuyers=${matrix[$l,buyer-quantity]}
        for ((i=1; i<=$nBuyers; i+=1)); do
            enrollByRole $orgName $orgNameUpper "buyer" "buyer$i" $enrollType &
        done

        echo -e $blueback \##Downloading sellers $orgName certificates $resetvid
        nSellers=${matrix[$l,seller-quantity]}
        for ((i=1; i<=$nSellers; i+=1)); do
            enrollByRole $orgName $orgNameUpper "seller" "seller$i" $enrollType &
        done

        echo -e $blueback \##Downloading sensors $orgName certificates $resetvid
        nSensors=${matrix[$l,sensor-quantity]}
        for ((i=1; i<=$nSensors; i+=1)); do
            enrollByRole $orgName $orgNameUpper "sensor" "sensor$i" $enrollType &
        done
    done
fi

echo -e $blueback \##"Waiting for certificates enroll" $resetvid
wait

#
# GENERATING ORGS INSTITUTIONAL MSP FOLDER
# THIS IS USEFUL FOR THE GENESIS BLOCK FOR THE NETWORK CHANNEL
# SET THE MSPFOLDER IN THE Field "MSPDir" IN THE FILE "config.tx" to 
# THE DIRECTORY I WILL CREATE BELOW
#
echo -e $blueback \## "creating MSP folders for every organization" $resetvid
for  ((l=0; l<$numberOfOrgs; l+=1)); do
    orgName=${matrix[$l,name]}
    enrollType=${matrix[$l,msptype]}

    #getting all admin certificates
    nAdms=${matrix[$l,admin-quantity]}
    mkdir -p ${BASE_DIR}/hyperledger/$orgName/msp/admincerts/
    for ((i=1; i<=nAdms; i+=1)); do
        cp ${BASE_DIR}/hyperledger/$orgName/admin$i/msp/signcerts/cert.pem ${BASE_DIR}/hyperledger/$orgName/msp/admincerts/$orgName-admin$i-cert.pem
    done
    mkdir -p ${BASE_DIR}/hyperledger/$orgName/msp/cacerts/
    cp ${BASE_DIR}/hyperledger/$orgName/ca/crypto/ca-cert.pem ${BASE_DIR}/hyperledger/$orgName/msp/cacerts/$orgName-rca-cert.pem
    mkdir -p ${BASE_DIR}/hyperledger/$orgName/msp/tlscacerts
    cp ${BASE_DIR}/hyperledger/tls-ca/crypto/ca-cert.pem ${BASE_DIR}/hyperledger/$orgName/msp/tlscacerts/tls-ca-cert.pem
    #copying the OU configuration file in "generated-config/config.yaml"
    cp ${BASE_DIR}/config-template/config.yaml ${BASE_DIR}/hyperledger/$orgName/msp/config.yaml

    #IF org uses IDEMIX, then copy "IssuerPublicKey" and "IssuerRevocationPublicKey"
    if [ $enrollType == "idemix" ]; then
        mkdir -p ${BASE_DIR}/hyperledger/$orgName/msp/msp/
        cp ${BASE_DIR}/hyperledger/$orgName/ca/crypto/IssuerPublicKey ${BASE_DIR}/hyperledger/$orgName/msp/msp/
        cp ${BASE_DIR}/hyperledger/$orgName/ca/crypto/IssuerRevocationPublicKey ${BASE_DIR}/hyperledger/$orgName/msp/msp/RevocationPublicKey
    fi 
done


#
# CREATING THE GENESIS BLOCK AND COPYING IT TO EVERY ORDERER
#
#
###read -p "Change the generated-config/configtx.yaml as wished and press ENTER to create the syschannel genesis block"
###read -p $"Type the desired Profile name. Ex:  SampleMultiMSPRaft, SampleSingleMSPSolo, SampleSingleMSPKafka, etc: " profile
sysChannelProfile="SampleMultiMSPRaft"

echo -e $blueback \# Gerando bloco genesis para syschannel -- NAO PRECISA NA VERSAO 2.3 $resetvid
configtxgen -configPath $BASE_DIR/generated-config -profile $sysChannelProfile -outputBlock ${BASE_DIR}/hyperledger/tempgenesis.block -channelID syschannel
find hyperledger/ -type d -regex ".*/orderer[0-9]+" | while read path; do cp ${BASE_DIR}/hyperledger/tempgenesis.block ${BASE_DIR}/$path/genesis.block; done  
rm ${BASE_DIR}/hyperledger/tempgenesis.block
defaultOrderer=$(python3 $BASE_DIR/scripts/getDefaultOrderer.py $BASE_DIR/generated-config/configtx.yaml $sysChannelProfile)

#
# CREATING ORDERERES AND TURNING THEM ON IN DOCKER
#
#
for  ((l=0; l<$numberOfOrgs; l+=1)); do
    export ORG_NAME=${matrix[$l,name]}
    export ORG_NAME_UPPER=${ORG_NAME^^}
    nOrds=${matrix[$l,orderer-quantity]}
    for ((i=1; i<=$nOrds; i+=1)); do
        export ORDERER_NUMBER=$i
        export BINDABLE_PORT
        echo -e $blueback \# "Changing the name of "orderer" service in file docker-compose.yml" $resetvid
        perl -pi -e 's/ orderer:/ orderer'$ORDERER_NUMBER'-'$ORG_NAME':/g' docker-compose.yml
        sleep 2s
        echo -e $blueback \# Turning on orderer$ORDERER_NUMBER-$ORG_NAME $resetvid
        docker-compose -f docker-compose.yml up -d orderer$ORDERER_NUMBER-$ORG_NAME
        perl -pi -e 's/ orderer'$ORDERER_NUMBER'-'$ORG_NAME':/ orderer:/g' docker-compose.yml
        sleep 1s
        docker logs orderer$ORDERER_NUMBER-$ORG_NAME
        orgsOrdPorts[orderer$i-$ORG_NAME]=$BINDABLE_PORT
        BINDABLE_PORT=$(($BINDABLE_PORT+1))
    done
done

#
# CREATING PEERS AND THEIR COUCHDBS AND TURNING THEM ON IN DOCKER
#
#
for  ((l=0; l<$numberOfOrgs; l+=1)); do
    export ORG_NAME=${matrix[$l,name]}
    export ORG_NAME_UPPER=${ORG_NAME^^}
    nPeers=${matrix[$l,peer-quantity]}
    for ((i=1; i<=$nPeers; i+=1)); do
        export PEER_NUMBER=$i
        export BINDABLE_PORT
        #mkdir -p ${BASE_DIR}/hyperledger/${ORG_NAME}/peer${PEER_NUMBER}/data/couchdb
        #mkdir -p ${BASE_DIR}/hyperledger/${ORG_NAME}/peer${PEER_NUMBER}/data/couchdb_config
        #echo -e $blueback \# "Changing the name of service "couchdb" in the file docker-compose.yml" $resetvid
        #perl -pi -e 's/ couch-db:/ couch-db-peer'$PEER_NUMBER'-'$ORG_NAME':/g' docker-compose.yml
        #sleep 1s
        #echo -e $blueback \# Turning on couchdb-peer$PEER_NUMBER-$ORG_NAME $resetvid
        #docker-compose -f docker-compose.yml up -d couch-db-peer$PEER_NUMBER-$ORG_NAME
        #perl -pi -e 's/ couch-db-peer'$PEER_NUMBER'-'$ORG_NAME':/ couch-db:/g' docker-compose.yml
        #sleep 1s
        #docker logs couchdb-peer$PEER_NUMBER-$ORG_NAME
        #BINDABLE_PORT=$(($BINDABLE_PORT+1))

        #export COUCHDB_ADDRESS="couchdb-peer$PEER_NUMBER-$ORG_NAME"
        export BINDABLE_PORT
        echo -e $blueback \# "Changing the name of service "peer" in the file docker-compose.yml" $resetvid
        perl -pi -e 's/ peer:/ peer'$PEER_NUMBER'-'$ORG_NAME':/g' docker-compose.yml
        sleep 1s
        echo -e $blueback \# Turning on peer$PEER_NUMBER-$ORG_NAME $resetvid
        docker-compose -f docker-compose.yml up -d peer$PEER_NUMBER-$ORG_NAME
        perl -pi -e 's/ peer'$PEER_NUMBER'-'$ORG_NAME':/ peer:/g' docker-compose.yml
        sleep 1s
        docker logs peer$PEER_NUMBER-$ORG_NAME
        orgsPeerPorts[peer$i-$ORG_NAME]=$BINDABLE_PORT
        BINDABLE_PORT=$(($BINDABLE_PORT+1))
    done
done

#
# CREATING ONE CLI FOR ONE ADMIN OF EACH ORGANIZATION
# VOLUME LINKED WITH admin1
#
#for  ((l=0; l<$numberOfOrgs; l+=1)); do
#    export ORG_NAME=${matrix[$l,name]}
#    nAdms=${matrix[$l,admin-quantity]}
#    echo -e $blueback \# "Changing 'cli' service name in file 'docker-compose.yml'" $resetvid
#    perl -pi -e 's/ cli:/ cli-'$ORG_NAME':/g' docker-compose.yml
#    sleep 1s
#    echo -e $blueback \# Turning cli-$ORG_NAME on $resetvid
#    docker-compose -f docker-compose.yml up -d cli-$ORG_NAME
#    perl -pi -e 's/ cli-'$ORG_NAME':/ cli:/g' docker-compose.yml
#    sleep 1s
#    docker logs cli-$ORG_NAME
#done
docker-compose -f docker-compose.yml up -d cli

#
# CREATING THE 'connection-tls.json' files for the applications SDKs
#
echo -e $blueback \# "Creating organizations LOCALHOST 'connections-tls.json' in folder 'generated-connection-tls'"$resetvid
mkdir -p $BASE_DIR/generated-connection-tls
for key in ${!orgsRootCAPorts[@]}; do
    caPortsArgs="$caPortsArgs\"$key\":${orgsRootCAPorts[$key]},"
done
for key in ${!orgsOrdPorts[@]}; do
    ordPortsArgs="$ordPortsArgs\"$key\":${orgsOrdPorts[$key]},"
done
for key in ${!orgsPeerPorts[@]}; do
    peerPortsArgs="$peerPortsArgs\"$key\":${orgsPeerPorts[$key]},"
done
python3 $BASE_DIR/scripts/localhostAppConnectionsCreator.py --basedir $BASE_DIR --caports \{${caPortsArgs%?}\} --ordererports \{${ordPortsArgs%?}\} --peerports \{${peerPortsArgs%?}\}

echo -e $blueback \# "Creating organizations DOCKER 'connections-tls.json' in folder 'generated-connections-tls'"$resetvid
python3 $BASE_DIR/scripts/dockerAppConnectionsCreator.py --basedir $BASE_DIR

echo -e $blueback \# "Copying organizations 'connections-tls.json' to 'energy-applications/cfgs'"$resetvid
cp $BASE_DIR/generated-connection-tls/*.*  $BASE_DIR/energy-applications/cfgs/

#
# CREATING CONFIGURED CHANNELS
# MAKING PEERS OF INCLUDED ORGS JOIN
# INSTALLING AND COMMMITING CHAINCODES
#
exit="false"
while true; do
    while true; do
        read -p "Do you wish create any channel? [yN]" yn
        case $yn in
            [Yy]* ) read -p   "$(echo -e $blueback "MAKE SURE YOU HAVE CONFIGURED THE CHANNEL IN generated-config/configtx.yaml (PRESS ENTER TO CONTINUE)" $resetvid)"; break;;
            [Nn]* ) exit="true"; break;;
            * ) echo "Please answer yes or no.";;
        esac
    done
    if [ $exit = "true" ]; then break; fi
    ###read -p "Type the desired channel Profile name. Ex:  SampleMultiMSPRaftAppChannel, SampleSingleMSPChannel, etc: " profile
    profile="SampleMultiMSPRaftAppChannel"
    ###read -p "Type the desired channel ID (lower case): " channelID
    channelID="canal"
    channelID=${channelID,,}
    #Python script reads the $profile variable and prints the orgs in the channel
    ORGS_IN_CHANNEL=( $(python3 $BASE_DIR/scripts/getOrganizationsInChannel.py $BASE_DIR/generated-config/configtx.yaml $profile) )
    #Generate channel CreateChannelTx

    firstOrgInChannelUpper=${ORGS_IN_CHANNEL[0]}
    firstOrgInChannelLower=${firstOrgInChannelUpper,,}
    configtxgen -configPath $BASE_DIR/generated-config -profile $profile -outputCreateChannelTx ${BASE_DIR}/hyperledger/$firstOrgInChannelLower/admin1/$channelID.tx -channelID $channelID --asOrg $firstOrgInChannelUpper
    export MSYS_NO_PATHCONV=1

    #
    # CREATION OF THE CHANNEL BY ADMIN1 FROM ORG1
    # MESSAGE SENT TO A ORDERER
    #
    echo -e $blueback \# Creating channel$resetvid
    docker exec -e CORE_PEER_LOCALMSPID=$firstOrgInChannelUpper -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/$firstOrgInChannelLower/admin1/msp cli peer channel create -c $channelID -f /tmp/hyperledger/$firstOrgInChannelLower/admin1/$channelID.tx -o $defaultOrderer:7050 --outputBlock /tmp/hyperledger/$firstOrgInChannelLower/admin1/$channelID.block --tls --cafile /tmp/hyperledger/$firstOrgInChannelLower/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem

    unset MSYS_NO_PATHCONV
    rm ${BASE_DIR}/hyperledger/$firstOrgInChannelLower/admin1/$channelID.tx

    echo -e $blueback \# Copying the channel genesis block for other orgs admin $resetvid
    for orgName in ${ORGS_IN_CHANNEL[@]} ; do
        orgName=${orgName,,}
        if [ $orgName != $firstOrgInChannelLower ]; then
            cp ${BASE_DIR}/hyperledger/$firstOrgInChannelLower/admin1/$channelID.block ${BASE_DIR}/hyperledger/$orgName/admin1/$channelID.block 
        fi
    done

    export MSYS_NO_PATHCONV=1
    #
    # MAKING EVERY PEER OF INVOLVED ORGS
    # JOIN THE CHANNEL USING THE CLI OF EACH ORG
    #
    for orgName in ${ORGS_IN_CHANNEL[@]} ; do
        echo -e $blueback \# Making every peer in org $orgName join channel $channelID $resetvid
        orgNameUpper=$orgName
        orgNameLower=${orgNameUpper,,}
        orgIndex=$(findOrgIndexByName $orgNameLower)
        x=$orgIndex

        enrollType=${matrix[$orgIndex,msptype]}
        if [ $enrollType != "idemix" ]; then
            enrollType="bccsp"
        fi

        nPeers=${matrix[$orgIndex,peer-quantity]}
        for ((i=1; i<=nPeers; i+=1)); do
            docker exec -e CORE_PEER_LOCALMSPID=$orgNameUpper -e CORE_PEER_LOCALMSPTYPE=$enrollType -e CORE_PEER_ADDRESS=peer$i-$orgNameLower:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/$orgNameLower/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/$orgNameLower/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer channel join -b /tmp/hyperledger/$orgNameLower/admin1/$channelID.block   
        done

        #
        # Setting ANCHOR PEERS for organizations in the channel
        #
        if (( nPeers > 0 )); then
            (echo -e $blueback \# "Setting peer1-$orgNameLower as ANCHOR PEER in org $orgNameLower for channel $channelID" $resetvid
            docker exec -e CORE_PEER_LOCALMSPID=$orgNameUpper -e CORE_PEER_LOCALMSPTYPE=$enrollType -e CORE_PEER_ADDRESS=peer1-$orgNameLower:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/$orgNameLower/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/$orgNameLower/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli bash -c "mkdir -p art$x && peer channel fetch config art$x/config_block.pb -o $defaultOrderer:7050 -c $channelID --tls --cafile /tmp/hyperledger/$orgNameLower/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem"

            docker exec cli configtxlator proto_decode --input art$x/config_block.pb --type common.Block --output art$x/config_block.json
            docker exec cli bash -c "jq .data.data[0].payload.data.config art$x/config_block.json > art$x/config.json"
            docker exec cli cp art$x/config.json art$x/config_copy.json
            docker exec cli bash -c "jq '.channel_group.groups.Application.groups.'$orgNameUpper'.values += {\"AnchorPeers\":{\"mod_policy\": \"Admins\",\"value\":{\"anchor_peers\": [{\"host\": \"'peer1-$orgNameLower'\",\"port\": 7051}]},\"version\": \"0\"}}' art$x/config_copy.json > art$x/modified_config.json"
            docker exec cli configtxlator proto_encode --input art$x/config.json --type common.Config --output art$x/config.pb
            docker exec cli configtxlator proto_encode --input art$x/modified_config.json --type common.Config --output art$x/modified_config.pb
            docker exec cli configtxlator compute_update --channel_id canal --original art$x/config.pb --updated art$x/modified_config.pb --output art$x/config_update.pb
            docker exec cli configtxlator proto_decode --input art$x/config_update.pb --type common.ConfigUpdate --output art$x/config_update.json
            docker exec cli bash -c "echo '{\"payload\":{\"header\":{\"channel_header\":{\"channel_id\":\"'$channelID'\",\"type\":2}},\"data\":{\"config_update\":'\$(cat art$x/config_update.json)'}}}' | jq . > art$x/config_update_in_envelope.json"
            docker exec cli configtxlator proto_encode --input art$x/config_update_in_envelope.json --type common.Envelope --output art$x/config_update_in_envelope.pb

            docker exec -e CORE_PEER_LOCALMSPID=$orgNameUpper -e CORE_PEER_LOCALMSPTYPE=$enrollType -e CORE_PEER_ADDRESS=peer1-$orgNameLower:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/$orgNameLower/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/$orgNameLower/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer channel update -f art$x/config_update_in_envelope.pb -c canal -o $defaultOrderer:7050 --tls --cafile /tmp/hyperledger/$orgNameLower/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem
            ) &
        fi 
    done 

    #
    # PACKING THE CHAINCODE
    # TO BE INSTALLED IN EVERY PEER IN THE CHANNEL
    #
    echo -e $blueback "Put the CHAINCODE to be installed in ALL peers in the FOLDER: ${BASE_DIR}/hyperledger/chaincode" $resetvid
    echo -e $blueback "!!!!!!!WE ONLY INSTALL GO CHAINCODES!!!!!!!!" $resetvid
    ###read -p "PRESS ENTER TO CONTINUE"
    echo -e $blueback "'cli' container is packing the chaincodes" $resetvid
    export MSYS_NO_PATHCONV=1
    chaincodeNames=$(ls ${BASE_DIR}/chaincode/)
    for chaincodeName in ${chaincodeNames[@]}; do
        docker exec cli peer lifecycle chaincode package $chaincodeName.tar.gz --path /opt/gopath/src/github.com/hyperledger/chaincode/$chaincodeName/go --lang golang --label $chaincodeName
    done

    #
    # INSTALLING AND APPROVING ALL CHAINCODE 
    # FOR EVERY ORG IN CHANNEL
    # 
    echo -e $blueback "Put the CHAINCODE to be installed in ALL peers in the FOLDER: ${BASE_DIR}/hyperledger/chaincode" $resetvid
    echo -e $blueback "!!!!!!!WE ONLY INSTALL GO CHAINCODES!!!!!!!!" $resetvid
    ###read -p "PRESS ENTER TO CONTINUE"
    for orgName in ${ORGS_IN_CHANNEL[@]} ; do
        (orgNameUpper=$orgName
        orgNameLower=${orgNameUpper,,}
        orgIndex=$(findOrgIndexByName $orgNameLower)
        adminMSP="/tmp/hyperledger/$orgNameLower/admin1/msp"
        adminCaTLSCert="/tmp/hyperledger/$orgNameLower/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem"

        enrollType=${matrix[$orgIndex,msptype]}
        if [ $enrollType != "idemix" ]; then
            enrollType="bccsp"
        fi

        #
        # INSTALLING THE CHAINCODE
        # IN EVERY PEER
        #
        echo -e $blueback "Installing the CHAINCODES in EVERY peer of org $orgNameUpper" $resetvid
        nPeers=${matrix[$orgIndex,peer-quantity]}
        for ((i=1; i<=nPeers; i+=1)); do
            for chaincodeName in ${chaincodeNames[@]}; do
                echo -e $blueback "Installing chaincode '$chaincodeName' in peer$i-$orgNameLower" $resetvid
                docker exec -e CORE_PEER_LOCALMSPID=$orgNameUpper -e CORE_PEER_LOCALMSPTYPE=$enrollType -e CORE_PEER_ADDRESS=peer$i-$orgNameLower:7051 -e CORE_PEER_MSPCONFIGPATH=$adminMSP -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=$adminCaTLSCert cli peer lifecycle chaincode install $chaincodeName.tar.gz

            done
        done


        #
        # STORING CHAINCODES IDS FOR APPROVAL
        #
        if [ $enrollType != "idemix" ]; then
            echo -e $blueback "Storing chaincode '$chaincodeName' identifier for later ORG approval" $resetvid
            jsonString=$(docker exec -e CORE_PEER_LOCALMSPID=$orgNameUpper -e CORE_PEER_ADDRESS=peer1-$orgNameLower:7051 -e CORE_PEER_MSPCONFIGPATH=$adminMSP -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=$adminCaTLSCert cli peer lifecycle chaincode queryinstalled --output json)
            
            #get installed chaincodes IDs
            unset MSYS_NO_PATHCONV
            chaincodeIDs=( $(python3 $BASE_DIR/scripts/getChaincodesIDs.py "$jsonString") )
            export MSYS_NO_PATHCONV=1

            #
            # APPROVING ALL CHAINCONDES FOR ORGANIZATION
            #
            chaincodeIndex=0
            for chaincodeName in ${chaincodeNames[@]}; do
                echo -e $blueback "Approving chaincode '$chaincodeName' for org $orgNameUpper" $resetvid
                docker exec -e CORE_PEER_LOCALMSPID=$orgNameUpper -e CORE_PEER_ADDRESS=peer1-$orgNameLower:7051 -e CORE_PEER_MSPCONFIGPATH=$adminMSP -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=$adminCaTLSCert cli peer lifecycle chaincode approveformyorg -o $defaultOrderer:7050 --tls --cafile $adminCaTLSCert --channelID $channelID --name $chaincodeName --version 1 --init-required --package-id ${chaincodeIDs[$chaincodeIndex]} --sequence 1
                
                chaincodeIndex=$((chaincodeIndex+1))  
            done

        fi
        ) &
    done

    wait

    #
    # COMMITING THE CHAINCODES!
    # ONLY ONE ORG ADMIN REQUESTS SIGNATURE 
    # FROM PEERS IN ALL ORGANIZATIONS IN THE CHANNEL
    #
    committerOrgUpper=$firstOrgInChannelUpper
    committerOrgLower=$firstOrgInChannelLower
    adminCaTLSCert="/tmp/hyperledger/$committerOrgLower/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem"
    echo -e $blueback "$commiterOrgUpper is committing ALL chaincodes" $resetvid
    for chaincodeName in ${chaincodeNames[@]}; do
        tlsRootFlags=""
        peerAddressesFlags=""
        echo -e $blueback "Org $committerOrgUpper is commiting chaincode $chaincodeName" $resetvid
        for orgName in ${ORGS_IN_CHANNEL[@]}; do
            orgNameLower=${orgName,,}
            orgIndex=$(findOrgIndexByName $orgNameLower)
            nPeers=${matrix[$orgIndex,peer-quantity]}
            if (( $nPeers > 0 )); then 
                tlsRootFlags+="--tlsRootCertFiles $adminCaTLSCert " 
                peerAddressesFlags+="--peerAddresses peer1-${orgNameLower}:7051 "
            fi
        done

        docker exec -e CORE_PEER_LOCALMSPID=$committerOrgUpper -e CORE_PEER_ADDRESS=peer1-$committerOrgLower:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/$committerOrgLower/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/$committerOrgLower/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer lifecycle chaincode commit -o $defaultOrderer:7050 --channelID $channelID --name $chaincodeName --version 1 --sequence 1 --init-required --tls --cafile $adminCaTLSCert $tlsRootFlags $peerAddressesFlags
    done
    
    #
    # THIS SCRIPT WILL NOT CALL THE "INIT" FUNCTION ON CHAINCODES
    # CALL YOURSELF FOLLOWING THE EXAMPLES BELOW
    #
done

mkdir -p $BASE_DIR/test-reports
#echo -e $blueback "Printing the CONTAINERS sizes to file 'test-reports/initial-containers-sizes.txt'" $resetvid
#docker ps --format "{{.Names}}: {{.Size}}" > $BASE_DIR/test-reports/initial-containers-sizes-windows.txt

#echo -e $blueback 'Printing (du -hs) result called from INSIDE peers and orderers to file test-reports/initial-containers-filesystem-sizes.txt' $resetvid
#echo 'Files size sum (du -hs) result from each container:' > $BASE_DIR/test-reports/initial-containers-filesystem-sizes.txt
#for ordererName in ${!orgsOrdPorts[@]}; do
#   echo -n "$ordererName: "  >> $BASE_DIR/test-reports/initial-containers-filesystem-sizes.txt
#   docker exec $ordererName du -hs >> $BASE_DIR/test-reports/initial-containers-filesystem-sizes.txt
#done
#for peerName in ${!orgsPeerPorts[@]}; do
#    echo -n "$peerName: "  >> $BASE_DIR/test-reports/initial-containers-filesystem-sizes.txt
#    docker exec $peerName du -hs >> $BASE_DIR/test-reports/initial-containers-filesystem-sizes.txt
#done

echo -e $blueback "Starting containers 'cli-application' to execute our applications inside the docker private network" $resetvid
for  ((i=1; i<=$applicationInstancesNumber; i+=1)); do
    export APPLICATION_INSTANCE_ID=$i
    perl -pi -e 's/cli-applications-ubuntu:/cli-applications-ubuntu-'$i':/g' docker-compose.yml
    sleep 1s 
    docker-compose -f docker-compose.yml up -d cli-applications-ubuntu-$i
    perl -pi -e 's/cli-applications-ubuntu-'$i':/cli-applications-ubuntu:/g' docker-compose.yml
done
#-------------------------------------------- EXIT --------------------------------------

#exit 1 

export MSYS_NO_PATHCONV=1

#
# EXAMPLE peer lifecycle chaincode package
#
#docker exec cli peer lifecycle chaincode package energy.tar.gz --path /opt/gopath/src/github.com/hyperledger/fabric-samples/chaincode/energy/go --lang golang --label energy

#
# EXAMPLE peer lifecycle chaincode install
#
#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer lifecycle chaincode install energy.tar.gz

#
# EXAMPLE peer lifecycle chaincode queryinstalled
#
#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem -e ORDERER_CA=/tmp/hyperledger/ufsc/admin1/msp/cacerts/0-0-0-0-7053.pem cli peer lifecycle chaincode queryinstalled

#
# EXAMPLE peer lifecycle chaincode approveformyorg
#
#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer lifecycle chaincode approveformyorg -o orderer1-ufsc:7050 --tls --cafile /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --channelID canal --name energy --version 1 --init-required --package-id energy:ebcc9e859c96abc105843c9cb75a5d34e7f32688f44a499730481d0f02414bf7 --sequence 1 


#
# EXAMPLE peer lifecycle chaincode checkcommitreadiness
#
#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer lifecycle chaincode checkcommitreadiness  --channelID canal --name energy --version 1 --init-required --sequence 1

#
# EXEMPLE peer lifecycle chaincode commit
#
#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer lifecycle chaincode commit -o orderer1-ufsc:7050 --channelID canal --name energy --version 1 --sequence 1 --init-required --tls --cafile /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --peerAddresses peer1-parma:7051

#
# EXEMPLE peer lifecycle chaincode querycommitted
#
#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer lifecycle chaincode querycommitted -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051

#
# EXAMPLES calling "init" on ENERGY chaincode
#
docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"Init","Args":[]}' --isInit

exit 1

#read -p "PRESS ENTER TO CONTINUE"

#the init below wil FAIL, because the chaincode INIT must be called in MAJORITY organizations (UFSC and PARMA)
#docker exec -e CORE_PEER_LOCALMSPID=PARMA -e CORE_PEER_ADDRESS=peer1-parma:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/parma/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/parma/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-parma:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/parma/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/parma/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"Init","Args":[]}' --isInit

#
# EXEMPLES  calling function "sensorDeclareActive" on ENERGY chaincode
#
docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/sensor1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/sensor1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/sensor1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/sensor1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"sensorDeclareActive","Args":[]}'

#read -p "PRESS ENTER TO CONTINUE"


docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/sensor2/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/sensor2/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/sensor2/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/sensor2/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"sensorDeclareActive","Args":[]}'

#read -p "PRESS ENTER TO CONTINUE"


docker exec -e CORE_PEER_LOCALMSPID=PARMA -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/parma/sensor1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/parma/sensor1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/parma/sensor1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/parma/sensor1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/parma/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"sensorDeclareActive","Args":[]}'

#read -p "PRESS ENTER TO CONTINUE"


docker exec -e CORE_PEER_LOCALMSPID=PARMA -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/parma/sensor2/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/parma/sensor2/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/parma/sensor2/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/parma/sensor2/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/parma/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"sensorDeclareActive","Args":[]}'

#read -p "PRESS ENTER TO CONTINUE"


#
# EXEMPLE  calling function "getActiveSensors" on ENERGY chaincode
#
#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 -c '{"function":"getActiveSensors","Args":[]}'

#read -p "PRESS ENTER TO CONTINUE"


#docker exec -e CORE_PEER_LOCALMSPID=PARMA -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/parma/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/parma/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/parma/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/parma/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 -c '{"function":"getActiveSensors","Args":[]}'

#read -p "PRESS ENTER TO CONTINUE"


#
# EXEMPLE  calling function "disableSensors" on ENERGY chaincode
#
#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"disableSensors","Args":["eDUwOTo6Q049c2Vuc29yMS11ZnNjLE9VPWNsaWVudCtPVT11ZnNjLE89VUZTQyxMPUZsb3JpYW5vcG9saXMsU1Q9U0MsQz1CUjo6Q049cmNhLWNhLE9VPUZhYnJpYyxPPUh5cGVybGVkZ2VyLFNUPU5vcnRoIENhcm9saW5hLEM9VVM=","eDUwOTo6Q049c2Vuc29yMi11ZnNjLE9VPWNsaWVudCtPVT11ZnNjLE89VUZTQyxMPUZsb3JpYW5vcG9saXMsU1Q9U0MsQz1CUjo6Q049cmNhLWNhLE9VPUZhYnJpYyxPPUh5cGVybGVkZ2VyLFNUPU5vcnRoIENhcm9saW5hLEM9VVM="]}'

#
# EXEMPLE  calling function "enableSensors" on ENERGY chaincode
#
#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"enableSensors","Args":["eDUwOTo6Q049c2Vuc29yMS11ZnNjLE9VPWNsaWVudCtPVT11ZnNjLE89VUZTQyxMPUZsb3JpYW5vcG9saXMsU1Q9U0MsQz1CUjo6Q049cmNhLWNhLE9VPUZhYnJpYyxPPUh5cGVybGVkZ2VyLFNUPU5vcnRoIENhcm9saW5hLEM9VVM=","eDUwOTo6Q049c2Vuc29yMi11ZnNjLE9VPWNsaWVudCtPVT11ZnNjLE89VUZTQyxMPUZsb3JpYW5vcG9saXMsU1Q9U0MsQz1CUjo6Q049cmNhLWNhLE9VPUZhYnJpYyxPPUh5cGVybGVkZ2VyLFNUPU5vcnRoIENhcm9saW5hLEM9VVM="]}'

#
# EXEMPLE  calling function "setTrustedSensors" on ENERGY chaincode
#
docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"setTrustedSensors","Args":["UFSC","UFSC","eDUwOTo6Q049c2Vuc29yMS11ZnNjLE9VPWNsaWVudCtPVT11ZnNjLE89VUZTQyxMPUZsb3JpYW5vcG9saXMsU1Q9U0MsQz1CUjo6Q049cmNhLWNhLE9VPUZhYnJpYyxPPUh5cGVybGVkZ2VyLFNUPU5vcnRoIENhcm9saW5hLEM9VVM=","eDUwOTo6Q049c2Vuc29yMi11ZnNjLE9VPWNsaWVudCtPVT11ZnNjLE89VUZTQyxMPUZsb3JpYW5vcG9saXMsU1Q9U0MsQz1CUjo6Q049cmNhLWNhLE9VPUZhYnJpYyxPPUh5cGVybGVkZ2VyLFNUPU5vcnRoIENhcm9saW5hLEM9VVM="]}'

#read -p "PRESS ENTER TO CONTINUE"

docker exec -e CORE_PEER_LOCALMSPID=PARMA -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/parma/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/parma/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/parma/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/parma/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/parma/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"setTrustedSensors","Args":["UFSC","UFSC","eDUwOTo6Q049c2Vuc29yMS11ZnNjLE9VPWNsaWVudCtPVT11ZnNjLE89VUZTQyxMPUZsb3JpYW5vcG9saXMsU1Q9U0MsQz1CUjo6Q049cmNhLWNhLE9VPUZhYnJpYyxPPUh5cGVybGVkZ2VyLFNUPU5vcnRoIENhcm9saW5hLEM9VVM=","eDUwOTo6Q049c2Vuc29yMi11ZnNjLE9VPWNsaWVudCtPVT11ZnNjLE89VUZTQyxMPUZsb3JpYW5vcG9saXMsU1Q9U0MsQz1CUjo6Q049cmNhLWNhLE9VPUZhYnJpYyxPPUh5cGVybGVkZ2VyLFNUPU5vcnRoIENhcm9saW5hLEM9VVM="]}'

#read -p "PRESS ENTER TO CONTINUE"

#
# EXEMPLE  calling function "setDistrustedSensors" on ENERGY chaincode
#
#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"setDistrustedSensors","Args":["eDUwOTo6Q049c2Vuc29yMS11ZnNjLE9VPWNsaWVudCtPVT11ZnNjLE89VUZTQyxMPUZsb3JpYW5vcG9saXMsU1Q9U0MsQz1CUjo6Q049cmNhLWNhLE9VPUZhYnJpYyxPPUh5cGVybGVkZ2VyLFNUPU5vcnRoIENhcm9saW5hLEM9VVM=","eDUwOTo6Q049c2Vuc29yMi11ZnNjLE9VPWNsaWVudCtPVT11ZnNjLE89VUZTQyxMPUZsb3JpYW5vcG9saXMsU1Q9U0MsQz1CUjo6Q049cmNhLWNhLE9VPUZhYnJpYyxPPUh5cGVybGVkZ2VyLFNUPU5vcnRoIENhcm9saW5hLEM9VVM="]}'

#
# EXEMPLE  calling function "getTrustedSensors" on ENERGY chaincode
#
#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"getTrustedSensors","Args":[]}'

#read -p "PRESS ENTER TO CONTINUE"

#docker exec -e CORE_PEER_LOCALMSPID=PARMA -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/parma/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/parma/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/parma/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/parma/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/parma/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"getTrustedSensors","Args":[]}'

#read -p "PRESS ENTER TO CONTINUE"


#
# EXEMPLE  calling function "publishSensorData" on ENERGY chaincode
#
#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/sensor1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/sensor1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/sensor1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/sensor1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/ufsc/sensor1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"publishSensorData","Args":["1","3834792229","777","306.5","0","0","0"]}'

#read -p "PRESS ENTER TO CONTINUE"


#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/sensor2/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/sensor2/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/sensor2/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/sensor2/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/ufsc/sensor2/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"publishSensorData","Args":["1","3835050276","777","306.7","0","0","0"]}'

#read -p "PRESS ENTER TO CONTINUE"


#
# EXEMPLE  calling function "getSensorsPublishedData" on ENERGY chaincode
#
#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"getSensorsPublishedData","Args":["eDUwOTo6Q049c2Vuc29yMS11ZnNjLE9VPWNsaWVudCtPVT11ZnNjLE89VUZTQyxMPUZsb3JpYW5vcG9saXMsU1Q9U0MsQz1CUjo6Q049cmNhLWNhLE9VPUZhYnJpYyxPPUh5cGVybGVkZ2VyLFNUPU5vcnRoIENhcm9saW5hLEM9VVM=","eDUwOTo6Q049c2Vuc29yMi11ZnNjLE9VPWNsaWVudCtPVT11ZnNjLE89VUZTQyxMPUZsb3JpYW5vcG9saXMsU1Q9U0MsQz1CUjo6Q049cmNhLWNhLE9VPUZhYnJpYyxPPUh5cGVybGVkZ2VyLFNUPU5vcnRoIENhcm9saW5hLEM9VVM="]}'

#read -p "PRESS ENTER TO CONTINUE"

#
# EXEMPLE  calling function "getCallerIDAndCallerMspID" on ENERGY chaincode
#
#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/seller1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/seller1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/seller1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/seller1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/ufsc/seller1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"getCallerIDAndCallerMspID","Args":[]}'

#read -p "PRESS ENTER TO CONTINUE"

#
# EXEMPLE  calling function "registerSeller" on ENERGY chaincode
#
docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"registerSeller","Args":["eDUwOTo6Q049c2VsbGVyMS11ZnNjLE9VPWNsaWVudCtPVT11ZnNjLE89VUZTQyxMPUZsb3JpYW5vcG9saXMsU1Q9U0MsQz1CUjo6Q049cmNhLWNhLE9VPUZhYnJpYyxPPUh5cGVybGVkZ2VyLFNUPU5vcnRoIENhcm9saW5hLEM9VVM=", "UFSC","eDUwOTo6Q049c2Vuc29yMS11ZnNjLE9VPWNsaWVudCtPVT11ZnNjLE89VUZTQyxMPUZsb3JpYW5vcG9saXMsU1Q9U0MsQz1CUjo6Q049cmNhLWNhLE9VPUZhYnJpYyxPPUh5cGVybGVkZ2VyLFNUPU5vcnRoIENhcm9saW5hLEM9VVM=", "3", "3"]}'

#read -p "PRESS ENTER TO CONTINUE"

#
# EXEMPLE  calling function "publishEnergyGeneration" on ENERGY chaincode
#
#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/sensor1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/sensor1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/sensor1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/sensor1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/ufsc/sensor1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"publishEnergyGeneration","Args":["0", "1614232439", "solar", "100000", "wind", "100000"]}'

#read -p "PRESS ENTER TO CONTINUE"

#
# EXEMPLE  calling function "registerSellBid" on ENERGY chaincode
#
#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/seller1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/seller1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/seller1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/seller1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/ufsc/seller1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"registerSellBid","Args":["15", "9", "solar"]}'


#read -p "PRESS ENTER TO CONTINUE"


#
# EXEMPLE  calling function "registerBuyBid" on ENERGY chaincode
#
#docker exec -e CORE_PEER_LOCALMSPID=IDEMIXORG -e  CORE_PEER_LOCALMSPTYPE=idemix -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/idemixorg/buyer1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/idemixorg/buyer1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/idemixorg/buyer1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/idemixorg/buyer1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/idemixorg/buyer1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"registerBuyBid","Args":["UFSC","tokentest1","UFSC", "15", "3", "solar"]}'

#read -p "PRESS ENTER TO CONTINUE"


#
# EXEMPLE  calling function "validateBuyBid" on ENERGY chaincode
#
#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"validateBuyBid","Args":["tokentest1", "1000", ""]}'

#read -p "PRESS ENTER TO CONTINUE"

#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/seller1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/seller1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/seller1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/seller1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/ufsc/seller1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"registerSellBid","Args":["15", "9", "solar"]}'

#
# EXEMPLES  calling function "auction" on ENERGY chaincode
#
#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"auction","Args":[]}'

#
# EXEMPLES  calling function "auctionSortedQueries" on ENERGY chaincode
#
#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"auctionSortedQueries","Args":[]}'

#read -p "PRESS ENTER TO CONTINUE"


#
# EXEMPLES  calling function "getEnergyTransactionsFromPaymentToken" on ENERGY chaincode
#
#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"getEnergyTransactionsFromPaymentToken","Args":["UFSC","tokentest1"]}'

#read -p "PRESS ENTER TO CONTINUE"

#-------------------------- TESTING FUNCTIONS ----------------------------------------

#
# EXEMPLE  calling function "registerMultipleSellBids" on ENERGY chaincode
#
#docker exec -e  GRPC_GO_LOG_SEVERITY_LEVEL=debug -e  GRPC_GO_LOG_VERBOSITY_LEVEL=2 -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/seller1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/seller1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/seller1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/seller1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/ufsc/seller1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"registerMultipleSellBids","Args":["100", "10", "20", "5", "15", "solar"]}'

#read -p "PRESS ENTER TO CONTINUE"

#
# EXEMPLE  calling function "registerMultipleBuyBids" on ENERGY chaincode
#
#docker exec -e CORE_PEER_LOCALMSPID=IDEMIXORG -e  CORE_PEER_LOCALMSPTYPE=idemix -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/idemixorg/buyer1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/idemixorg/buyer1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/idemixorg/buyer1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/idemixorg/buyer1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/idemixorg/buyer1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"registerMultipleBuyBids","Args":["100", "UFSC", "10", "20", "5", "15", "solar"]}'

#sleep 5s

#
# EXEMPLE  calling function "validateMultipleBuyBids" on ENERGY chaincode
#
#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"validateMultipleBuyBids","Args":["100"]}'


#read -p "PRESS ENTER TO CONTINUE"


#
# EXEMPLES  calling function "clearSellBids" on ENERGY chaincode
#
#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"clearSellBids","Args":[]}'

#
# EXEMPLES  calling function "clearBuyBids" on ENERGY chaincode
#
#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"clearBuyBids","Args":[]}'

#
# EXEMPLES  calling function "printDataQuantityByPartialCompositeKey" on ENERGY chaincode
#
#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"printDataQuantityByPartialCompositeKey","Args":["BuyBid"]}'

#
# EXEMPLES  calling function "deleteDataByPartialCompositeKey" on ENERGY chaincode
#
#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"deleteDataByPartialCompositeKey","Args":["BuyBid"]}'

#
# EXEMPLES  calling function "printDataQuantityByPartialSimpleKey" on ENERGY chaincode
#
#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"printDataQuantityByPartialSimpleKey","Args":["SmartData"]}'

#
# EXEMPLES  calling function "deleteDataByPartialSimpleKey" on ENERGY chaincode
#
#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"deleteDataByPartialSimpleKey","Args":["SmartData"]}'


#
# EXEMPLES  calling function "registerMultipleSellers" on ENERGY chaincode
#
#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"registerMultipleSellers","Args":["1999"]}'

#
# EXEMPLES  calling function "measureTimeDifferentAuctions" on ENERGY chaincode
#
#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"measureTimeDifferentAuctions","Args":["1"]}'

#
# EXEMPLES  calling function "testWorldStateLogic" on ENERGY chaincode
#
#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"testWorldStateLogic","Args":[]}'

#-------------------------- END TESTING FUNCTIONS ----------------------------------------



#
# EXEMPLE calling function query with idemix credentials
#
#docker exec -e CORE_PEER_LOCALMSPID=IDEMIXORG -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_LOCALMSPTYPE=idemix -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/idemixorg/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/idemixorg/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/idemixorg/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/idemixorg/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051  -c '{"function":"query","Args":["A"]}'


#
# EXEMPLE calling function invoke with idemix credentials
# Transfering 1 from A to B
#
#docker exec -e CORE_PEER_LOCALMSPID=IDEMIXORG -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_LOCALMSPTYPE=idemix -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/idemixorg/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/idemixorg/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer chaincode invoke -o orderer1-ufsc:7050 --channelID canal --name energy --tls --cafile /tmp/hyperledger/idemixorg/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem  --tlsRootCertFiles /tmp/hyperledger/idemixorg/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-ufsc:7051 --tlsRootCertFiles /tmp/hyperledger/idemixorg/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem --peerAddresses peer1-parma:7051 -c '{"function":"invoke","Args":["A", "B", "1"]}'

#
# EXEMPLE calling peer channel fetch to see the blocks of the channel
#
#docker exec -e CORE_PEER_LOCALMSPID=IDEMIXORG -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_LOCALMSPTYPE=idemix -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/idemixorg/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/idemixorg/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli peer channel fetch --channelID canal newest

#
# EXEMPLE decoding the newest block fetched with the command above
#
#docker exec -e CORE_PEER_LOCALMSPID=IDEMIXORG -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_LOCALMSPTYPE=idemix -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/idemixorg/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/idemixorg/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli configtxlator proto_decode --type=common.Block --input=canal_newest.block

#configtxgen -configPath ./generated-config -profile SampleMultiMSPRaft -outputBlock ./genesis.block -channelID syschannel
#docker exec -e CORE_PEER_LOCALMSPID=UFSC -e CORE_PEER_ADDRESS=peer1-ufsc:7051 -e CORE_PEER_LOCALMSPTYPE=idemix -e CORE_PEER_MSPCONFIGPATH=/tmp/hyperledger/ufsc/admin1/msp -e CORE_PEER_TLS_ENABLED=true -e CORE_PEER_TLS_ROOTCERT_FILE=/tmp/hyperledger/ufsc/admin1/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem cli configtxlator proto_decode --type=common.Block --input=./genesis.block

#
# Example discover
#
#docker exec cli discover --configFile /tmp/hyperledger/ufsc/conf.yaml config --channel canal --server peer1-ufsc:7051
#docker exec cli discover --configFile /tmp/hyperledger/ufsc/conf.yaml endorsers --channel canal --server peer1-ufsc:7051 --chaincode energy

export MSYS_NO_PATHCONV=1