import sys
import platform
import regex
import yaml
import json
import argparse
import platform


parser = argparse.ArgumentParser(
    description='Creates the "connection-tls.json" for each network organization ASSUMING the caller is a container within the DOCKER private NETWORK.')
parser.add_argument('--basedir', type=str,
                    help='Network root dir, parent of "hyperledger/" folder')



args = parser.parse_args()

ORDERER_DEFAULT_PORT=7050
PEER_DEFAULT_PORT=7051
RCA_DEFAULT_PORT=7053

baseDir = args.basedir
CONTAINER_BASE_DIR="/EnergyNetwork"

'''Parsing "CONFIG-ME-FIRST.yaml" file'''
with open("CONFIG-ME-FIRST.yaml", 'r') as preconfig:
    parsedPreConfig = yaml.safe_load(preconfig)

appConnectionsTls = {
    "name": "johann-generated",
    "version": "1.0.0",
    "client": {
            "organization": "",
            "connection": {
                "timeout": {
                    "peer": {
                        "endorser": "300"
                    },
                    "orderer": "300"
                }
            }
    },
    "channels": {
        "canal": {
            "orderers": [],
            "peers": {}
        }
    },
    "organizations": {},
    "orderers": {},
    "peers": {},
    "certificateAuthorities": {}
}

for org in parsedPreConfig["organizations"]:
    orgNameLower = org["name"]
    orgNameUpper = org["name"].upper()

    appConnectionsTls["organizations"][orgNameUpper] = {
        "mspid": orgNameUpper,
        "peers": [],
        "certificateAuthorities": [
            "rca-{}".format(orgNameLower)
        ],
        "adminPrivateKeyPEM": {
            "path": CONTAINER_BASE_DIR+"/hyperledger/{}/admin1/msp/keystore/key.pem".format(orgNameLower)
        },
        "signedCertPEM": {
            # CERTIFICADO TLS MESMO????
            "path": CONTAINER_BASE_DIR+"/hyperledger/{}/admin1/msp/signcerts/cert.pem".format(orgNameLower)
        }
    }

    for i in range(0, org["orderer-quantity"]):
        ordererIndex = i + 1
        ordererName = "orderer{}-{}".format(ordererIndex, orgNameLower)

        appConnectionsTls["channels"]["canal"]["orderers"].append(ordererName)

        appConnectionsTls["orderers"][ordererName] = {
            "url": "grpcs://{}:{}".format(ordererName, ORDERER_DEFAULT_PORT),
            "mspid": "UFSC",
            "grpcOptions": {
                "ssl-target-name-override": ordererName,
                "hostnameOverride": ordererName
            },
            "tlsCACerts": {
                "path": CONTAINER_BASE_DIR+"/hyperledger/{}/msp/tlscacerts/tls-ca-cert.pem".format(orgNameLower)
            },
            "adminPrivateKeyPEM": {
                "path": CONTAINER_BASE_DIR+"/hyperledger/{}/admin1/msp/keystore/key.pem".format(orgNameLower)
            },
            "signedCertPEM": {
                "path": CONTAINER_BASE_DIR+"/hyperledger/{}/admin1/msp/signcerts/cert.pem".format(orgNameLower)
            }
        }

    for i in range(0, org["peer-quantity"]):
        peerIndex = i + 1
        peerName = "peer{}-{}".format(peerIndex, orgNameLower)
        # ONLY CHANNEL 'canal' in this script at this moment
        appConnectionsTls["channels"]["canal"]["peers"][peerName] = {
            "endorsingPeer": True,
            "chaincodeQuery": True,
            "ledgerQuery": True,
            "eventSource": True,
            "discover": False
        }
        appConnectionsTls["organizations"][orgNameUpper]["peers"].append(
            peerName)

        appConnectionsTls["peers"][peerName] = {
            "url": "grpcs://{}:{}".format(peerName, PEER_DEFAULT_PORT),
            "grpcOptions": {
                "ssl-target-name-override": peerName,
                "hostnameOverride": peerName,
                "request-timeout": 120001
            },
            "tlsCACerts": {
                "path": CONTAINER_BASE_DIR+"/hyperledger/{}/msp/tlscacerts/tls-ca-cert.pem".format(orgNameLower)
            }
        }

    rcaName = "rca-{}".format(orgNameLower)
    appConnectionsTls["certificateAuthorities"][rcaName] = {
        "url": "https://{}:{}".format(rcaName, RCA_DEFAULT_PORT),
        "grpcOptions": {
            "verify": True
        },
        "tlsCACerts": {
            "path": CONTAINER_BASE_DIR+"/hyperledger/{}/msp/tlscacerts/tls-ca-cert.pem".format(orgNameLower)
        },
        "registrar": [
            {
                "enrollId": "admin",
                "enrollSecret": "adminpw"
            }
        ]
    }

for org in parsedPreConfig["organizations"]:
    orgNameLower = org["name"]
    orgNameUpper = org["name"].upper()
    appConnectionsTls["client"]["organization"] = orgNameUpper

    with open(baseDir+"/generated-connection-tls/docker-{}-connection-tls.json".format(orgNameLower), "w") as orgAppConnectionFile:
        for channel in appConnectionsTls["channels"]:
            for peer in appConnectionsTls["channels"][channel]["peers"]:
                appConnectionsTls["channels"][channel]["peers"][peer]["discover"] = True
                appConnectionsTls["channels"][channel]["peers"][peer]["eventSource"] = True

        orgSpecificAppConnectionTlsJSON = json.dumps(
            appConnectionsTls, indent=4)
        orgAppConnectionFile.write(orgSpecificAppConnectionTlsJSON)

    with open(baseDir+"/generated-connection-tls/docker-non-blocking-{}-connection-tls.json".format(orgNameLower), "w") as orgAppConnectionFile:
        for channel in appConnectionsTls["channels"]:
            for peer in appConnectionsTls["channels"][channel]["peers"]:
                appConnectionsTls["channels"][channel]["peers"][peer]["discover"] = True
                appConnectionsTls["channels"][channel]["peers"][peer]["eventSource"] = False

        orgSpecificAppConnectionTlsJSON = json.dumps(
            appConnectionsTls, indent=4)
        orgAppConnectionFile.write(orgSpecificAppConnectionTlsJSON)
