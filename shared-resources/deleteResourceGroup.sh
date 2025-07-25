#!/bin/bash

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'
#Deletes resource group
export resourcesName=$1
echo -e "In the process of deleting servicehubval-$resourcesName-rg"
az group delete -n servicehubval-$resourcesName-rg --yes
if [ $? -ne 0 ]
then
    echo -e "${RED}Deletion of resource group failed${NC}"
    exit 1
else
    echo -e "${GREEN}Resource group deletion was successful${NC}"
fi
