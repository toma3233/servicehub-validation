#!/bin/bash

#Define color codes for printing to help analysis.
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'
#--------------------------------------------------------
# We are assuming resource provisioning has been complete to deploy service on resources

# TODO: Some sort of check that resources are provisioned
#---------
cd csharpuserrp
cd server
make install
if [ $? -ne 0 ]
then
    echo -e "${RED}Service deployment failed with exit code $?${NC}"
    exit 1
fi
cd test
chmod +x ./checkServicePodStatus.sh ./checkServicePodLogs.sh
export KUBECONFIG=$(grep "export KUBECONFIG=" ~/.bashrc | awk -F '=' '{print $2}')
#--------------------------------------------------------
#Make sure service was deployed correctly.
./checkServicePodStatus.sh
if [ $? -ne 0 ]
then
    echo -e "${RED}Pod is not running as expected, please check output.${NC}"
    echo -e "${RED}Service deployment FAILED!${NC}"
    exit 1
fi
./checkServicePodLogs.sh
if [ $? -ne 0 ]
then
    echo -e "${RED}Pod logs are not as expected, please check output.${NC}"
    echo -e "${RED}Service deployment FAILED!${NC}"
    exit 1
fi

#--------------------------------------------------------
# Check service status (e.g., LoadBalancer IP assignment)
chmod +x ./checkServiceStatus.sh
./checkServiceStatus.sh
if [ $? -ne 0 ]
then
    echo -e "${RED}Service status check failed, please check output.${NC}"
    echo -e "${RED}Service deployment FAILED!${NC}"
    exit 1
fi
echo -e "${GREEN}Service deployment was successful!${NC}"
