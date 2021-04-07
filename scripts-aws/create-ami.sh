#!/bin/bash

# This script creates the AWS Ami for the experiments with the EnergyNetwork
# containing the patched Hyperledger Fabric images, the patched fabric-gateway-java and
# the patched fabric-sdk-java

unset MSYS_NO_PATHCONV
blueback="\0033[1;37;44m"
resetvid="\0033[0m"
SCRIPT_DIR=$(dirname "$(realpath "$0")")

echo -e $blueback \# getting default security group id $resetvid
securityGroup=$(aws ec2 describe-security-groups --output text --query 'SecurityGroups[0].GroupId') && echo $securityGroup

echo -e $blueback \# getting default subnet id $resetvid
subnetId=$(aws ec2 describe-subnets --output text --query 'Subnets[0].SubnetId') && echo $subnetId

echo -e $blueback \# trying to create key pair 'EnergyNetwork' $resetvid
keyContent=$(aws ec2 create-key-pair --key-name EnergyNetwork --query "KeyMaterial" --output text)
if [[ $keyContent ]]; then
   echo "$keyContent" > $SCRIPT_DIR/EnergyNetworkAwsKeyPair.pem
fi

echo -e $blueback \# getting the first key pair name $resetvid
keyPairName=$(aws ec2 describe-key-pairs --output text --query 'KeyPairs[0].KeyName') && echo $keyPairName

echo -e $blueback \# starting instance with Amazon Ubuntu 20.04 Server: ami-0b9517e2052e8be7a $resetvid
instanceId=$(aws ec2 run-instances --image-id ami-0b9517e2052e8be7a --count 1 --instance-type t2.micro --key-name $keyPairName --security-group-ids $securityGroup --subnet-id $subnetId --output text --query 'Instances[0].InstanceId') && echo $instanceId

#instanceId=$(aws ec2 describe-instances --output text --query 'Reservations[0].Instances[0].InstanceId') && echo $instanceId
aws ec2 wait instance-running --instance-ids $instanceId

echo -e $blueback \# getting instance hostname $resetvid
publicDnsName=$(aws ec2 describe-instances --instance-ids $instanceId --output text --query 'Reservations[0].Instances[0].NetworkInterfaces[0].Association.PublicDnsName') && echo $publicDnsName

echo -e $blueback \# remote copying 'scripts' and 'patches' folder $resetvid
until scp -i $SCRIPT_DIR/EnergyNetworkAwsKeyPair.pem -r -p $SCRIPT_DIR/../scripts ubuntu@$publicDnsName:/home/ubuntu/scripts; do sleep 5; done
scp -i $SCRIPT_DIR/EnergyNetworkAwsKeyPair.pem -r -p $SCRIPT_DIR/../patches ubuntu@$publicDnsName:/home/ubuntu/patches

echo -e $blueback \# increasing the EBS volume size to 15G to support the installations $resetvid
ebsVolumeId=$(aws ec2 describe-volumes --output text --filters Name=attachment.instance-id,Values=$instanceId --query 'Volumes[0].VolumeId')
aws ec2 modify-volume --size 15 --volume-id $ebsVolumeId
aws ec2 wait volume-in-use --volume-id $ebsVolumeId

echo -e $blueback \# expanding the ebs /dev/ to the new size $resetvid
ssh -i $SCRIPT_DIR/EnergyNetworkAwsKeyPair.pem ubuntu@$publicDnsName 'until sudo growpart "/dev/$(lsblk | grep 15G | awk '"'"{ print \$1 }"'"')" 1; do sleep 5; done'

echo -e $blueback \# installing software requirements $resetvid
ssh -i $SCRIPT_DIR/EnergyNetworkAwsKeyPair.pem ubuntu@$publicDnsName << EOF
    sudo dd if=/dev/zero of=/swapfile bs=64M count=32
    sudo chmod 600 /swapfile
    sudo mkswap /swapfile
    sudo swapon /swapfile
    sudo swapon -s
    echo '/swapfile swap swap defaults 0 0' | sudo tee -a /etc/fstab

    sudo apt update
    sudo apt install git -y
    git config --global core.autocrlf false && git config --global core.longpaths true
    sudo apt install openjdk-13-jre -y
    sudo apt install openjdk-13-jdk -y
    echo 'export JAVA_HOME=/usr/lib/jvm/java-13-openjdk-amd64' | sudo tee /etc/profile.d/javapaths.sh
    echo 'export PATH=\$PATH:\$JAVA_HOME/bin' | sudo tee -a /etc/profile.d/javapaths.sh
    source /etc/profile
    sudo apt install maven -y
    sudo apt install -y \
        apt-transport-https \
        ca-certificates \
        curl \
        gnupg \
        lsb-release
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
    echo \
        "deb [arch=amd64 signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu \
        \$(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
    sudo apt update
    sudo apt install -y docker-ce docker-ce-cli containerd.io
    sudo usermod -aG docker \$USER
EOF

echo -e $blueback \# rebooting instance $resetvid
aws ec2 reboot-instances --instance-ids $instanceId

aws ec2 wait instance-running --instance-ids $instanceId

until ssh -i $SCRIPT_DIR/EnergyNetworkAwsKeyPair.pem ubuntu@$publicDnsName "echo 'ssh available'"; do sleep 5; done

echo -e $blueback \# installing docker-compose, golang and patched fabric software $resetvid
ssh -i $SCRIPT_DIR/EnergyNetworkAwsKeyPair.pem ubuntu@$publicDnsName << EOF
    sudo curl -L "https://github.com/docker/compose/releases/download/1.28.6/docker-compose-\$(uname -s)-\$(uname -m)" -o /usr/local/bin/docker-compose
    sudo chmod +x /usr/local/bin/docker-compose
    sudo apt install build-essential -y
    wget https://golang.org/dl/go1.15.11.linux-amd64.tar.gz
    sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.15.11.linux-amd64.tar.gz
    rm go1.15.11.linux-amd64.tar.gz
    echo 'export GOPATH=\$HOME/go' | sudo tee /etc/profile.d/gopaths.sh
    echo 'export PATH=\$PATH:/usr/local/go/bin:\$GOPATH/bin' | sudo tee -a /etc/profile.d/gopaths.sh
    source /etc/profile
    /home/ubuntu/scripts/install-dependencies.sh
    rm -r patches
    rm -r scripts
    mkdir EnergyNetwork
EOF


echo -e $blueback \# creating AWS AMI for the EBS $resetvid
# https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/creating-an-ami-ebs.html
# create AMI image
imageId=$(aws ec2 create-image --instance-id $instanceId --name "EnergyNetworkImage" --description "AMI with all requirements to run applications or hyperledger fabric containers for the EnergyNetwork" --output text --query 'ImageId') && echo "The AMI ID is: $imageId"

aws ec2 wait image-available --image-ids $imageId

aws ec2 terminate-instances --instance-ids $instanceId