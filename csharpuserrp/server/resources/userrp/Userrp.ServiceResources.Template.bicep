// This file provisions specific resources for the Userrp setup.
// TODO: Most of the resources here are same as Geneva setup, work with Christine to see if we can turn this into a reusable module.
@sys.description('The name for the resources.')
param resourcesName string

@sys.description('The subscription the resources are deployed to.')
param subscriptionId string

@sys.description('The location of the resource group the resources are deployed to.')
param location string

@sys.description('The name of the resource group the resources are deployed to.')
param resourceGroupName string

@sys.description('Subject alternative name for the certificate. This should satisfy the registered domain that is registered in OneCert.')
param san string

@sys.description('Registered in OneCert.')
param cn string

@sys.description('The prefix for the DNS label.')
param dnsLabelPrefix string

targetScope = 'subscription'

// Managed identity to create certificate for Userrp service TLS termination.
module scriptIdentity 'br/public:avm/res/managed-identity/user-assigned-identity:0.4.0' = {
  name: 'servicehubval-${resourcesName}-csharpuserrp-script-identityDeploy'
  scope: resourceGroup(subscriptionId, resourceGroupName)
  params: {
    name: 'servicehubval-${resourcesName}-userrp-setup-script-identity'
    // Non-required parameters
    location: location
  }
}

resource aks 'Microsoft.ContainerService/managedClusters@2024-09-02-preview' existing = {
  name: 'servicehubval-${resourcesName}-cluster'
  scope: resourceGroup(subscriptionId, resourceGroupName)
}

// Key Vault to store the certificate for the Userrp service TLS termination.
// The max length of the name is 24 characters. The name must be globally unique.
var servicePrefix = substring('servicehubval', 0, min(8, length('servicehubval')))
var resourceNamePrefix = substring(resourcesName, 0, min(8, length(resourcesName)))
var regionPrefix = substring(location, 0, min(8, length(location)))
var truncatedKeyVaultName = '${servicePrefix}${resourceNamePrefix}${regionPrefix}'

// Reference the Virtual Network and PE Subnet for private endpoint setup
resource vnet 'Microsoft.Network/virtualNetworks@2024-01-01' existing = {
  name: 'servicehubval-${resourcesName}-vnet'
  scope: resourceGroup(subscriptionId, resourceGroupName)
}

resource peSubnet 'Microsoft.Network/virtualNetworks/subnets@2024-01-01' existing = {
  parent: vnet
  name: 'pe-subnet'
}

// Reference existing private DNS zones created in SharedResources template
// This zone can be used by any keyvault within the same vnet
resource keyVaultPrivateDnsZone 'Microsoft.Network/privateDnsZones@2020-06-01' existing = {
  name: 'privatelink.vaultcore.azure.net'
  scope: resourceGroup(subscriptionId, resourceGroupName)
}

resource storageFilePrivateDnsZone 'Microsoft.Network/privateDnsZones@2020-06-01' existing = {
  name: 'privatelink.file.${environment().suffixes.storage}'
  scope: resourceGroup(subscriptionId, resourceGroupName)
}

// Reference the existing script subnet created in SharedResources template
resource scriptSubnet 'Microsoft.Network/virtualNetworks/subnets@2024-01-01' existing = {
  parent: vnet
  name: 'script-subnet'
}

module keyvault 'br/public:avm/res/key-vault/vault:0.9.0' = {
  name: 'servicehubval-${resourcesName}-csharpuserrp-keyvaultDeploy'
  scope: resourceGroup(subscriptionId, resourceGroupName)
  params: {
    // Required parameters
    name: truncatedKeyVaultName
    // Non-required parameters
    enablePurgeProtection: false
    publicNetworkAccess: 'Disabled' 
    privateEndpoints: [      {
        name: '${truncatedKeyVaultName}-pe'
        service: 'vault'
        subnetResourceId: peSubnet.id
        privateDnsZoneGroup: {
          privateDnsZoneGroupConfigs: [
            {
              name: 'default'
              privateDnsZoneResourceId: keyVaultPrivateDnsZone.id
            }
          ]
        }
      }
    ]
    roleAssignments: [
      {
        principalId: aks.properties.addonProfiles.azureKeyvaultSecretsProvider.identity.objectId
        roleDefinitionIdOrName: 'Key Vault Secrets User' // Allows read access to secrets in the key vault, which is required for the AKS Azure Key Vault Secrets Provider addon.
        principalType: 'ServicePrincipal'
      }
      {
        principalId: scriptIdentity.outputs.principalId
        roleDefinitionIdOrName: 'Key Vault Certificates Officer' // Perform any action on the certificates of a key vault, excluding reading the secret and key portions, and managing permissions. Only works for key vaults that use the 'Azure role-based access control' permission model.
        principalType: 'ServicePrincipal'
      }
    ]
  }
}

