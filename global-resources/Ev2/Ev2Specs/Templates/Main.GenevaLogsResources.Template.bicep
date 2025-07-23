// Geneva Azure Resource for e2e setup
// This file provisions the initial resources for the Geneva setup. It creates a Key Vault, a Geneva account, and a Kusto cluster with a database and table. It also creates a data connection to the Geneva account.
// Categories:
// - SUPPLEMENTARY: Resources not used directly in the Geneva pipeline but required for setup.
// - PRE-REQ: Resources or features that are required for the Geneva setup but not used directly in the pipeline.
// - REQ: Resources or features that are required for the Geneva setup and used directly in the pipeline.

@sys.description('The name for the resources.')
param resourcesName string

@sys.description('The subscription the resources are deployed to.')
param subscriptionId string

@sys.description('The name of the resource group the resources are deployed to.')
param resourceGroupName string

// Although this parameter isn't used, it is required for subscription-level deployments.
@sys.description('The location of the resources.')
param location string

@sys.description('Subject alternative name for the certificate. This should satisfy the registered wildcard domain that is registered in OneCert, i.e. test.mygreeterv3.servicehubval.sre.azure-prod.net')
param san string

// TODO: We assume that the role definition exists. We should check if it exists and create it if it doesn't. Source: https://eng.ms/docs/products/geneva/logs/howtoguides/manageaccount/subscriptionpermissions
@sys.description('The Geneva role definition ID. This is the role definition ID associated with the GenevaWarmPathResourceContributor role.')
param genevaRoleDefinitionId string

// This value is different for each environment. The details on how to obtain this value are specified here: https://eng.ms/docs/products/geneva/logs/howtoguides/manageaccount/subscriptionpermissions. It also details how to create the service principal if it doesn't exist.
// The geneva service principal id is the object id of the service principal that is created by Geneva.
// The value is provided via the ScopeBinding.json file. The value in ScopeBinding.json is populated by the Configuration.json for each environment.
@sys.description('The Geneva service principal ID. This is the service principal that is used to access the Geneva account that automatically provisions the resources for the Geneva account.')
param genevaServicePrincipalId string

@sys.description('The name of the Geneva environment. This is the name of the Geneva environment that is used to create the Geneva account.')
param genevaEnvironment string

@sys.description('The admin security group ID. This is thesecurity  group that will be used to access the production Kusto cluster from their admin accounts. This security group should also be linked to the Ev2 release pipeline.')
param prodAdminSecurityGroupId string

@sys.description('The corp security group ID. This is the security group that will be used to access the production Kusto cluster from their coporate accounts.')
param corpAdminSecurityGroupId string

@sys.description('JSON string array of Ev2 IP addresses for network access rules.')
param ev2IpAddresses array

targetScope = 'subscription'

// Convert EV2 IP addresses to CIDR ranges for Network Security Perimeter
var ev2IpAddressRanges = [for ip in ev2IpAddresses: '${ip}/32']

module rg 'br/public:avm/res/resources/resource-group:0.2.3' = {
  name: '${resourceGroupName}LogsDeploy'
  scope: subscription(subscriptionId)
  params: {
    name: resourceGroupName
    location: location
  }
}

module networkSecurityPerimeter 'br/public:avm/res/network/network-security-perimeter:0.1.0' = {
  name: 'servicehubval-${resourcesName}-nspDeploy'
  scope: resourceGroup(subscriptionId, resourceGroupName)
  params: {
    // Required parameters
    name: 'servicehubval-${resourcesName}-nsp'
    // Non-required parameters
    location: rg.outputs.location
    profiles: [
      {
        accessRules: [
          {
            // Required since deployment script creates an ACI in azure cloud that is used to access key vault
            // Used traffic analysis tool to determine this is the best service tag to use for access rule
            // https://dataexplorer.azure.com/dashboards/0799d844-3039-4736-9d1a-9daab2aff826
            serviceTags: [
              'AzureCloud.${location}'
            ]
            direction: 'Inbound'
            name: 'rule-inbound-azcloud'
          }
          // EV2 best practice for identifying IPs used by ev2 extensions is by referencing the central configuration
          // The list of IP addresses is provided by the EV2 team, users don't need to hardcode it
          // https://ev2docs.azure.net/features/rollout-infra/overview.html#best-practice-always-reference-ev2-ip-addresses-from-central-configuration-instead-of-hardcoding
          // TODO (tomabraham): Monitor best practices doc and update to using service tag once they migrate to that
          {
            addressPrefixes: ev2IpAddressRanges
            direction: 'Inbound'
            name: 'rule-inbound-ev2'
          }
        ]
        name: 'profile-01'
      }
    ]
    resourceAssociations: [
      {
        accessMode: 'Enforced'
        privateLinkResource: keyvault.outputs.resourceId
        profile: 'profile-01'
      }
    ]
  }
}

