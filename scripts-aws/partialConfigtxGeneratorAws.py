import sys
import platform
import regex
import yaml
import json
import os

orgTemplate="""    
    - &ORG
        Name: ORG

        SkipAsForeign: false

        ID: ORG

        MSPDir: ##BASE_DIR##/hyperledger/org/msp

        Policies: &ORGPolicies
            Readers:
                Type: Signature
                Rule: "OR('ORG.member')"
            Writers:
                Type: Signature
                Rule: "OR('ORG.member')"
            Admins:
                Type: Signature
                Rule: "OR('ORG.admin')"
            Endorsement:
                Type: Signature
                Rule: "OR('ORG.peer')"

        OrdererEndpoints:


        AnchorPeers:
            - Host: ##PEER##
              Port: 7051
"""
raftConsentersTemplate="""
            - Host: ##ORDERER_HOST##
              Port: 7050
              ClientTLSCert: ##BASE_DIR##/hyperledger/org/##ORDERER_NAME##/tls-msp/signcerts/cert.pem
              ServerTLSCert: ##BASE_DIR##/hyperledger/org/##ORDERER_NAME##/tls-msp/signcerts/cert.pem"""

baseDir = sys.argv[1]
ordererHosts = json.loads(sys.argv[2])
peerHosts = json.loads(sys.argv[3])

'''Parsing "CONFIG-ME-FIRST.yaml" file'''
with open("CONFIG-ME-FIRST.yaml", 'r') as preconfig:
  parsedPreConfig = yaml.safe_load(preconfig)

#Writing "Organizations:" part of the new configtx.yaml
orgsSection = ""
raftConsentersSection = ""
for org in parsedPreConfig["organizations"]:
  orgName = org["name"]
  orgConfigPart = orgTemplate.replace("ORG", orgName.upper()).replace("org", orgName.lower()).replace("##BASE_DIR##", baseDir)
  #put all OrdererEndpoints: and Consenters:
  for ordererNumber in reversed(range(1, org["orderer-quantity"]+1)):
    ordererName = "orderer{}".format(str(ordererNumber))
    ordererFullName = ordererName+"-{}".format(orgName)
    ordererHost = ordererHosts[ordererFullName]
    orderSectionBeginning = orgConfigPart.find("OrdererEndpoints:")+len("OrdererEndpoints:")
    orderer = '\n            - "{}:7050"'.format(ordererHost)
    orgConfigPart = orgConfigPart[:orderSectionBeginning] + orderer + orgConfigPart[orderSectionBeginning:]

    raftConsentersSection = raftConsentersTemplate.replace("##ORDERER_HOST##", ordererHost).replace('##ORDERER_NAME##', ordererName).replace("org", orgName).replace("##BASE_DIR##", baseDir) + raftConsentersSection
  
  #if org does not have peers
  anchorPeerBeginning = orgConfigPart.find("AnchorPeers:")
  if org["peer-quantity"] < 1:
    orgConfigPart = orgConfigPart[:anchorPeerBeginning]
  else:
    orgConfigPart = orgConfigPart.replace("##PEER##", peerHosts["peer1-"+orgName])

  #if organization uses idemix
  try:
    if org["msptype"] == "idemix":
      idOrgField = "ID: {}".format(orgName.upper())
      idOrgEnd = orgConfigPart.find(idOrgField)+len(idOrgField)
      mspTypeText = "\n\n        msptype: idemix"
      orgConfigPart = orgConfigPart[:idOrgEnd] + mspTypeText + orgConfigPart[idOrgEnd:]
    print(orgConfigPart)
  except:
    pass
  orgsSection += orgConfigPart

#building generated-config/configtx.yaml based on template
with open("config-template/configtxTemplate.yaml", 'r') as templateYaml:
  configtx = templateYaml.read()
  #adding orgsSection
  orgsBeginning = configtx.find("Organizations:")+len("Organizations:")
  configtx = configtx[:orgsBeginning] + orgsSection + configtx[orgsBeginning:]
  #addomg raftConsentersSection
  consentersBeginning = configtx.find("Consenters:")+len("Consenters:")
  configtx = configtx[:consentersBeginning] + raftConsentersSection + configtx[consentersBeginning:]
  #substituting "SampleOrg" references with reference of the first organization provided
  configtx = configtx.replace("SampleOrg", "{}".format(parsedPreConfig["organizations"][0]["name"].upper()))

#writing to "generated-config-aws/configtx.yaml"
os.makedirs(baseDir+"/generated-config-aws", exist_ok=True)
with open("generated-config-aws/configtx.yaml", 'w') as newYaml:
  newYaml.write(configtx)
