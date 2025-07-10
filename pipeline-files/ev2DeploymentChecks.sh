buildReason=$2
sourceBranch=$3
pullRequestId=$4
testInfraSubscription=$5
directoryName=$6

function matchingGroups() {
    echo "Connecting to subscription $testInfraSubscription"
    az account set --subscription "$testInfraSubscription"
    if [ $? -ne 0 ]; then
        echo "Failed to connect to subscription $testInfraSubscription. Exiting."
        exit 1
    fi
    echo "Connected to subscription $testInfraSubscription"
    if [ "$buildReason" == "PullRequest" ]; then
        echo "This build is being triggered by a pull request."
    elif [ "$sourceBranch" == "refs/heads/master" ]; then
        echo "This build is being triggered by the default branch."
    else
        echo "This build is not being triggered by a PR or the default branch. Please only run this pipeline through a Pull Request or on the default branch for resources to work as expected. Exiting."
        exit 1
    fi

    echo "This is correctly being run from within a pull request or on the default branch. Checking for existing resource groups."
    if [ "$sourceBranch" = "refs/heads/master" ]; then
        resourcesName="TestCorp"
    else
        resourcesName="${pullRequestId}TestCorp"
    fi
    echo "Resources name is set to $resourcesName"
    matchingGroups=($(az group list --query "[].name" -o tsv | grep -i 'servicehubval' | grep -v 'cluster' | grep -i "$resourcesName"))
    echo "Matching resource groups: ${matchingGroups[@]}"
    if [ ${#matchingGroups[@]} -eq 0 ]; then
        echo "No matching resource groups found. Exiting."
        exit 1
    fi
    echo "Found ${#matchingGroups[@]} matching resource groups."
}

function checkDeploymentSuccess(){
    matchingGroups
    if [ $? -ne 0 ]; then
        echo "Failed to find matching resource groups. Exiting."
        exit 1
    fi
    echo "Checking deployment success for resource groups."
    for group in "${matchingGroups[@]}"; do
        echo "Getting AKS Credentials for: $group"
        clusterName=$(az aks list --resource-group $group --query "[].name" -o tsv)
        if [ -z "$clusterName" ]; then
            echo "No AKS cluster found in resource group $group. Exiting."
            exit 1
        fi
        az aks get-credentials --resource-group $group --name $clusterName --overwrite-existing
        if [ $? -ne 0 ]; then
            echo "Failed to get AKS credentials for cluster $clusterName in resource group $group. Exiting."
            exit 1
        fi
        echo "Checking pod status for $directoryName for cluster $clusterName in resource group $group."
        cd $directoryName/server/test
        export suppressImageTagCheck=true
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
        echo -e "${GREEN}Service deployment was successful!${NC}"

    done
    echo "All deployments succeeded."
}

"$@"
