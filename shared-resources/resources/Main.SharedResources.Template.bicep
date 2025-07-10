targetScope = 'subscription'

@sys.description('The name for the resources.')
param resourcesName string

@sys.description('The subscription the resources are deployed to.')
param subscriptionId string

@sys.description('The location of the resource group the resources are deployed to.')
param location string

@sys.description('The name of the resource group the resources are deployed to.')
param resourceGroupName string

// keep vnet entirely outside the default AKS serviceCidr range (10.0.0.0/16) by default
// default network config for AKS managed cluster is Azure CNI Overlay
// more info: https://learn.microsoft.com/en-us/azure/aks/concepts-network-azure-cni-overlay
@sys.description('The address prefix for the VNet.')
param vnetAddressPrefix string = '10.1.0.0/16'

@sys.description('The address prefix for the AKS Managed Cluster subnet.')
param nodePoolAddressPrefix string = '10.1.10.0/24'

@sys.description('The address prefix for the Private Endpoint subnet.')
param peSubnetPrefix string = '10.1.20.0/24'

@sys.description('The address prefix for the deployment script subnet.')
param scriptSubnetPrefix string = '10.1.30.0/24'

@sys.description('Whether or not to use geneva monitoring.') // We need to add conditionals such that we deploy the correct monitoring resources. We use Geneva monitoring when deploying via ev2.
param useGenevaMonitoring bool = true

@sys.description('The date this resource group will be deleted on, with format YYYY-MM-DD. This is used to tag the resource group for deletion.')
param deletionDate string

module rg 'br/public:avm/res/resources/resource-group:0.2.3' = {
  name: '${resourceGroupName}Deploy'
  scope: subscription(subscriptionId)
  params: {
    name: resourceGroupName
    location: location
    tags: {
      deletionDate: deletionDate
    }
  }
}

module vnet 'br/public:avm/res/network/virtual-network:0.7.0' = {
  name: 'servicehubval-${resourcesName}-shared-resources-vnetDeploy'
  scope: resourceGroup(subscriptionId, resourceGroupName)

  params: {
    name: 'servicehubval-${resourcesName}-vnet'
    location: rg.outputs.location
    addressPrefixes: [
      vnetAddressPrefix
    ]
    subnets: [
      // subnet for aks managed cluster
      { name: 'aks-subnet', addressPrefix: nodePoolAddressPrefix, defaultOutboundAccess: false }
      // subnet for private endpoints for all service specific resources
      { name: 'pe-subnet', addressPrefix: peSubnetPrefix, defaultOutboundAccess: false }
      // subnet for deployment scripts with container instance delegation
      { name: 'script-subnet', addressPrefix: scriptSubnetPrefix, defaultOutboundAccess: false, delegation: 'Microsoft.ContainerInstance/containerGroups' }
    ]
  }
}

// used for secure private name resolution for SQL Server private endpoint
// ensures server is accessed from within virtual network without exposure to public internet
module sqlPrivateDnsZone 'br/public:avm/res/network/private-dns-zone:0.7.0' = {
  name: 'servicehubval-${resourcesName}-sqlPrivatelinkZoneDeploy'
  scope: resourceGroup(subscriptionId, resourceGroupName)
  params: {
    // The SQL private DNS zone must match the Azure SQL privatelink hostname suffix
    // Example resulting name: "privatelink.database.windows.net"
    // more info here: https://learn.microsoft.com/en-us/azure/private-link/private-endpoint-dns
    name: 'privatelink${environment().suffixes.sqlServerHostname}'
    location: 'global'

    virtualNetworkLinks: [
      {
        name: 'servicehubval-${resourcesName}-vnet-link'
        virtualNetworkResourceId: vnet.outputs.resourceId
      }
    ]
  }
}

// used for secure private name resolution for Key Vault private endpoint
// Used for one keyvault in Main.MsftMonitoring.Template.bicep, but can be used for any key vault
// within the vnet
module keyVaultPrivateDnsZone 'br/public:avm/res/network/private-dns-zone:0.7.0' = {
  name: 'servicehubval-${resourcesName}-kvPrivatelinkZoneDeploy'
  scope: resourceGroup(subscriptionId, resourceGroupName)
  params: {
    // The Key Vault private DNS zone must match the Azure Key Vault privatelink hostname suffix
    name: 'privatelink.vaultcore.azure.net'
    location: 'global'

    virtualNetworkLinks: [
      {
        name: 'servicehubval-${resourcesName}-vnet-kv-link'
        virtualNetworkResourceId: vnet.outputs.resourceId
      }
    ]
  }
}

// used for secure private name resolution for Storage Account file private endpoint
module storageFilePrivateDnsZone 'br/public:avm/res/network/private-dns-zone:0.7.0' = {
  name: 'servicehubval-${resourcesName}-saFilePrivatelinkZoneDeploy'
  scope: resourceGroup(subscriptionId, resourceGroupName)
  params: {
    // The Storage private DNS zone for file must match the Azure Storage privatelink hostname suffix
    name: 'privatelink.file.${environment().suffixes.storage}'
    location: 'global'

    virtualNetworkLinks: [
      {
        name: 'servicehubval-${resourcesName}-vnet-sa-file-link'
        virtualNetworkResourceId: vnet.outputs.resourceId
      }
    ]
  }
}

