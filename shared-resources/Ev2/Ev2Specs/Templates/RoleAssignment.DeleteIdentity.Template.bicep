targetScope = 'subscription'

@sys.description('The subscription the resources are deployed to.')
param subscriptionId string

@sys.description('The location of the resource group the resources are deployed to.')
param location string

@sys.description('The subscription the global resources are deployed to.')
param globalSubscriptionId string

@sys.description('The name for the global resources.')
param globalResourcesName string

@sys.description('The name of the resource group the resources are deployed to.')
var globalResourceGroupName string = 'servicehubval-${globalResourcesName}-global-delete-rg'

// Created by global resources rollout and sits in the global subscription. Do not touch this resource from this template.
resource deleteManagedIdentity 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' existing = {
  name: 'servicehubval-${globalResourcesName}-delete-identity'
  scope: resourceGroup(globalSubscriptionId, globalResourceGroupName)
}

// Role Assignment Deployment. This is used to assign the Managed Identity the Owner role in all the subscriptions we perform 
// shared-resources deployments in such that it has the necessary capabilities to delete resources in those subscriptions.
module deleteRoleAssignment 'br:servicehubregistry.azurecr.io/bicep/modules/subscription-role-assignment:v6' = {
  name: 'servicehubval-deletera${location}Deploy'
  scope: subscription(subscriptionId)
  params: {
    principalId: deleteManagedIdentity.properties.principalId
    description: 'servicehubval-${globalResourcesName}-delete--owner-role-assignment'
    roleDefinitionIdOrName: 'Owner'
    principalType: 'ServicePrincipal'
    subscriptionId: subscriptionId
  }
}
