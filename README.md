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

falar que usamos arm instances e que AMI eh gerada na instancia t4g.micro (que eh gratuita)