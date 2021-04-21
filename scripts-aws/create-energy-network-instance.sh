#!/bin/bash

# This script creates the AWS Ami for the experiments with the EnergyNetwork
# containing the patched Hyperledger Fabric images, the patched fabric-gateway-java and
# the patched fabric-sdk-java

unset MSYS_NO_PATHCONV
blueback="\0033[1;37;44m"
resetvid="\0033[0m"
SCRIPT_DIR=$(dirname "$(realpath "$0")")

machineName=$1
awsInstanceType=$2

securityGroup=$(aws ec2 describe-security-groups --output text --query 'SecurityGroups[0].GroupId')

subnetId=$(aws ec2 describe-subnets --output text --query 'Subnets[0].SubnetId')

keyContent=$(aws ec2 create-key-pair --key-name EnergyNetwork --query "KeyMaterial" --output text)
if [[ $keyContent ]]; then
   echo "$keyContent" > $SCRIPT_DIR/EnergyNetworkAwsKeyPair.pem
fi

keyPairName=$(aws ec2 describe-key-pairs --output text --query 'KeyPairs[0].KeyName')

if [[ $awsInstanceType == *"g."* ]]; then imageName="EnergyNetworkImageArm"; else imageName="EnergyNetworkImage"; fi
imageId=$(aws ec2 describe-images --owners "self" --filters Name=name,Values=$imageName --output text --query 'Images[0].ImageId')

instanceId=$(aws ec2 run-instances --image-id $imageId --count 1 --instance-type $awsInstanceType --key-name $keyPairName --security-group-ids $securityGroup --subnet-id $subnetId --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=$machineName}]" --output text --query 'Instances[0].InstanceId')

#instanceId=$(aws ec2 describe-instances --output text --query 'Reservations[0].Instances[0].InstanceId') && echo $instanceId
aws ec2 wait instance-running --instance-ids $instanceId

publicDnsName=$(aws ec2 describe-instances --instance-ids $instanceId --output text --query 'Reservations[0].Instances[0].NetworkInterfaces[0].Association.PublicDnsName') && echo $publicDnsName
