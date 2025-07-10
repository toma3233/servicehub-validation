// SLNM Azure Resource for SLNM configuration commit
// This file provisions the necessary resources for SLNM commit operation. It creates a resource group and deploys a deployment script to commit SLNM configuration.

@sys.description('Azure AD tenant ID for authentication')
param tenantId string

@sys.description('The subscription the resources are deployed to.')
param subscriptionId string

@sys.description('Region where the SLNM commit will run')
param location string

@sys.description('The name of the resource group to be created or referenced')
param resourceGroupName string

@sys.description('Subscription ID of the Network Manager instance')
param networkManagerSubscriptionId string

@sys.description('Resource group name of the Network Manager instance')
param networkManagerResourceGroup string

@sys.description('Name of the Network Manager instance')
param networkManagerName string

@sys.description('Configuration ID(s) for the SLNM commit')
param slnmConfigIds string

@sys.description('User-assigned managed identity resource ID used by the pipeline')
param pipelineIdentityId string

targetScope = 'subscription'

resource rg 'Microsoft.Resources/resourceGroups@2021-04-01' existing = {
  name: resourceGroupName
  scope: subscription(subscriptionId)
}

module slnmCommitScript 'br/public:avm/res/resources/deployment-script:0.5.1' = {
  name: 'slnm-commit-script-deployment-${location}'
  scope: resourceGroup(subscriptionId, rg.name)
  params: {
    name: 'Commit_SLNM_${location}'
    location: location
    managedIdentities: {
      userAssignedResourceIds: [
        pipelineIdentityId
      ]
    }
    kind: 'AzurePowerShell'
    azPowerShellVersion: '8.3'
    retentionInterval: 'PT1H'
    timeout: 'PT1H'
    environmentVariables: [
      {
        name: 'tenantId'
        value: tenantId
      }
      {
        name: 'location'
        value: location
      }
      {
        name: 'networkManagerSubscriptionId'
        value: networkManagerSubscriptionId
      }
      {
        name: 'networkManagerResourceGroup'
        value: networkManagerResourceGroup
      }
      {
        name: 'networkManagerName'
        value: networkManagerName
      }
      {
        name: 'slnmConfigIds'
        value: slnmConfigIds
      }
      {
        name: 'pipelineIdentityId'
        value: pipelineIdentityId
      }
    ]
    scriptContent: '''
    $null = Login-AzAccount -Identity -Tenant $env:tenantId -Subscription $env:networkManagerSubscriptionId

    $deployment = @{
        ResourceGroupName = $env:networkManagerResourceGroup
        Name            = $env:networkManagerName
        ConfigurationId = $env:slnmConfigIds
        TargetLocation  = $env:location
        CommitType      = 'SecurityAdmin'
    }

    try {
        Deploy-AzNetworkManagerCommit @deployment -ErrorAction Stop
    }
    catch {
        Write-Error "Deployment failed with error: $_"
        throw "Deployment failed with error: $_"
    }
    '''
  }
}
