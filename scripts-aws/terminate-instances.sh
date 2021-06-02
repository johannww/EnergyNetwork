#!/bin/bash

# This script terminates ALL instances in AWS

unset MSYS_NO_PATHCONV
blueback="\0033[1;37;44m"
resetvid="\0033[0m"
SCRIPT_DIR=$(dirname "$(realpath "$0")")

instanceIDs=$(aws ec2 describe-instances --output text --filter 'Name=instance-state-name,Values=running,stopped' --query 'Reservations[*].Instances[0].[InstanceId]' | dos2unix)
for instanceID in ${instanceIDs[@]}; do
    instanceIDsOneLiner="$instanceID $instanceIDsOneLiner"
done
aws ec2 terminate-instances --instance-ids $instanceIDsOneLiner