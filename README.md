# Energy Network - Developed in Hyperledger Fabric

FALAR SOBRE

## Testing conditions

> Windows 10

> Docker 20.10.5

> docker-compose version 1.28.5, build c4eb3a1f

> Python 3.8.2

> shyaml 0.6.2 --> to parse YAMLs in Bash scripts

> go version go1.16 windows/amd64

> git version 2.28.0.windows.1

> GitBash for windows

> curl 7.72.0 (x86_64-pc-win32)

> Apache Maven 3.6.3

> Patched [Hyperledger Fabric tag/v2.3.0](https://github.com/hyperledger/fabric/tree/v2.3.0)

> Patched [Hyperledger Fabric SDK for Java tag/v2.2.5](https://github.com/hyperledger/fabric-sdk-java/tree/v2.2.5)

> Patched [Hyperledger Fabric Gateway tag/v2.2.1](https://github.com/hyperledger/fabric-gateway-java/tree/v2.2.1)

> AWS Cli - aws-cli/2.1.32 Python/3.8.8 Windows/10 exe/AMD64 prompt/off

## Installing and patching dependencies

After following the tutorial on [Hyperldeger Fabric Pre-requisites](https://hyperledger-fabric.readthedocs.io/en/release-2.2/prereqs.html), the `install-dependencies.sh` script will clone, patch and install:

[Hyperledger Fabric](https://github.com/hyperledger/fabric)

[Hyperledger Fabric gRPC Service Definitions](https://github.com/hyperledger/fabric-protos)

[Hyperledger Fabric SDK for Java](https://github.com/hyperledger/fabric-sdk-java)

[Hyperledger Fabric Gateway](https://github.com/hyperledger/fabric-gateway-java)

```bash
./scripts/install-dependencies.sh
```

## Configuring network

## Initiating network


```bash
./scripts/automated-creation-factored-energy.sh
```