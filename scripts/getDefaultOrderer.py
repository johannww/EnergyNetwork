
import yaml
import sys

configurationFilePath = sys.argv[1]

with open(configurationFilePath) as configStream:
  #fazer parse do generated-config/configtx.yaml
  parsedConfig = yaml.safe_load(configStream)
  #ler variavel de ambiente profile
  sysChannelProfile = sys.argv[2]
  #descobrir as organizacoes que estao no canal descrito pela variavel profile
  organizationsOrdering = parsedConfig["Profiles"][sysChannelProfile]["Orderer"]["Organizations"]
  orgNameLower = organizationsOrdering[0]["Name"].lower()
  ordererName = "orderer1-"+orgNameLower
  print(ordererName, end="")
