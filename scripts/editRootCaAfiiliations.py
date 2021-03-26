import sys
import platform
import regex
import yaml

orgName = sys.argv[1]
caCryptoDir = sys.argv[2]

affiliationsTemplate="""
    org:
      - org
"""

#building generated-config/configtx.yaml based on template
with open(caCryptoDir+"fabric-ca-server-config.yaml", 'r') as caServerConfigYamlFile:
  caServerConfig = caServerConfigYamlFile.read()
  #finding where affiliations are set on fabric-ca-server-config.yaml text
  affiliationsBeginning = caServerConfig.find("affiliations:")+len("affiliations:")
  affiliationsEnd = caServerConfig.find("\n#",affiliationsBeginning)
  #adjusting affiliationsTemplate for organization name
  affiliationsTemplate = affiliationsTemplate.replace("org", orgName)
  caServerConfig = caServerConfig[:affiliationsBeginning] + affiliationsTemplate  + caServerConfig[affiliationsEnd:]


#writing to "fabric-ca-server-config.yaml"
with open(caCryptoDir+"fabric-ca-server-config.yaml", 'w') as newCaServerConfig:
  newCaServerConfig.write(caServerConfig)
