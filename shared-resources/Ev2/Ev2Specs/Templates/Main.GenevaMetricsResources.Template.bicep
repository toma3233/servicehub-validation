// -----------------------------------------------------------------------------
// Geneva Metrics Resources Deployment
// -----------------------------------------------------------------------------
// This template is an entry point for deploying shared Geneva metrics resources
// using the main template. It passes all required parameters for a multi-region,
// multi-AKS, and multi-AMW deployment.
// -----------------------------------------------------------------------------

@sys.description('The unique name for this deployment instance.')
param resourcesName string

@sys.description('The global resources name prefix for the resources.')
param globalResourcesName string

@sys.description('The resource group where resources will be deployed.')
param resourceGroupName string

@sys.description('The subscription where resources will be deployed.')
param subscriptionId string

@sys.description('The subscription containing the global Azure Monitor Workspaces.')
param globalSubscriptionId string

@sys.description('The name of the AKS cluster to associate with DCRs.')
param aksClusterName string

targetScope = 'subscription'

resource rg 'Microsoft.Resources/resourceGroups@2021-04-01' existing = {
  name: resourceGroupName
  scope: subscription(subscriptionId)
}

var existingAzureMonitorWorkspaces = [
  {
    workspaceName: 'servicehubval-${globalResourcesName}-eastus'
    workspaceLocation: 'eastus'
  }
]

module sharedMetrics 'br:servicehubregistry.azurecr.io/bicep/modules/first-party-metrics-shared:v1' = {
  name: take('servicehubval-${resourcesName}-shared-metricsDeploy', 64)
  scope: resourceGroup(subscriptionId, resourceGroupName)
  params: {
    resourcesName: resourcesName
    globalSubscriptionId: globalSubscriptionId
    globalResourceGroupName: 'servicehubval-${globalResourcesName}-global-metrics-rg'
    productShortName: 'servicehubval'
    existingMonitorWorkspaces: existingAzureMonitorWorkspaces
    aksClusterName: aksClusterName
  }
}
