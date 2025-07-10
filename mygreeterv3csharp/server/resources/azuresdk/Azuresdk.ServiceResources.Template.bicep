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

var serviceAccountNamespace = 'servicehubval-mygreeterv3csharp-server'
var serviceAccountName = 'servicehubval-mygreeterv3csharp-server'
module managedIdentity 'br/public:avm/res/managed-identity/user-assigned-identity:0.2.1' = {
  name: 'servicehubval-mygreeterv3csharp-managed-identityDeploy'
  scope: resourceGroup(subscriptionId, resourceGroupName)
  params: {
    name: 'servicehubval-${resourcesName}-${location}-mygreeterv3csharp-managedIdentity'
    location: rg.location
    federatedIdentityCredentials: [
      {
        name: 'servicehubval-mygreeterv3csharp-fedIdentity'
        issuer: aks.properties.oidcIssuerProfile.issuerURL
        subject: 'system:serviceaccount:${serviceAccountNamespace}:${serviceAccountName}'
        audiences: ['api://AzureADTokenExchange']
      }
    ]
  }
}

// TODO: Migrate to use bicep module registry. Current bicep registry module is management group scoped but we use subscription scoped.
module azureSdkRoleAssignment 'br:servicehubregistry.azurecr.io/bicep/modules/subscription-role-assignment:v6' = {
  name: 'servicehubval-mygreeterv3csharpazuresdkra${location}Deploy'
  scope: subscription(subscriptionId)
  params: {
    principalId: managedIdentity.outputs.principalId
    description: 'servicehubval-mygreeterv3csharp-${resourcesName}-contributor-azuresdk-role-assignment'
    roleDefinitionIdOrName: 'Contributor'
    principalType: 'ServicePrincipal'
    subscriptionId: subscriptionId
  }
}

@sys.description('Client Id of the managed identity.')
output clientId string = managedIdentity.outputs.clientId
