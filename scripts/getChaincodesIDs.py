import json
import sys


jsonFormatText = sys.argv[1]
installedChaincodes = json.loads(jsonFormatText)

for chaincode in installedChaincodes["installed_chaincodes"]:
  print(chaincode["package_id"])
