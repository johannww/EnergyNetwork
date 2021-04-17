# This script reads the 'stat-*' files of a specific "test-report/NUMBER" dir and
# generates CPU, MEMORY and NETWORK graphics in the respective test report folder

import sys
import os
import yaml
import statistics
import matplotlib.pyplot as plt

units = {}
CONTAINER_METRICS_QUANTITY=4
CPU = "Cpu"
units[CPU] = "%"
MEM = "Memory"
units[MEM] = "GiB"
NET_IN = "Network In"
NET_OUT = "Network Out"
units[NET_IN] = "MB"
units[NET_OUT] = "MB"
DISK_READ = "Disk Reads"
DISK_WRITE = "Disk Writes"
units[DISK_READ] = "MB"
units[DISK_WRITE] = "MB"


def memToGibFloat(memStrDockerStats):
  numPosition = memStrDockerStats.find("MiB /")
  if numPosition >= 0:
    mem = float(memStrDockerStats[:numPosition]) / 1024
  else:
    numPosition = memStrDockerStats.find("GiB /")
    mem = float(memStrDockerStats[:numPosition])
  return mem

def netUsageToMbFloat(netStrDockerStats):

  numPosition = netStrDockerStats.find("GB")
  if numPosition >= 0:
    return float(netStrDockerStats[:numPosition]) * 1024

  numPosition = netStrDockerStats.find("MB")
  if numPosition >= 0:
    return float(netStrDockerStats[:numPosition])

  numPosition = netStrDockerStats.find("kB")
  if numPosition >= 0:
    return float(netStrDockerStats[:numPosition]) / 1024

  numPosition = netStrDockerStats.find("B")
  if numPosition >= 0:
    return float(netStrDockerStats[:numPosition]) / (1024 * 1024)

def saveEntityGraphs(entityName):
  with(open(testReportDir+"/stats-"+entityName+".txt", "r")) as stat:
    statTxt = stat.read()
    statTxt = statTxt.replace("[2J[H", "").replace("\n", ":")
    dataCells = statTxt.split(":")[:-1]
    stats[entityName] = {CPU: [], MEM: [], NET_IN: [], NET_OUT: [], DISK_READ: [], DISK_WRITE: []}
    for metricsSetNumber in range(0, len(dataCells), CONTAINER_METRICS_QUANTITY):
      stats[entityName][CPU].append(float(dataCells[metricsSetNumber].replace("%","")) * 100)
      mem = memToGibFloat(dataCells[metricsSetNumber+1])
      stats[entityName][MEM].append(mem)
      netIn = netUsageToMbFloat(dataCells[metricsSetNumber+2].split(" / ")[0])
      stats[entityName][NET_IN].append(netIn)
      netOut = netUsageToMbFloat(dataCells[metricsSetNumber+2].split(" / ")[1])
      stats[entityName][NET_OUT].append(netOut)
      diskRead = netUsageToMbFloat(dataCells[metricsSetNumber+3].split(" / ")[0])
      stats[entityName][DISK_READ].append(diskRead)
      diskWrite = netUsageToMbFloat(dataCells[metricsSetNumber+3].split(" / ")[1])
      stats[entityName][DISK_WRITE].append(diskWrite)
    
    for metric in stats[entityName]:
      plt.plot(stats[entityName][metric])
      plt.ylabel("{} ({})".format(metric, units[metric]))
      plt.grid(True)
      #plt.ylim(bottom=-0.001)
      plt.savefig("{}/stats-{}-{}.jpg".format(plotsDir, entityName, metric.replace(" ","-")))
      plt.close()

#testReportDir = sys.argv[1]
testReportDir = "D:/UFSC/Mestrado/Hyperledger/Fabric/EnergyNetwork/test-reports/1"
plotsDir = testReportDir+"/plots"
if not os.path.isdir:
  os.mkdir(plotsDir, 1)

'''Parsing "CONFIG-ME-FIRST.yaml" file'''
with open("CONFIG-ME-FIRST.yaml", 'r') as preconfig:
  parsedPreConfig = yaml.safe_load(preconfig)

# Ploting metrics for orderers and peers
stats = {}
for org in parsedPreConfig["organizations"]:
  orgName = org["name"]
  for ordererNumber in reversed(range(1, org["orderer-quantity"]+1)):
    ordererName = "orderer{}-{}".format(str(ordererNumber), orgName)
    saveEntityGraphs(ordererName)
      
  for peerNumber in reversed(range(1, org["peer-quantity"]+1)):
    peerName = "peer{}-{}".format(str(peerNumber), orgName)
    saveEntityGraphs(peerName)
    saveEntityGraphs("chaincode-{}".format(peerName))

saveEntityGraphs("cli-applications")