// SUPPLEMENTARY: Managed Identity for Geneva setup
// This identity is used to run the scripts that create the Key Vault certificate and to register the feature AllowGenevaObtainer for Kusto.
module scriptIdentity 'br/public:avm/res/managed-identity/user-assigned-identity:0.4.0' = {
  name: 'servicehubval-${resourcesName}-geneva-setup-script-identityDeploy'
  scope: resourceGroup(subscriptionId, resourceGroupName)
  params: {
    name: 'servicehubval-${resourcesName}-geneva-setup-script-identity'
    // Non-required parameters, but required for dependency with resource group
    location: rg.outputs.location // This ensures that the identity is created after the resource group is created.
  }
}

// SUPPLEMENTARY: Create Key Vault for Geneva Account Creation in Ev2
// Geneva Link: https://eng.ms/docs/products/geneva/getting_started/environments/servicefabric/keyvault
// Ev2 Link: https://ev2docs.azure.net/features/service-artifacts/actions/http-extensions/shared-extensions/Microsoft.Geneva.Logs.html
// This Key Vault is used to store the Geneva account certificate that is used to authenticate the creation of the Geneva account. The certificate is created using OneCert and is stored in the Key Vault.
// This is NOT the key vault where we store the certificate that the geneva agent uses to authenticate to the Geneva account. This Key Vault is used to store the certificate that is used to create the Geneva account.
// Note that you will have to registered the domain in OneCert before you can create the certificate. The domain is the SAN of the certificate. The CN of the certificate is also set up in OneCert.
// The cert in the key vault is used in GenevaAccount.Rollout.Parameters.json.


// The max length of the name is 24 characters. The name must be globally unique.
var servicePrefix = substring('servicehubval', 0, min(12, length('servicehubval')))
var resourceNamePrefix = substring(resourcesName, 0, min(12, length(resourcesName)))
var truncatedKeyVaultName = '${servicePrefix}${resourceNamePrefix}'

