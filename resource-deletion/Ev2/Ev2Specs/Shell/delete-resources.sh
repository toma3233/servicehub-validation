#!/bin/bash
# Script to login to multiple Azure subscriptions and delete specified resource groups

# Define colors for terminal output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# Enable exit on error

# Function to login to a specific subscription
loginToSubscription() {
    local subscription_id=$1
    echo -e "${BLUE}Logging in to subscription: ${subscription_id}${NC}"
    
    az account set --subscription "$subscription_id"
    if [ $? -ne 0 ]; then
        echo -e "${RED}Failed to set subscription: ${subscription_id} even after login${NC}"
        return 1
    fi
    echo -e "${GREEN}Successfully logged in to subscription: ${subscription_id}${NC}"
    return 0
}

# Function to purge a Key Vault
# It checks if the waitForKVPurge flag is set to True, and if so, it waits for the Key Vault to be purged, 
# otherwise it triggers the purge without waiting
keyVaultPurge(){
    local keyVault=$1
    local waitForKVPurge=$2
    if [ "$waitForKVPurge" == true ]; then
        echo -e "${YELLOW}Waiting for Key Vault ${keyVault} to be purged...${NC}"
        az keyvault purge --name "$keyVault"
        if [ $? -ne 0 ]; then
            echo -e "${RED}Failed to purge Key Vault: ${keyVault}${NC}"
            return 1
        else
            echo -e "${GREEN}Key Vault ${keyVault} purged successfully${NC}"
        fi
    else 
        echo -e "${YELLOW}Triggered Key Vault ${keyVault} to be purged, but not waiting...${NC}"
        az keyvault purge --name "$keyVault" --no-wait
    fi
}

# Function to delete a resource group
# It saves the key vault names in the resource group and passes them to the keyVaultPurge function after the resource group is deleted
# If the resource group does not exist, it skips the deletion
deleteResourceGroup() {
    local resource_group=$1
    local waitForKVPurge=$2
    # Check if the resource group exists
    if ! az group show --name "$resource_group" &>/dev/null; then
        echo -e "${RED}Resource group ${resource_group} does not exist. Skipping deletion.${NC}"
        return 0
    fi
    
    echo -e "${BLUE}Checking to see if there are any key vaults in the resource group ${resource_group}${NC}"
    keyVaults=$(az resource list --resource-group "$resource_group" --query "[?type=='Microsoft.KeyVault/vaults'].name" -o tsv)
    echo -e "${YELLOW}Found Key Vaults in resource group ${resource_group}: ${keyVaults}${NC}"


    # Delete the resource group
    az group delete -n "$resource_group" --yes
    if [ $? -ne 0 ]; then
        echo -e "${RED}Deletion of resource group ${resource_group} failed${NC}"
        return 1
    else
        echo -e "${GREEN}Resource group ${resource_group} deletion was successful${NC}"
        for keyVault in $keyVaults; do
            keyVaultPurge "$keyVault" "$waitForKVPurge"
        done
    fi
}

# Function to delete a resource by its ID
# It checks if the resource exists, and if it does, it deletes it
# If the resource is a Key Vault, it calls the keyVaultPurge function to purge the Key Vault
deleteResource() {
    local resource_id=$1
    local waitForKVPurge=$2

    # Check if the resource exists
    if ! az resource show --id "$resource_id" &>/dev/null; then
        echo -e "${RED}Resource ${resource_id} does not exist. Skipping deletion.${NC}"
        return 0
    fi

    # Delete the resource
    az resource delete --ids "$resource_id"
    if [ $? -ne 0 ]; then
        echo -e "${RED}Deletion of resource ${resource_id} failed${NC}"
        return 1
    else
        echo -e "${GREEN}Resource ${resource_id} deletion was successful${NC}"
    fi
    if [[ "$resource_id" == *"Microsoft.KeyVault/vaults"* ]]; then
        keyVault=$(echo "$resource_id" | awk -F'/vaults/' '{print $2}')
        echo "Key Vault name: $keyVault"
        echo -e "${YELLOW}Resource is a KV, purging Key Vault: ${keyVault}${NC}"
        keyVaultPurge "$keyVault" "$waitForKVPurge"
    fi
}

