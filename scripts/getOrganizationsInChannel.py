
import yaml
import sys

with open("generated-config/configtx.yaml") as configStream:
  #fazer parse do generated-config/configtx.yaml
  parsedConfig = yaml.safe_load(configStream)
  #ler variavel de ambiente profile
  channelProfile = sys.argv[1]
  #descobrir as organizacoes que estao no canal descrito pela variavel profile
  organizatiosInTheChannel = parsedConfig["Profiles"][channelProfile]["Application"]["Organizations"]
  #setar uma variavel de ambiente com as organizacoes do canal
  orgsEnv = ''
  for org in organizatiosInTheChannel:
    orgsEnv += org["Name"]+' '

  #orgsEnv = orgsEnv[:len(orgsEnv)]
  print(orgsEnv.upper())