// An alternative to securing the key vault via NSP is deleting the key vault after the Geneva account is created
// This will avoid the NSP setup
module keyvault 'br/public:avm/res/key-vault/vault:0.9.0' = {
  name: 'servicehubval-${resourcesName}-keyvaultDeploy'
  scope: resourceGroup(subscriptionId, resourceGroupName)
  params: {
    // Required parameters
    name: truncatedKeyVaultName
    // Non-required parameters
    enablePurgeProtection: false
    location: rg.outputs.location
    roleAssignments: [
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

// SUPPLEMENTARY: Create Key Vault Certificate for Geneva Account Creation in Ev2
// Geneva Link: https://eng.ms/docs/products/geneva/getting_started/environments/servicefabric/keyvault
// Ev2 Link: https://ev2docs.azure.net/features/service-artifacts/actions/http-extensions/shared-extensions/Microsoft.Geneva.Logs.html
// This certificate is used by the security group for Ev2 release to authenticate the creation of the Geneva account. The certificate is created using OneCert and is stored in the Key Vault. The certificate is created using a script that is run in the Key Vault.
var issuer = 'OneCertV2-PrivateCA'
var sub = 'CN=${san}'
var certName = 'servicehubval${resourcesName}genevaaccountcert'

// TODO: separate this out into a separate module. This is a bit of a hack to get the certificate creation script to work.
module certCreationScriptDeployment 'br/public:avm/res/resources/deployment-script:0.5.1' = {
  name: 'servicehubval-${resourcesName}-cert-scriptDeploy'
  scope: resourceGroup(subscriptionId, resourceGroupName)
  params: {
    // Required parameters
    kind: 'AzurePowerShell'
    name: 'servicehubval-${resourcesName}-cert-script'
    // Non-required parameters
    azPowerShellVersion: '12.3'
    location: rg.outputs.location
    managedIdentities: {
      userAssignedResourceIds: [
        scriptIdentity.outputs.resourceId
      ]
    }
    arguments: '-VaultName ${keyvault.outputs.name} -CertName ${certName} -DomainName ${san} -IssuerName ${issuer} -CN ${sub}'
    scriptContent: '''
      param(
        [string] $VaultName,
        [string] $CertName,
        [string] $DomainName,
        [string] $IssuerName,
        [string] $CN
      )

      $ipaddress = (Invoke-WebRequest -uri "http://ifconfig.me/ip").Content

      Write-Host "Adding current IP Address $ipaddress to Firewall Rules"
      Add-AzKeyVaultNetworkRule -VaultName $VaultName -IpAddressRange $ipaddress

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
      finally
      {
        $cidrRange = $ipaddress + "/32"
        Write-Host "Removing current IP Address $cidrRange from Firewall Rules"
        Remove-AzKeyVaultNetworkRule -VaultName $VaultName -IpAddressRange $cidrRange
      }
    '''
  }
}

output certCreationScriptDeploymentLogs string[] = certCreationScriptDeployment.outputs.deploymentScriptLogs

// SUPPLEMENTARY: Give contributor access to the script identity on the subscription level.
// To register the feature AllowGenevaObtainer for Kusto, we need to give the script identity contributor access on the subscription level.
module registrationScriptRoleAssignment 'br:servicehubregistry.azurecr.io/bicep/modules/subscription-role-assignment:v6' = {
  name: 'servicehubval-${resourcesName}-geneva-script-role-assignmentDeploy'
  scope: subscription(subscriptionId)
  params: {
    principalId: scriptIdentity.outputs.principalId
    description: 'servicehubval-${resourcesName}-geneva-script-role-assignment'
    roleDefinitionIdOrName: 'Contributor'
    principalType: 'ServicePrincipal'
    subscriptionId: subscriptionId
  }
}

// PRE-REQ: Register feature AllowGenevaObtainer for Kusto
// Link: https://kusto.azurewebsites.net/docs/kusto/ops/manage-geneva-dataconnections.html#prerequisites
// This is required in order to create the data connection to Geneva. The script is run with the script identity that was created above.
module registrationScriptDeployment 'br/public:avm/res/resources/deployment-script:0.5.1' = {
  name: 'servicehubval-${resourcesName}-allow-geneva-obtainer-scriptDeploy'
  scope: resourceGroup(subscriptionId, resourceGroupName)
  params: {
    // Required parameters
    kind: 'AzureCLI'
    name: 'servicehubval-${resourcesName}-allow-geneva-obtainer-script'
    // Non-required parameters
    azCliVersion: '2.52.0'
    location: rg.outputs.location
    managedIdentities: {
      userAssignedResourceIds: [
        scriptIdentity.outputs.resourceId // This is the identity that will be used to run the script. It needs to be registered as a contributor on the subscription level.
      ]
    }
    scriptContent: '''
      #!/bin/bash
      set -e
      echo "Registering feature AllowGenevaObtainer"
      az feature register --namespace Microsoft.Kusto --name AllowGenevaObtainer
    '''
  }
  dependsOn: [
    registrationScriptRoleAssignment // This is required to give the script identity contributor access on the subscription level.
  ]
}

output registrationScriptDeploymentLogs string[] = registrationScriptDeployment.outputs.deploymentScriptLogs

// REQ: Create Kusto Cluster and Database
// Max length of the name is 22 characters. The name must be globally unique.
// TODO (Christine): This is a hack to avoid length issues with Kusto cluster name, but it will cause issues with dashboard generation since the naming will no longer be consistent if we truncate the name.
var originalKustoClusterName = 'servicehubval${resourcesName}'
var kustoServicePrefix = substring('servicehubval', 0, min(11, length('servicehubval')))
var kustoResourceNamePrefix = substring(resourcesName, 0, min(11, length(resourcesName)))
var truncatedKustoClusterName = '${kustoServicePrefix}${kustoResourceNamePrefix}'
var finalKustoClusterName = length(originalKustoClusterName) <= 22 ? originalKustoClusterName : truncatedKustoClusterName

var databaseName = 'servicehubval${resourcesName}db'
module kustoCluster 'br/public:avm/res/kusto/cluster:0.5.0' = {
  name: 'servicehubval-${resourcesName}-kusto-clusterDeploy'
  scope: resourceGroup(subscriptionId, resourceGroupName)
  params: {
    // Required parameters
    name: finalKustoClusterName
    sku: 'Standard_E2ads_v5'
    enableDiskEncryption: true
    location: rg.outputs.location
    tags: {
      'NRMS.KustoPlatform.Classification.1P': 'Corp' // TODO: pass this in as a parameter. options are Prod/Corp/Default/Other.
      'opt-out-of-soft-delete': true // This is required to opt out of soft delete for the Kusto cluster. This is a temporary measure until we can figure out how to enable soft delete for the Kusto cluster.
    }
    principalAssignments: [
      {
        principalId: corpAdminSecurityGroupId
        role: 'AllDatabasesViewer' // allows accounts in the corp tenant security group to view all databases
        principalType: 'Group'
        tenantId: '72f988bf-86f1-41af-91ab-2d7cd011db47' // This is the corporate tenant ID. It is required for the role assignment to work.
      }
      {
        principalId: prodAdminSecurityGroupId
        role: 'AllDatabasesViewer' // allows accounts in the prod tenant security group to view all databases
        principalType: 'Group'
        tenantId: '33e01921-4d64-4f8c-a055-5bdaffd5e33d' // This is the AME production tenant ID. It is required for the role assignment to work. TODO (Christine): Consider how to add other tenants or tenants in other Cloud.
      }
    ]
    databases: [
      {
        kind: 'ReadWrite'
        name: databaseName
        readWriteProperties: {
          hotCachePeriod: 'P1D'
          softDeletePeriod: 'P7D'
        }
      }
    ]
  }
}

// REQ: Create table in Kusto Database
// TODO: In the future, need to account for the other tables that are created in the Kusto database. This is currently hardcoded to create the ApiRequestLog table. The table is created using a script that is run in the Kusto database. Extract this out in the future.
module kustoDatabaseTable './kusto_table.bicep' = {
  name: 'servicehubval-${resourcesName}-kusto-tableDeploy'
  scope: resourceGroup(subscriptionId, resourceGroupName)
  params: {
    // Required parameters
    continueOnErrors: false
    clusterName: kustoCluster.outputs.name
    databaseName: databaseName
    scriptName: 'createApiRequestLog'
    tableName: 'ApiRequestLog'
  }
}

// REQ: Create Geneva Data Connection to Kusto Cluster
// Link: https://kusto.azurewebsites.net/docs/kusto/ops/manage-geneva-dataconnections.html#prerequisites
// This is required in order to create the Geneva data connection for a Kusto cluster. The script is run with the script identity that was created above.
// Note that this is a legacy data connection. We are working on migrating to the new data connection model.
module dataConnection './first_party_data_connection.bicep' = {
  name: 'servicehubval-${resourcesName}-kusto-data-connectionDeploy'
  scope: resourceGroup(subscriptionId, resourceGroupName)
  params: {
    // Required parameters
    location: rg.outputs.location
    clusterName: kustoCluster.outputs.name
    dataConnectionName: '${resourcesName}data' // The name of the Geneva data connection. This is limited in characters. Refer to data_connection.bicep for more details.
    genevaEnvironment: genevaEnvironment
    mdsAccounts: [
      'servicehubvalev2logs'
    ]
    isScrubbed: true
  }
  dependsOn: [
    registrationScriptDeployment // TODO: figure out how to make this work without the dependency. The script needs to be run before the data connection is created.
  ]
}

// PRE-REQ: Give Geneva Access to Azure Subscription
// Documentation link: https://eng.ms/docs/products/geneva/logs/howtoguides/manageaccount/subscriptionpermissions (Specifies all subscription prerequisites for Geneva)
// This role assignment enables Geneva to provision necessary storage accounts and Event Hub namespaces in your Azure Subscription. This gives Geneva RW access to the subscription.
// The automated resource provisioning process is triggered when we create or update a Geneva account.
// The other step specified in the documentation is to register the necessary resource providers. This is done using the Ev2 extension and is not included in this Bicep file.
// Note that most role assignments are done via bicep, but there is a role assignment that is done via Ev2 during subscription provisioning. This grants Ev2 permissions to provision the resources in the subscription, so it has to be done via Ev2.
module genevaRoleAssignment 'br:servicehubregistry.azurecr.io/bicep/modules/subscription-role-assignment:v7' = {
  name: 'servicehubval-global-resources-genevara${location}Deploy'
  scope: subscription(subscriptionId)
  params: {
    principalId: genevaServicePrincipalId
    description: 'servicehubval-global-resources-${resourcesName}-contributor-geneva-role-assignment'
    roleDefinitionIdOrName: '/providers/Microsoft.Authorization/roleDefinitions/${genevaRoleDefinitionId}'
    principalType: 'ServicePrincipal'
    subscriptionId: subscriptionId
  }
}

output keyVaultName string = keyvault.outputs.name
output kustoClusterName string = kustoCluster.outputs.name
