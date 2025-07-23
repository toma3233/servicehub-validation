// Geneva 1P Metrics Setup Test Deployment
// Provisions resources for Geneva 1P metrics: Azure Monitor workspaces, Grafana workspace, managed identity, and role assignments.
//
// Resource Naming:
//   - Grafana workspace: <23 chars, first 11 from 'servicehubval', first 10 from resourcesName

@sys.description('The name for the resources.')
param resourcesName string

@sys.description('The subscription where the resources are deployed.')
param subscriptionId string

@sys.description('The name of the resource group where the resources are deployed.')
param resourceGroupName string

@sys.description('The Azure region for the resource group.')
param resourceGroupLocation string

@sys.description('The tenant id of the azure monitor workspace. This is the tenant that will be used to access the azure monitor workspace from grafana.')
param tenantId string

@sys.description('The object ID of the MSFT tenant (corp) security group for Azure Monitor workspace viewer and Grafana Admin/Viewer access.')
param corpAdminSecurityGroupId string

targetScope = 'subscription'

// Resource group deployment
module resourceGroup 'br/public:avm/res/resources/resource-group:0.2.3' = {
  name: '${resourceGroupName}MetricsDeploy'
  scope: subscription(subscriptionId)
  params: {
    name: resourceGroupName
    location: resourceGroupLocation
  }
}

// Main Geneva Metrics Resources Deployment
var monitorWorkspaceLocations array = ['eastus']
var grafanaWorkspaceLocations array = ['eastus', 'westus']
module metrics 'br:servicehubregistry.azurecr.io/bicep/modules/first-party-metrics-global:v3' = {
  name: 'servicehubval-${resourcesName}-MetricsResourcesDeploy'
  scope: subscription(subscriptionId)
  params: {
    resourcesName: resourcesName
    subscriptionId: subscriptionId
    monitorWorkspaces: [for location in monitorWorkspaceLocations: {
      workspaceName: 'servicehubval-${resourcesName}-${location}'
      location: location
    }]
    tenantId: tenantId
    corpAdminSecurityGroupId: corpAdminSecurityGroupId
    resourceGroupName: resourceGroup.outputs.name // This enforces the resource group to be created before the metrics resources
    productShortName: 'servicehubval'
    grafanaWorkspaces: [for location in grafanaWorkspaceLocations: {
      name: '${take('servicehubval', 10)}${take(resourcesName, 8)}${take(location, 5)}'
      location: location
    }]
  }
}
