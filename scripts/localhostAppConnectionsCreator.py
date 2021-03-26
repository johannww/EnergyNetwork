import sys
import platform
import regex
import yaml
import json
import argparse
import platform


parser = argparse.ArgumentParser(
    description='Creates the "connection-tls.json" for each network organization to be used with applications.')
parser.add_argument('--basedir', type=str,
                    help='Network root dir, parent of "hyperledger/" folder')
parser.add_argument('--caports', type=str,
                    help='JSON formated map {"CA-NAME": PORT,...}')
parser.add_argument('--ordererports', type=str,
                    help='JSON formated map {"orderer1-org": PORT,...}')
parser.add_argument('--peerports', type=str,
                    help='JSON formated map {"peer1-org": PORT,...}')


args = parser.parse_args()

caPorts = json.loads(args.caports)
ordererPorts = json.loads(args.ordererports)
peerPorts = json.loads(args.peerports)

baseDir = args.basedir


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
            "path": baseDir+"/hyperledger/{}/admin1/msp/keystore/key.pem".format(orgNameLower)
        },
        "signedCertPEM": {
            # CERTIFICADO TLS MESMO????
            "path": baseDir+"/hyperledger/{}/admin1/msp/signcerts/cert.pem".format(orgNameLower)
        }
    }

    for i in range(0, org["orderer-quantity"]):
        ordererIndex = i + 1
        ordererName = "orderer{}-{}".format(ordererIndex, orgNameLower)

        appConnectionsTls["channels"]["canal"]["orderers"].append(ordererName)

        appConnectionsTls["orderers"][ordererName] = {
            "url": "grpcs://localhost:{}".format(ordererPorts[ordererName]),
            "mspid": "UFSC",
            "grpcOptions": {
                "ssl-target-name-override": ordererName,
                "hostnameOverride": ordererName
            },
            "tlsCACerts": {
                "path": baseDir+"/hyperledger/{}/msp/tlscacerts/tls-ca-cert.pem".format(orgNameLower)
            },
            "adminPrivateKeyPEM": {
                "path": baseDir+"/hyperledger/{}/admin1/msp/keystore/key.pem".format(orgNameLower)
            },
            "signedCertPEM": {
                "path": baseDir+"/hyperledger/{}/admin1/msp/signcerts/cert.pem".format(orgNameLower)
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
            "url": "grpcs://localhost:{}".format(peerPorts[peerName]),
            "grpcOptions": {
                "ssl-target-name-override": peerName,
                "hostnameOverride": peerName,
                "request-timeout": 120001
            },
            "tlsCACerts": {
                "path": baseDir+"/hyperledger/{}/msp/tlscacerts/tls-ca-cert.pem".format(orgNameLower)
            }
        }

    rcaName = "rca-{}".format(orgNameLower)
    appConnectionsTls["certificateAuthorities"][rcaName] = {
        "url": "https://localhost:{}".format(caPorts[orgNameLower]),
        "grpcOptions": {
            "verify": True
        },
        "tlsCACerts": {
            "path": baseDir+"/hyperledger/{}/msp/tlscacerts/tls-ca-cert.pem".format(orgNameLower)
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

    with open(baseDir+"/generated-connection-tls/{}-connection-tls.json".format(orgNameLower), "w") as orgAppConnectionFile:
        for channel in appConnectionsTls["channels"]:
            for peer in appConnectionsTls["channels"][channel]["peers"]:
                appConnectionsTls["channels"][channel]["peers"][peer]["eventSource"] = True

        orgSpecificAppConnectionTlsJSON = json.dumps(
            appConnectionsTls, indent=4)
        orgAppConnectionFile.write(orgSpecificAppConnectionTlsJSON)

    with open(baseDir+"/generated-connection-tls/non-blocking-{}-connection-tls.json".format(orgNameLower), "w") as orgAppConnectionFile:
        for channel in appConnectionsTls["channels"]:
            for peer in appConnectionsTls["channels"][channel]["peers"]:
                appConnectionsTls["channels"][channel]["peers"][peer]["eventSource"] = False

        orgSpecificAppConnectionTlsJSON = json.dumps(
            appConnectionsTls, indent=4)
        orgAppConnectionFile.write(orgSpecificAppConnectionTlsJSON)

