import sys
import platform
import regex
import yaml

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
            - Host: peer1-org
              Port: 7051
"""
raftConsentersTemplate="""
            - Host: ##ORDERER##-org
              Port: 7050
              ClientTLSCert: ##BASE_DIR##/hyperledger/org/##ORDERER##/tls-msp/signcerts/cert.pem
              ServerTLSCert: ##BASE_DIR##/hyperledger/org/##ORDERER##/tls-msp/signcerts/cert.pem"""

baseDir = sys.argv[1]

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
    orderSectionBeginning = orgConfigPart.find("OrdererEndpoints:")+len("OrdererEndpoints:")
    orderer = '\n            - "orderer{}-{}:7050"'.format(str(ordererNumber), orgName)
    orgConfigPart = orgConfigPart[:orderSectionBeginning] + orderer + orgConfigPart[orderSectionBeginning:]

    raftConsentersSection = raftConsentersTemplate.replace("##ORDERER##", "orderer{}".format(ordererNumber)).replace("org", orgName).replace("##BASE_DIR##", baseDir) + raftConsentersSection
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

#writing to "generated-config/configtx.yaml"
#with open("generated-config/configtx.yaml", 'w') as newYaml:
  #newYaml.write(configtx)
