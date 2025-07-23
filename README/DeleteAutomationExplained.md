# Deletion Automation Steps in Ev2 - Explained

This document provides a detailed explanation of the deletion process in Ev2.

## Table of Contents

1. [Overview of Deletion Steps in Ev2](#overview-of-deletion-steps-in-ev2)
2. [Global Resources](#global-resources)
3. [Shared Resources](#shared-resources)
4. [Resource Deletion](#resource-deletion)
    - [Deletion Options](#deletion-options)
    - [Configuration Setup](#configuration-setup)
5. [Running the Deletion Rollout](#running-the-deletion-rollout)

## Overview of Deletion Steps in Ev2

This document provides a high-level overview of the deletion process.

The deletion automation consists of three steps:
1. **Managed Identity Creation**: Performed during the global-resources rollout.
2. **Role Assignment Creation**: Performed during both the global and shared-resources rollouts.
3. **Deletion Script Execution**: Performed during the resource-deletion rollout.

By automating these steps, we achieve a hands-free approach to deleting resources via managed SDP in both test and production environments.

## Global Resources

This step is part of the global-resources rollout and is independent of other steps. It does not require any resource providers and is completely isolated from Geneva resources.

![Ev2 Delete Resources Step - global-resources rollout](images/ev2global_DeployResourcesUsedByDeletion.png)

**DeployResourcesUsedByDeletion**: This rollout step deploys three resources:
1. **Resource Group**: Stores the managed identity and is completely isolated from Geneva resources.
2. **Managed Identity**: Used to run the deletion script in the delete service. It is created once and resides in the global subscription.
3. **Role Assignment**: Grants the managed identity owner access to the global subscription, providing the necessary permissions to delete resources in the global subscription.

## Shared Resources

This step is part of the shared-resources rollout and is independent of other steps. It does not require any resource providers and is completely isolated from shared resources deployments.

![Ev2 Delete Resources Step - shared-resources rollout](images/ev2shared_DeployResourcesUsedByDeletion.png)

**DeployResourcesUsedByDeletion**: This rollout step deploys one resource and references another:
1. **Existing Managed Identity**: References the managed identity created during the global-resources rollout.
2. **Role Assignment**: Grants the managed identity owner access to all subscriptions provisioned or used in each region rollout for shared-resources, enabling it to delete resources in those subscriptions.

> **_NOTE:_** Due to the reference to resources created in the global subscription, certain information about the global resources is required. As a result, the configuration files (Configuration.json for both Shared-Resources and Resource-Deletion) now require "globalSubscriptionId" and "globalResourcesName". The 'globalResourcesName' is hardcoded to match 'resourcesName'; however, 'resourcesName' is tied to a PR, while 'globalResourcesName' remains constant as it is not unique.
> - **Test**: In test environments, the globalSubscriptionId is the same as your backfilledSubId.
> - **Prod**: In production, the globalSubscriptionId is passed as backfilledProdGlobalSubscriptionId. If your team provisioned a global subscription, copy its ID into the Configuration.json files for both Shared-Resources and Resource-Deletion. If you used a backfilled subscription, the value is templated from your common-config.yaml file.

## Resource Deletion

This section excludes the setup of build and release pipelines. If you haven't set them up, refer to Ev2_README.md for more information.

The deletion rollout focuses on the **subscription** from which resources are being deleted:
- **Test**:
    - The deletion rollout is performed in a single region, as there is only one subscription to delete resources from.
- **Production**:
    - The deletion rollout mimics the region-agnostic rollout for shared-resources, releasing to the same regions to enable resource deletion by region.
    - Each subscription stores resources for its own region. If a subscription or resource group stores data for multiple regions, this deletion method violates SDP.
    > **_NOTE:_** The global subscription in production is a special case. Although it is not tied to shared-resources, it requires deletion capabilities. The globalLocation configuration variable allows resource deletion from the global subscription during the rollout for the same region.

> **_NOTE:_** Deleting a key vault (whether by resource ID or as part of a resource group) will **PURGE** the key vault completely to avoid issues with redeploying key vaults with the same name. Deleted key vaults cannot be recovered.

### Deletion Options

- **Deletion by Tag (byTag)**:
    - Iterates through each resource group in the subscription, checks the deletionDate tag, and deletes the resource group if the tag's value is less than or equal to today's date.
    - Best suited for wiping test resources created as unique resources for pull requests.

- **Deletion by Lists**:
    > **_NOTE:_** This strategy allows you to specify exactly which resources to delete without requiring a tag. It supports deletion from multiple subscriptions. The list is passed via the configuration. The script checks if the subscription in the configuration matches the current rollout's subscription before performing the deletion.

    - **Deletion by Resource Group List (byResourceGroupList)**:
        - Similar to byTag but allows you to list specific resource groups for deletion without requiring a tag.
        - Best suited for wiping production resources.

    - **Deletion by Resource ID List (byResourceIdList)**:
        - Provides the most flexibility, allowing you to list individual resources in any subscription for deletion.
        - Suitable for both test and production environments.

### Configuration Setup

The configuration requires three settings:

1. **waitForKVPurge**: Set to `true` to wait for key vaults to purge, or `false` to run the purge in the background.
2. **deletionOption**: Choose from `byTag`, `byResourceGroupList`, or `byResourceIdList`.
3. **deletionList**: A list containing:
    - **subscription**: The subscription where the resources exist. Supports multiple subscriptions for both test and production.
    - **resourceList**: A list of resource groups or resource IDs (cannot mix resource groups and resource IDs).

Examples of deletion configurations for byResourceGroupList and byResourceIdList can be found in the resource-deletion folder under SampleConfigs/. Below is a sample:

```yaml
waitForKVPurge: true
deletionOption: "byResourceGroupList"
deletionList:
  - subscription: "a447fa09-82a5-4123-8d83-198b46a21b00"
    resourceList:
      - "servicehubval-ev2-sec-rg"
      - "servicehubval-ev2-sg-rg"
      - "servicehubval-ev2-sy-rg"
      - "servicehubval-ev2-bn-rg"
      - "servicehubval-ev2-global-rg"
  - subscription: "dab1af7c-1826-4fa2-a8e3-0d72c0d88f13"
    resourceList:
      - "servicehubval-ev2-sec-rg"
      - "servicehubval-ev2ta2-sec-rg"
```


### Running the Deletion Rollout
There are two rollout strategies for running the Ev2 Build Pipeline for resource-deletion. The configurations vary depending on the strategy. Upon running the "[Build] resource-deletion" pipeline, you can choose either the **Daily Cleanup by Tag for Test Environment** or **Manual Deletion by Configured List for All Environments** configuration type.

![Delete ev2 build pipeline runtime configuration options](images/deleteruntimeoptions.png)

The choice is routed into deleteConfigType in Configuration.json for the Ev2 specs. The deletion script selects the appropriate configuration file based on this option.

- **Daily Cleanup by Tag for Test Environment**:
    - Default configuration for daily runs to delete resources in the test subscription.
    - Uses the byTag strategy to consistently remove unused resources.
    - Only applicable for test environments.
    - Configuration file: Shell/Daily.Delete.config.yaml
- **Manual Deletion by Configured List for All Environments**:
    - Allows users to specify the deletion option and list of resources to delete.
    - Users must check in the resources they want to delete, maintaining a history of deleted resources.
    - Applicable for both test and production environments.
    - Configuration file: Shell/Manual.Delete.config.yaml

#### Ev2
- Always run the "[Build] resource-deletion" pipeline regardless of the environment or tenant where deletion is required. Deletion will automatically occur in the test environment upon running this build, provided there are resources to delete.
- Run the "[Release] resource-deletion" pipeline only if there are resources to delete in production.