// Script to create the certificate for the Userrp service TLS termination.
var issuer = 'OneCertV2-PublicCA'
var sub = 'CN=${cn}'
var certName='servicehubval${resourcesName}userrpcert'
module certCreationScript 'br:servicehubregistry.azurecr.io/bicep/modules/private-deployment-script:v3' = {
  name: 'servicehubval-${resourcesName}-csharpuserrp-cert-scriptDeploy'
  scope: resourceGroup(subscriptionId, resourceGroupName)
  params: {
    scriptName: 'servicehubval-${resourcesName}-cert-script'
    location: location
    resourceGroupName: resourceGroupName
    subscriptionId: subscriptionId
    managedIdentityName: scriptIdentity.outputs.name
    resourcesName: resourcesName
    vnetName: vnet.name
    scriptSubnetName: scriptSubnet.name
    privateEndpointSubnetName: peSubnet.name
    storageFilePrivateDnsZoneName: storageFilePrivateDnsZone.name
    scriptArguments: '-VaultName ${keyvault.outputs.name} -CertName ${certName} -DomainName ${san} -IssuerName ${issuer} -CN ${sub}'
    scriptContent: '''
      param(
        [string] $VaultName,
        [string] $CertName,
        [string] $DomainName,
        [string] $IssuerName,
        [string] $CN
      )

      Write-Host "Script running in private networking mode with private endpoints"
      Write-Host "Connecting to Key Vault: $VaultName via private endpoint..."
      
      try
      {
        Write-Host "Checking if Certificate $CertName exists in $VaultName"
        $cert = Get-AzKeyVaultCertificate -VaultName $VaultName -Name $CertName

        $certCreated = $false

        $subjectName = "$CN"
        $san = "$DomainName"
        $policy = New-AzKeyVaultCertificatePolicy -SubjectName $subjectName -IssuerName $issuerName -DnsNames $San -ValidityInMonths 3 -RenewAtPercentageLifetime 90

        if ($cert -eq $null)
        {
          Write-Host "Certificate does not exist. Creating"
          Set-AzKeyVaultCertificateIssuer -VaultName $VaultName -IssuerProvider $IssuerName -Name $IssuerName
          Add-AzKeyVaultCertificate -VaultName $VaultName -Name $CertName -CertificatePolicy $policy
        }
        else
        {
          Write-Host "Certificate already exists - checking creation status"

          $certOp = Get-AzKeyVaultCertificateOperation -VaultName $VaultName -Name $CertName
          if ($certOp.status -eq 'completed')
          {
            Write-Host "  certificate created";
            $certCreated = $true
          }
          else
          {
            Write-Host "  certificate operation failed with status $($certOp.status) and error $($certOp.errormessage). Applying updated policy and retrying..."

            Add-AzKeyVaultCertificate -VaultName $VaultName -Name $CertName -CertificatePolicy $policy
          }
        }

        while (-not $certCreated)
        {
          $certOp = Get-AzKeyVaultCertificateOperation -VaultName $VaultName -Name $CertName
          if ($certOp.status -eq 'completed')
          {
              Write-Host "  certificate created";
              $certCreated = $true
          }
          elseif ($certOp.status -eq 'inProgress')
          {
            Write-Host "  waiting for certificate to be created..."
            Start-Sleep -Seconds 2
          }
          else
          {
            # Just throw, and we can rerun and try again
            throw "  certificate operation failed with status $($certOp.status) and error $($certOp.errormessage) ..."
          }
        }
      }
      catch {
        Write-Error "Certificate creation failed: $($_.Exception.Message)"
        # Re-throw the error to fail the deployment script
        throw
      }
    '''
  }
}

