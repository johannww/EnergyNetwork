# This script reads the 'stat-*' files of a specific "test-report/NUMBER" dir and
# generates CPU, MEMORY and NETWORK graphics in the respective test report folder

import sys
import os
import yaml
import statistics
import matplotlib.pyplot as plt
import numpy

units = {}
MOVING_AVERAGE_QUANTITY=20
CONTAINER_METRICS_QUANTITY=4
CPU = "Cpu"
units[CPU] = {"unit":"%", "type": "Instantaneous"}
MEM = "Memory"
units[MEM] = {"unit":"GiB", "type": "Instantaneous"}
NET_IN = "Network In"
NET_OUT = "Network Out"
units[NET_IN] = {"unit": "MB", "type": "Cumulative"} 
units[NET_OUT] = {"unit": "MB", "type": "Cumulative"}
DISK_READ = "Disk Reads"
DISK_WRITE = "Disk Writes"
units[DISK_READ] = {"unit": "MB", "type": "Cumulative"}
units[DISK_WRITE] = {"unit": "MB", "type": "Cumulative"}



def memToGibFloat(memStrDockerStats):
  numPosition = memStrDockerStats.find("GiB /")
  if numPosition >= 0:
    return float(memStrDockerStats[:numPosition])

  numPosition = memStrDockerStats.find("MiB /")
  if numPosition >= 0:
    return float(memStrDockerStats[:numPosition]) / 1024

  numPosition = memStrDockerStats.find("KiB /")
  if numPosition >= 0:
    return float(memStrDockerStats[:numPosition]) / (1024 * 1024)



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

def mountEntityStats(entityName):
  with(open(testReportDir+"/stats-"+entityName+".txt", "r")) as stat:
    statTxt = stat.read()
    statTxt = statTxt.replace("[2J[H", "").replace("\n", ":").replace("--:-- / --:--:--", "-5%:-100MiB / -100MiB:-100MB / -100MB:-100MB / -100MB")
    dataCells = statTxt.split(":")[:-1]
    stats[entityName] = {CPU: [], MEM: [], NET_IN: [], NET_OUT: [], DISK_READ: [], DISK_WRITE: []}
    for metricsSetNumber in range(0, len(dataCells), CONTAINER_METRICS_QUANTITY):
      stats[entityName][CPU].append(float(dataCells[metricsSetNumber].replace("%","")))
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


def getBiggestStatLen():
  biggestLen = 0
  for entity in stats:
    statLen = len(stats[entity][CPU])
    if statLen > biggestLen:
      biggestLen = statLen
  return biggestLen


def saveCpuMemAverages(fileName):
  with open(testReportDir+"/"+fileName, 'w') as file:
    for entityName in stats:
      file.write("{}:\n".format(entityName))
      for metric in stats[entityName]:
        if metric == CPU or metric == MEM:
          file.write("  {}:\n".format(metric))
          file.write("    Max: {} {}\n".format(numpy.max(stats[entityName][metric]), units[metric]["unit"]))
          file.write("    Min: {} {}\n".format(numpy.min(stats[entityName][metric]), units[metric]["unit"]))
          file.write("    Average: {} {}\n".format(numpy.average(stats[entityName][metric]), units[metric]["unit"]))
          file.write("    Standard deviation: {} {}\n".format(numpy.std(stats[entityName][metric]), units[metric]["unit"]))
    
def syncStats():
  biggestLen = getBiggestStatLen()
  for entity in stats:
    for metric in stats[entity]:
      stats[entity][metric] = [0 for i in range(0, biggestLen-len(stats[entity][metric]))] + stats[entity][metric]

def plotMovingAverage(plt, cpuMetrics):
  movingAverages = [0.0 for i in range(0, MOVING_AVERAGE_QUANTITY)]
  for i in range(MOVING_AVERAGE_QUANTITY, len(cpuMetrics)):
    movingAverages.append(movingAverages[-1] + cpuMetrics[i]/MOVING_AVERAGE_QUANTITY 
    - cpuMetrics[i-MOVING_AVERAGE_QUANTITY]/MOVING_AVERAGE_QUANTITY)
  plt.plot(movingAverages, 'r', label="{} second Moving Average ".format(MOVING_AVERAGE_QUANTITY))
  plt.legend()

def saveEntitiesGraphs():
  for entityName in stats:
    for metric in stats[entityName]:
      plt.title("{} - {} - {}".format(entityName, metric, units[metric]["type"]))
      plt.plot(stats[entityName][metric], label="CPU %")
      plt.xlabel("Time (s)")
      if metric == CPU:
        plotMovingAverage(plt, stats[entityName][metric])
      plt.ylabel("{} ({})".format(metric, units[metric]["unit"]))
      plt.grid(True)
      #plt.ylim(bottom=-0.001)
      plt.savefig("{}/stats-{}-{}.jpg".format(plotsDir, entityName, metric.replace(" ","-")))
      plt.close()

testReportDir = sys.argv[1]
#testReportDir = "D:/UFSC/Mestrado/Hyperledger/Fabric/EnergyNetwork/test-reports/5"
plotsDir = testReportDir+"/plots"
if not os.path.isdir(plotsDir):
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
    mountEntityStats(ordererName)
      
  for peerNumber in reversed(range(1, org["peer-quantity"]+1)):
    peerName = "peer{}-{}".format(str(peerNumber), orgName)
    mountEntityStats(peerName)
    mountEntityStats("chaincode-{}".format(peerName))

  for appNumber in reversed(range(1, parsedPreConfig["applications-quantity"]+1)):
    mountEntityStats("cli-applications-{}".format(str(appNumber)))


# calculate the average of processing and mem usage
# of each machine and print to file instances-mem-cpu-averages.txt
saveCpuMemAverages("instances-mem-cpu-averages.txt")

# sync the stats time by adding '0' to the beggining of them
# until all stats are same length
syncStats()

# save plotted graphs to 'test-reports/NUMBER/plots'
saveEntitiesGraphs()