var aksSubnetIndex = indexOf(vnet.outputs.subnetNames, 'aks-subnet')
var aksSubnetId = vnet.outputs.subnetResourceIds[aksSubnetIndex]

module aks 'br/public:avm/res/container-service/managed-cluster:0.8.1' = {
  name: 'servicehubval-${resourcesName}-shared-resources-clusterDeploy'
  scope: resourceGroup(subscriptionId, resourceGroupName)
  params: {
    // Required parameters
    name: 'servicehubval-${resourcesName}-cluster'
    location: rg.outputs.location
    autoNodeOsUpgradeProfileUpgradeChannel: 'NodeImage'
    dnsPrefix: resourcesName
    primaryAgentPoolProfiles: [
      {
        name: 'agentpool'
        count: 3 // agentCount
        vmSize: 'standard_d2s_v3'
        osType: 'Linux'
        osSKU: 'AzureLinux'
        mode: 'System'
        availabilityZones: [] // use this when availability zones ar not availabile in region
        vnetSubnetResourceId: aksSubnetId
      }
    ]
    disableLocalAccounts: false
    managedIdentities: {
      systemAssigned: true
    }
    publicNetworkAccess: 'Enabled'
    omsAgentEnabled: true
    monitoringWorkspaceResourceId: !useGenevaMonitoring ? workspace.outputs.resourceId : null
    omsAgentUseAADAuth: true
    enableOidcIssuerProfile: true
    enableWorkloadIdentity: true
    istioServiceMeshEnabled: true
    enableKeyvaultSecretsProvider: useGenevaMonitoring ? true : false
    istioServiceMeshRevisions: ['asm-1-24']
  }
}

module dataCollectionRuleAssociation 'br:servicehubregistry.azurecr.io/bicep/modules/data-collection-rule-association:v5' = if (!useGenevaMonitoring) {
  name: 'servicehub-${resourcesName}-shared-resources-dcr-associationDeploy'
  scope: resourceGroup(subscriptionId, resourceGroupName)
  params: {
    dataCollectionRuleId: dataCollectionRule.outputs.resourceId
    aksName: aks.outputs.name
  }
}

module workspace 'br/public:avm/res/operational-insights/workspace:0.3.4' = if (!useGenevaMonitoring) {
  name: 'servicehubval-${resourcesName}-workspaceDeploy'
  scope: resourceGroup(subscriptionId, resourceGroupName)
  params: {
    name: 'servicehubval-${resourcesName}-workspace'
    location: rg.outputs.location
  }
}

var streams = ['Microsoft-ContainerLogV2']
module dataCollectionRule 'br/public:avm/res/insights/data-collection-rule:0.1.2' = if (!useGenevaMonitoring) {
  name: 'servicehubval-${resourcesName}-data-collection-ruleDeploy'
  scope: resourceGroup(subscriptionId, resourceGroupName)
  params: {
    name: 'servicehubval-${resourcesName}-data-collection-rule'
    location: rg.outputs.location
    dataFlows: [
      {
        streams: streams
        destinations: [
          'ciworkspace'
        ]
      }
    ]
    dataSources: {
      extensions: [
        {
          name: 'ContainerInsightsExtension'
          streams: streams
          extensionSettings: {
            dataCollectionSettings: {
              enableContainerLogV2: true
              interval: '1m'
              namespaceFilteringMode: 'Exclude'
            }
          }
          extensionName: 'ContainerInsights'
        }
      ]
    }
    destinations: {
      logAnalytics: [
        {
          workspaceResourceId: workspace.outputs.resourceId
          name: 'ciworkspace'
        }
      ]
    }
  }
}

// TODO: potentially make unique to cloud
module acr 'br/public:avm/res/container-registry/registry:0.1.1' = {
  name: 'servicehubval-${resourcesName}-${location}acrDeploy'
  scope: resourceGroup(subscriptionId, resourceGroupName)
  params: {
    name: 'servicehubval${resourcesName}${location}acr'
    location: rg.outputs.location
    roleAssignments: [
      {
        principalId: aks.outputs.kubeletIdentityObjectId
        principalType: 'ServicePrincipal'
        roleDefinitionIdOrName: 'AcrPull'
      }
    ]
  }
}

module serviceBusNamespace 'br/public:avm/res/service-bus/namespace:0.9.0' = {
  name: 'servicehubval-${resourcesName}-${location}-sb-nsDeploy'
  scope: resourceGroup(subscriptionId, resourceGroupName)
  params: {
    name: 'servicehubval-${resourcesName}-${location}-sb-ns'
    location: rg.outputs.location
    queues: [
      {
        name: 'servicehubval-${resourcesName}-queue'
      }
    ]
    skuObject: {
      name: 'Basic'
    }
    zoneRedundant: false
  }
}

output aksSecretStoreClientId string = useGenevaMonitoring
  ? aks.outputs.addonProfiles.azureKeyvaultSecretsProvider.identity.clientId
  : ''
output aksResourceId string = aks.outputs.resourceId
