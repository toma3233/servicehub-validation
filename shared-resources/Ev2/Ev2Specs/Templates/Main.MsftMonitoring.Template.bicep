// Geneva Azure Resource for setup for Geneva agent
// This file provisions the initial resources for the Geneva setup.
// Categories:
// - SUPPLEMENTARY: Resources not used directly in the Geneva pipeline but required for setup.
// - PRE-REQ: Resources or features that are required for the Geneva setup but not used directly in the pipeline.
// - REQ: Resources or features that are required for the Geneva setup and used directly in the pipeline.
@sys.description('The name for the resources.')
param resourcesName string

@sys.description('The subscription the resources are deployed to.')
param subscriptionId string

@sys.description('The location of the resource group the resources are deployed to.')
param location string

@sys.description('The short name of the region the resources are deployed to.')
param regionShortName string

@sys.description('The name of the resource group the resources are deployed to.')
param resourceGroupName string

@sys.description('Subject alternative name for the certificate. This should satisfy the registered wildcard domain that is registered in OneCert, i.e. test.mygreeterv3.servicehub.sre.azure-prod.net')
param san string

@sys.description('Registered wildcard test domain you regstered in OneCert, i.e. *.mygreeterv3.servicehub.sre.azure-prod.net.')
param cn string

targetScope = 'subscription'


// SUPPLEMENTARY: Managed identity to create certificate for geneva agent authentication into Geneva Account
module scriptIdentity 'br/public:avm/res/managed-identity/user-assigned-identity:0.4.0' = {
  name: 'servicehubval-${resourcesName}-geneva-setup-script-identityDeploy'
  scope: resourceGroup(subscriptionId, resourceGroupName)
  params: {
    name: 'servicehubval-${resourcesName}-geneva-setup-script-identity'
    // Non-required parameters
    location: location
  }
}


// REQ: AKS cluster to deploy the Geneva agent
// TODO (Christine): Migrate this to pass in a cluster name such that it's consistent and not hardcoded.
resource aks 'Microsoft.ContainerService/managedClusters@2024-09-02-preview' existing = {
  name: 'servicehubval-${resourcesName}-cluster'
  scope: resourceGroup(subscriptionId, resourceGroupName)
}

// REQ: Key Vault to store the certificate for the Geneva agent authentication into Geneva Account
// The max length of the name is 24 characters. The name must be globally unique.
var servicePrefix = substring('servicehubval', 0, min(8, length('servicehubval')))
var resourceNamePrefix = substring(resourcesName, 0, min(8, length(resourcesName)))
var regionPrefix = substring(regionShortName, 0, min(8, length(regionShortName)))
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
  name: 'servicehubval-${resourcesName}-keyvaultDeploy'
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
        roleDefinitionIdOrName: 'Key Vault Certificate User'
        principalType: 'ServicePrincipal'
      }
      {
        principalId: scriptIdentity.outputs.principalId
        roleDefinitionIdOrName: 'Key Vault Certificates Officer' // Perform any action on the certificates of a key vault, excluding reading the secret and key portions, and managing permissions. Only works for key vaults that use the 'Azure role-based access control' permission model. TODO (Christine): Check if this is the correct role to use, or if it's need at all.
        principalType: 'ServicePrincipal'
      }
      {
        principalId: scriptIdentity.outputs.principalId
        // This allows the script to create the certificate
        roleDefinitionIdOrName: 'Key Vault Contributor' // Allow creation and management of key vaults, but not the keys, secrets, or certificates within them. This role is intended for use by applications that need to create and manage key vaults.
        principalType: 'ServicePrincipal'
      }
    ]
  }
}

// REQ: Certificate creation using the private deployment script module
// This replaces manual storage account creation, deployment script, and cleanup script
var issuer = 'OneCertV2-PrivateCA'
var sub = 'CN=${cn}'
var certName='servicehubval${resourcesName}genevacert'

module certCreationScript 'br:servicehubregistry.azurecr.io/bicep/modules/private-deployment-script:v2' = {
  name: 'servicehubval-${resourcesName}-cert-script'
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

output certCreationScriptLogs string[] = certCreationScript.outputs.mainScriptLogs
output cleanupScriptLogs string[] = certCreationScript.outputs.cleanupScriptLogs
output keyVaultName string = keyvault.outputs.name
