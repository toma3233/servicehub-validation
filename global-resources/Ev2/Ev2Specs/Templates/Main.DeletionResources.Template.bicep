targetScope = 'subscription'

@sys.description('The name for the resources.')
param resourcesName string

@sys.description('The subscription the resources are deployed to.')
param subscriptionId string

@sys.description('The location of the resource group the resources are deployed to.')
param location string

@sys.description('The name for the resource group.')
param resourceGroupName string

// Resource group deployment
module rg 'br/public:avm/res/resources/resource-group:0.2.3' = {
  name: '${resourceGroupName}Deploy'
  scope: subscription(subscriptionId)
  params: {
    name: resourceGroupName
    location: location
  }
}

// Managed Identity Deployment. This is used to delete resources across subscriptions in the same environment
module identityUsedByDeletion 'br/public:avm/res/managed-identity/user-assigned-identity:0.2.1' = {
  name: 'servicehubval-${resourcesName}-delete-identityDeploy'
  scope: resourceGroup(subscriptionId, resourceGroupName)
  params: {
    // Name needs to be unique in the entire subscription, thus why we add the `${resourcesName}` to avoid conflicts from different developers.
    name: 'servicehubval-${resourcesName}-delete-identity'
    location: rg.outputs.location
  }
}

// Role Assignment Deployment. This is used to assign the Managed Identity the Owner role on the subscription, so it can delete resources within the global subscription.
module deleteRoleAssignment 'br:servicehubregistry.azurecr.io/bicep/modules/subscription-role-assignment:v6' = {
  name: 'servicehubval-deletera${location}Deploy'
  scope: subscription(subscriptionId)
  params: {
    principalId: identityUsedByDeletion.outputs.principalId
    description: 'servicehubval-${resourcesName}-delete--owner-role-assignment'
    roleDefinitionIdOrName: 'Owner'
    principalType: 'ServicePrincipal'
    subscriptionId: subscriptionId
  }
}
