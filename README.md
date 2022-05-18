# Energy Network - Developed in Hyperledger Fabric

Energy Network is the result of a Master's tehesis reasearch on blockchain use in clean energy trading and generation validation. Our model considers energy sellers, buyers, sensors, utility companies and payment companies. We work with **PATCHED** versions of *Hyperledger Fabric*, *Hyperledger Fabric SDK for Java* and *Hyperledger Fabric Gateway for Java*.



## Testing conditions

The Energy Network was deployed locally in Ubuntu 20.04 and Windows 10. Also, our scripts allow a fully configurable deploy in AWS and can be easily adapted to other Cloud infrastructures. The following dependencies are required:

### Exclusive windows dependencies

> GitBash for windows

### Common dependencies

> Docker 20.10.5

> docker-compose version 1.28.5, build c4eb3a1f

> Python 3.8.2

> python3 regex

> python3 matplotlib

> shyaml 0.6.2 --> to parse YAMLs in Bash scripts

> Java OpenJDK 13

> Apache Maven 3.6.3

> dos2unix

> go version go1.16 

> git version 2.28.0.windows.1

> curl 7.72.0 (x86_64-pc-win32)

> Patched [Hyperledger Fabric tag/v2.3.0](https://github.com/hyperledger/fabric/tree/v2.3.0)

> Patched [Hyperledger Fabric SDK for Java tag/v2.2.5](https://github.com/hyperledger/fabric-sdk-java/tree/v2.2.5)

> Patched [Hyperledger Fabric Gateway for Java tag/v2.2.1](https://github.com/hyperledger/fabric-gateway-java/tree/v2.2.1)

> AWS Cli - aws-cli/2.1.32 Python/3.8.8 Windows/10 exe/AMD64 prompt/off

## Installing and patching dependencies

After following the tutorial on [Hyperldeger Fabric Pre-requisites](https://hyperledger-fabric.readthedocs.io/en/release-2.2/prereqs.html), the `scripts/install-dependencies.sh` script will clone, patch and install **MODIFIED**:

[Hyperledger Fabric](https://github.com/hyperledger/fabric)

[Hyperledger Fabric gRPC Service Definitions](https://github.com/hyperledger/fabric-protos)

```bash
./scripts/install-dependencies.sh
```

To deal with the  **energy-applications**, `scripts/install-java-dependencies.sh` will install the patched versions of:

[Hyperledger Fabric SDK for Java](https://github.com/hyperledger/fabric-sdk-java)

[Hyperledger Fabric Gateway](https://github.com/hyperledger/fabric-gateway-java)

```bash
./scripts/install-java-dependencies.sh
```

## Configuring network

First edit the file `CONFIG-ME-FIRST.yaml` to set the **organizations**:

```yaml
organizations:
    - name: ufsc
      admin-quantity: 1
      client-quantity: 0
      peer-quantity: 1
      orderer-quantity: 1
      buyer-quantity: 0
      seller-quantity: 1
      sensor-quantity: 1
      msptype: x509

applications-quantity: 1
```

The configurations above will cause the creation of x509 or Idemix credentials for each *admin*, *peer*, *orderer*, *buyer*, *seller*, and *sensor*.

All credentials are available in the `hyperledger/` folder.

## Initiating network

1. Start docker

2. Run the network creation script and follow the instructions:

```bash
./scripts/automated-creation-factored-energy.sh
```

3. **CONTINUES**

## AWS deploy

Our network already has scripts to deploy and experiment on the AWS cloud infrastructure.

### Generating an AMI

We support ARM and x86 AMI generations with all dependencies to run our applications, orderers, peers, and experiments. Since AWS ARM instances are more efficient than x86, we opted to perform our experiments with them. The AMI includes our patched Hyperledger Fabric and Fabric SDK Java.

```
./scripts-aws/create-ami-arm.sh
```

### Deploying the network

After the AMI creation, the network can be deployed by executing the AWS creation script. The flags "-o", "-p", and "-a" indicate the instance type of orderers, peers and applications, respectively.

```
./scripts-aws/automated-aws-creation.sh -o t4g.micro -p t4g.micro -a t4g.micro
```

### AWS Experiments

    The experiments are configured in the **test-configuration.yaml** file. It is possible to set the number of simulated sensors, buyers, and sellers per application instance. The SmartData unit is set in the sensors' configuration (Read more in [Smart Data docs](https://epos.lisha.ufsc.br/IoT+Platform#SmartData)). Each entity (sensor, buyer or seller) publishes the total of **publishquantity** during the experiment. ** publishinterval** milliseconds space the transactions. At the end of the file, the auction period **auctioninterval** in milliseconds can be set, and the interval between two consecutive network auctions.

```yaml
#quantity per cli-application
sensors:
  quantity: 50
  unit: 3834792229
  #Interval in ms
  publishinterval: 5000
  publishquantity: 20
  msp: UFSC

sellers:
  quantity: 50
  #Interval in ms
  publishinterval: 5000
  publishquantity: 5
  msp: UFSC


buyers:
  quantity: 1
  #Interval in ms
  publishinterval: 5000
  publishquantity: 1
  msp: IDEMIXORG

#Interval in ms
auctioninterval: 30000
```