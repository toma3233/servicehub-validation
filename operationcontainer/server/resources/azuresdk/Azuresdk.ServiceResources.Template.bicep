targetScope = 'subscription'

@sys.description('The name for the resources.')
param resourcesName string

@sys.description('The subscription the resources are deployed to.')
param subscriptionId string

@sys.description('The location of the resource group the resources are deployed to.')
param location string

@sys.description('The name of the resource group the resources are deployed to.')
param resourceGroupName string

// This resource is shared and defined in resources/Main.SharedResources.Template.bicep in shared-resources directory; we only reference it here. Do not remove `existing` syntax.
resource rg 'Microsoft.Resources/resourceGroups@2021-04-01' existing = {
  name: resourceGroupName
  scope: subscription(subscriptionId)
}

resource aks 'Microsoft.ContainerService/managedClusters@2024-09-02-preview' existing = {
  name: 'servicehubval-${resourcesName}-cluster'
  scope: resourceGroup(subscriptionId, resourceGroupName)
}

resource vnet 'Microsoft.Network/virtualNetworks@2024-01-01' existing = {
  name: 'servicehubval-${resourcesName}-vnet'
  scope: resourceGroup(subscriptionId, resourceGroupName)
}

resource peSubnet 'Microsoft.Network/virtualNetworks/subnets@2024-01-01' existing = {
  parent: vnet
  name: 'pe-subnet'
}

resource sqlPrivateDnsZone 'Microsoft.Network/privateDnsZones@2020-06-01' existing = {
  name: 'privatelink${environment().suffixes.sqlServerHostname}'
  scope: resourceGroup(subscriptionId, resourceGroupName)
}

var serviceAccountNamespace = 'servicehubval-operationcontainer-server'
var serviceAccountName = 'servicehubval-operationcontainer-server'
module managedIdentity 'br/public:avm/res/managed-identity/user-assigned-identity:0.2.1' = {
  name: 'servicehubval-operationcontainer-managed-identityDeploy'
  scope: resourceGroup(subscriptionId, resourceGroupName)
  params: {
    name: 'servicehubval-${resourcesName}-${location}-operationcontainer-managedIdentity'
    location: rg.location
    federatedIdentityCredentials: [
      {
        name: 'servicehubval-operationcontainer-fedIdentity'
        issuer: aks.properties.oidcIssuerProfile.issuerURL
        subject: 'system:serviceaccount:${serviceAccountNamespace}:${serviceAccountName}'
        audiences: ['api://AzureADTokenExchange']
      }
    ]
  }
}

// TODO: Migrate to use bicep module registry. Current bicep registry module is management group scoped but we use subscription scoped.
module azureSdkRoleAssignment 'br:servicehubregistry.azurecr.io/bicep/modules/subscription-role-assignment:v6' = {
  name: 'servicehubval-operationcontainerazuresdkra${location}Deploy'
  scope: subscription(subscriptionId)
  params: {
    principalId: managedIdentity.outputs.principalId
    description: 'servicehubval-operationcontainer-${resourcesName}-contributor-azuresdk-role-assignment'
    roleDefinitionIdOrName: 'Contributor'
    principalType: 'ServicePrincipal'
    subscriptionId: subscriptionId
  }
}

//TODO(mheberling): SQL server can only add other users to the db (after the admin is set) via SQL users.
// Look into using SQL Managed instance or setting the admin managed identity to the pods.
module server 'br/public:avm/res/sql/server:0.9.1' = {
  name: 'operationcontainer-${resourcesName}-${location}-serverDeploy'
  scope: resourceGroup(subscriptionId, resourceGroupName)
  params: {
    // Required parameters
    name: 'operationcontainer-${resourcesName}-${location}-sql-server'
    location: rg.location
    // Non-required parameters
    administrators: {
      azureADOnlyAuthentication: true
      login: 'myspn'
      principalType: 'Application'
      sid: managedIdentity.outputs.clientId
    }
    databases: [
      {
        name: 'operationcontainer-${resourcesName}-sql-db'
        zoneRedundant: false
      }
    ]
    // Create a private endpoint for secure, private communication with SQL server
    privateEndpoints: [
      {
        service: 'sqlServer'
        subnetResourceId: peSubnet.id
        privateDnsZoneGroup: {
          name: 'default'
          privateDnsZoneGroupConfigs: [
            {
              name: 'default'
              privateDnsZoneResourceId: sqlPrivateDnsZone.id
            }
          ]
        }
      }
    ]
    publicNetworkAccess: 'Disabled'
  }
}

@sys.description('Client Id of the managed identity.')
output clientId string = managedIdentity.outputs.clientId