// Microsoft Corp tenant ID
var corpTenantId = '72f988bf-86f1-41af-91ab-2d7cd011db47'
// Get the tenant ID from the deployer function
var tenantId = deployer().tenantId
// Determine if we should use Microsoft tenant tagging
var isMicrosoftTenant = tenantId == corpTenantId
// Public IP address for the Userrp service ingress.
// This public IP is used for the ingress controller to expose the Userrp service to the internet
// The DNS label prefix is used to create a unique DNS name for the public IP address.
module ingressPublicIp 'br/public:avm/res/network/public-ip-address:0.8.0' = {
  name: 'servicehubval-${resourcesName}-csharpuserrp-ingressIpDeploy'
  scope: resourceGroup(subscriptionId, resourceGroupName)
  params: {
    name: 'servicehubval-${resourcesName}-csharpuserrp-ingress-ip'
    location: location
    publicIPAllocationMethod: 'Static'
    skuName: 'Standard'
    skuTier: 'Regional'
    // Conditional tagging based on tenant: NonProd for Microsoft Corp, AKSServiceHubValidation for AME
    // Ensure you have provisioned service tag for your service by following the steps here: 
    // https://eng.ms/docs/cloud-ai-platform/azure-core/azure-networking/sdn-dbansal/sdn-buildout-and-deployments/sdn-fundamentals/service-tag-onboarding/onboarding-process
    // Additionally, ensure your target subscription(s) has the "AllowBringYourOwnPublicIpAddress" AFEC feature flag enabled
    // Steps can be found here: https://eng.ms/docs/cloud-ai-platform/azure-core/azure-networking/sdn-dbansal/sdn-buildout-and-deployments/sdn-fundamentals/service-tag-onboarding/get-access-and-create-tagged-ips/enable-feature-flag
    ipTags: [
      {
        ipTagType: 'FirstPartyUsage'
        tag: isMicrosoftTenant ? '/NonProd' : '/AKSServiceHubValidation'
      }
    ]
    dnsSettings: {
      domainNameLabel: dnsLabelPrefix
    }
    zones: []
  }
}

// Below resources are not required for Userrp service setup, but are needed to access other Azure resources for real code business logic.
// This resource is shared and defined in resources/Main.SharedResources.Template.bicep in shared-resources directory; we only reference it here. Do not remove `existing` syntax.
resource rg 'Microsoft.Resources/resourceGroups@2021-04-01' existing = {
  name: resourceGroupName
  scope: subscription(subscriptionId)
}

var serviceAccountNamespace = 'servicehubval-csharpuserrp-server'
var serviceAccountName = 'servicehubval-csharpuserrp-server'
module managedIdentity 'br/public:avm/res/managed-identity/user-assigned-identity:0.2.1' = {
  name: 'servicehubval-csharpuserrp-managed-identityDeploy'
  scope: resourceGroup(subscriptionId, resourceGroupName)
  params: {
    name: 'servicehubval-${resourcesName}-${location}-csharpuserrp-managedIdentity'
    location: rg.location
    federatedIdentityCredentials: [
      {
        name: 'servicehubval-csharpuserrp-fedIdentity'
        issuer: aks.properties.oidcIssuerProfile.issuerURL
        subject: 'system:serviceaccount:${serviceAccountNamespace}:${serviceAccountName}'
        audiences: ['api://AzureADTokenExchange']
      }
    ]
  }
}

// TODO: Migrate to use bicep module registry. Current bicep registry module is management group scoped but we use subscription scoped.
module azureSdkRoleAssignment 'br:servicehubregistry.azurecr.io/bicep/modules/subscription-role-assignment:v6' = {
  name: 'servicehubval-csharpuserrpazuresdkra${location}Deploy'
  scope: subscription(subscriptionId)
  params: {
    principalId: managedIdentity.outputs.principalId
    description: 'servicehubval-csharpuserrp-${resourcesName}-contributor-azuresdk-role-assignment'
    roleDefinitionIdOrName: 'Contributor'
    principalType: 'ServicePrincipal'
    subscriptionId: subscriptionId
  }
}

output certCreationScriptLogs string[] = certCreationScript.outputs.mainScriptLogs
output keyVaultName string = keyvault.outputs.name
output tenantId string = subscription().tenantId
output akvSecretsProviderClientId string = aks.properties.addonProfiles.azureKeyvaultSecretsProvider.identity.clientId
output certName string = certName

@sys.description('Client Id of the managed identity.')
output managedIdentityClientId string = managedIdentity.outputs.clientId