# Generic function to loop through a list of resources and call the appropriate deletion function
# It is designed to handle both resource groups and individual resources
# This is performed in parallel to speed up the deletion process
performDeletion() {
    local subscription=$1
    local resourceList=$2
    local deletionFunction=$3
    local waitForKVPurge=$4

    loginToSubscription "$SUBSCRIPTION_ID"
    if [ $? -ne 0 ]; then
        echo -e "${RED}Failed to login to subscription ${SUBSCRIPTION_ID}${NC}"
        return 1
    fi
    echo -e "--------------------------"
    echo -e "${YELLOW}Starting deletion process...${NC}"
    echo -e "--------------------------"
    for resource in $resourceList; do
        echo -e "${BLUE}Deleting: ${resource}${NC}"
        $deletionFunction "$resource" "$waitForKVPurge" &
        deletionPids+=($!)
    done

    for pid in "${deletionPids[@]}"; do
        if ! wait "$pid"; then
            echo "Error: Process with PID $pid failed."
            exit 1
        fi
    done
    echo -e "--------------------------"
    echo -e "${GREEN}All resources have been deleted successfully.${NC}"
    echo -e "--------------------------"
}

# Function to delete resources by tag 'deletionDate'.
# It checks all resource groups in the subscription for the tag and deletes them if the tag's value is less than or equal to today's date.
deletionByTag() {
    local waitForKVPurge=$1
    local tagPids=() # Array to hold PIDs of background processes
    local groupsToDelete=() # Array to hold resource groups to delete

    loginToSubscription "$SUBSCRIPTION_ID"
    if [ $? -ne 0 ]; then
        echo -e "${RED}Failed to login to subscription ${SUBSCRIPTION_ID}${NC}"
        return 1
    fi
    TODAYS_DATE=$(date +"%Y-%m-%d") # Format: YYYY-MM-DD
    # Get all resource groups in the subscription
    RESOURCE_GROUPS=$(az group list --query "[].name" -o tsv)
    if [ $? -ne 0 ]; then
        echo -e "${RED}Failed to retrieve resource groups${NC}"
        return 1
    fi

    echo -e "${YELLOW}Checking resource groups for tag 'deletionDate' set to today's date (${TODAYS_DATE})...${NC}"
    for rg in $RESOURCE_GROUPS; do
        local tagValue=$(az group show --name "$rg" --query "tags.deletionDate" -o tsv)
        if [ -n "$tagValue" ]; then
            if [ $(date -d "$tagValue" +%s) -le $(date -d "$TODAYS_DATE" +%s) ]; then
                echo "Resource group '$rg' has the deletionDate: $tagValue which is less than or equal to today's date ($TODAYS_DATE)."
                groupsToDelete+=("$rg")
            fi
        fi
    done

    groupsToDeleteString=$(printf "%s " "${groupsToDelete[@]}")
    if [ ${#groupsToDelete[@]} -eq 0 ]; then
        echo -e "${YELLOW}No resource groups found with deletionDate set to today's date (${TODAYS_DATE}).${NC}"
        return 0
    else
        echo -e "${YELLOW}Found ${#groupsToDelete[@]} resource groups to delete.${NC}"
        performDeletion "$SUBSCRIPTION_ID" "$groupsToDeleteString" deleteResourceGroup "$waitForKVPurge"
    fi
    echo -e "${GREEN}Finished checking resource groups for today's deletion date.${NC}"
}


# Function to handle deletion based on the deletion configuration type.
# ------------------------------------------------------------
# Required variables that should be set in the environment:
# SUBSCRIPTION_ID: The subscription ID current rollout is being performed in.
# LOCATION: The location for the current rollout.
# GLOBAL_SUBSCRIPTION_ID: The subscription ID for the global resources rollout.
# GLOBAL_LOCATION: The location for the global resources rollout.
# DELETE_CONFIG_TYPE: "Daily" or "Manual"
# DAILY_CONFIG_FILE: Path to the daily deletion configuration file.
# MANUAL_CONFIG_FILE: Path to the manual deletion configuration file.
# Function to read YAML configuration using yq
main() {
    az login --identity
    if [ $? -ne 0 ]; then
        echo -e "${RED}Failed to login ${NC}"
        exit 1
    fi

    if [ "$DELETE_CONFIG_TYPE" == "Daily" ]; then
        CONFIG_FILE=$DAILY_CONFIG_FILE
    elif [ "$DELETE_CONFIG_TYPE" == "Manual" ]; then
        CONFIG_FILE=$MANUAL_CONFIG_FILE
    else
        echo -e "${RED}Invalid DELETE_CONFIG_TYPE specified: $DELETE_CONFIG_TYPE${NC}"
        exit 1
    fi

    WAIT_FOR_KV_PURGE=$(yq ".settings.waitForKVPurge" "$CONFIG_FILE")
    DELETION_OPTION=$(yq ".settings.deletionOption" "$CONFIG_FILE")

    echo -e "${YELLOW}Wait for KV Purge: $WAIT_FOR_KV_PURGE${NC}"
    echo -e "${YELLOW}Deletion Option: $DELETION_OPTION${NC}"
    if [ "$DELETION_OPTION" == "byTag" ]; then
        echo -e "--------------------------"
        echo -e "${YELLOW}Starting deletion by tag...${NC}"
        echo -e "--------------------------"
        deletionByTag "$WAIT_FOR_KV_PURGE"
    elif [ "$DELETION_OPTION" == "byResourceGroupList" ] || [ "$DELETION_OPTION" == "byResourceIdList" ]; then
        DELETION_LIST=$(yq ".settings.deletionList" "$CONFIG_FILE")
        DELETION_LIST_COUNT=$(yq ".settings.deletionList | length" "$CONFIG_FILE")
        # Loop through each item in the deletionList
        for ((i=0; i<DELETION_LIST_COUNT; i++)); do
            # Extract subscription and resourceList for the current item
            subscription=$(yq ".settings.deletionList[$i].subscription" "$CONFIG_FILE")
            resourceList=$(yq ".settings.deletionList[$i].resourceList[]" "$CONFIG_FILE")
            if [ "$subscription" != "$SUBSCRIPTION_ID" ]; then
                if [ "$subscription" == "$GLOBAL_SUBSCRIPTION_ID" ] && [ "$LOCATION" == "$GLOBAL_LOCATION" ]; then
                    echo -e "${YELLOW}Performing deletion for global subscription as current rollout region matches.${NC}"
                else
                    echo -e "${YELLOW}Skipping subscription ${subscription} as it does not match current subscription.${NC}"
                    continue
                fi
            fi
            echo -e "${BLUE}Processing subscription: ${subscription} with resources: ${resourceList[*]}${NC}"
            if [ "$DELETION_OPTION" == "byResourceGroupList" ]; then
                echo -e "--------------------------"
                echo -e "${YELLOW}Starting deletion by resource group...${NC}"
                echo -e "--------------------------"
                performDeletion "$subscription" "$resourceList" deleteResourceGroup "$WAIT_FOR_KV_PURGE"
            elif [ "$DELETION_OPTION" == "byResourceIdList" ]; then
                echo -e "--------------------------"
                echo -e "${YELLOW}Starting deletion by resource ID list...${NC}"
                echo -e "--------------------------"
                performDeletion "$subscription" "$resourceList" deleteResource "$WAIT_FOR_KV_PURGE"
            fi
        done
    else 
        echo -e "${RED}Invalid deletion option specified: $DELETION_OPTION${NC}"
        exit 1
    fi
    echo -e "--------------------------"
    echo -e "${GREEN}All specified resources have been deleted successfully.${NC}"
    echo -e "--------------------------"
}


main