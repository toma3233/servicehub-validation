# Deploy SLNM Commit Step
**Prerequisite**: [SLNM Create Rule TSG](https://eng.ms/docs/microsoft-security/digital-security-and-resilience/azure-security/security-health-analytics/network-isolation/tsgs/azurenetworkmanager/slnm_onboarding)

Please refer to the [SLNM Commit TSG](https://eng.ms/docs/microsoft-security/digital-security-and-resilience/azure-security/security-health-analytics/network-isolation/tsgs/azurenetworkmanager/slnm_commit_tsg) for further info. 

## What is SLNM?

SLNM is a program that all 1p service teams are required to onboard to. It enforces security rules onto all vnets and denies all traffic from the external internet by default. Enforces “zero-trust” security posturing from the network perspective
-	Even if a service doesn’t currently have any vnets, it is still required to “future proof” the service

As a service team, we must define network manager security rules for our service and then deploy this custom policy (applied at the environment and tenant scope e.g. prod/nonprod and msft/ame/pme) via SDP

These security rules are enforced by the an Azure Virtual Network Manager (AVNM) to all vnets provisioned by your service. When creating the network rulesets, you must specify service tree ID, security group object ID (cloud group if AME tenant), cloud, tenant, and environment. The SLNM team will create an AVNM and network ruleset config on your behalf in ***their*** subscription that can apply these rules to any vnet tied to your service once the commit step has been completed


### What is a security admin rule?

Security admin rules are global network security rules that enforce security policies defined in the rule collection on virtual networks. These rules can be used to Allow, Always Allow, or Deny traffic across virtual networks within your targeted network groups. These network groups can only consist of virtual networks within the scope of your virtual network manager instance. Security admin rules can't apply to virtual networks not managed by a virtual network manager.
-	This is useful to us since we will be provisioning a vnet in every RG in prod
-	These will all be managed under a single network group

## Steps

1. Prior to creating the network rulesets, ensure you have an AAD Security group in Corp tenant (Cloud Group in AME) and that you have provisioned an MSI that is a member of this group. Each tenant will require its own MSI and security group.
    - this MSI will be used to access AVNM and apply the configuration to each region during the commit step

2. Create the network rulesets in the [SLNM Portal](https://netiso-slnm-portal-prod-a6gmbyhvhffnfxhv.b02.azurefd.net/NetworkRulesets)
3. Wait some time for your S360 "Create" item to be updated to "Commit" with the necessary values provided by the SLNM team
4. Update config file with the appropriate values so that the SLNM Ev2 specs included in this template can use them to commit the rules 
    - In order for the SFI flag to be cleared, the commit step must be applied to **every** region that SLNM supports, regardless of whether your actual service has any resources these
    - Below are some changes made to the sample EV2 specs provided in the commit  TSG
        - This region list also includes **euap** regions which are non-public, and most subscriptions cannot access them by default. To access them, enable the AFEC flag in the subscription(s) used as the subscription key in the service model
        - This can be done via using [Microsoft.AzureCIS/plannedQuotas](https://eng.ms/docs/products/azcis-resource-provisioning/template-user-guide/resources/quota/planned-quota/plannedquota-in-arm) extension in your Bicep template
            - First you need to create a [blueprint](https://eng.ms/docs/cloud-ai-platform/azure-core/one-fleet-platform/microsoft-capacity-infrastructure-services-mjubran/azure-build-out-automation/azure-build-out-automation/catseye/introduction) for your service and assign to euap regions
            - Ensure that your service tree ID has an associated [CIS tenant](https://eng.ms/docs/cloud-ai-platform/azure-core/one-fleet-platform/microsoft-capacity-infrastructure-services-mjubran/azure-capacity-infrastructure-service/azcis-platform/getting-started/onboarding-to-cis/onboarding/configs/onboard-tenant)
            - Include this as part of the EV2 rollout
        - Used the default stage map from EV2 rather than creating our own
        - Used Bicep template to specify powershell commands rather than including them directly in an ARM template
        - Utilize templating in various files to maintain a single set of files, rather than having one for each env (i.e. Rolloutspec.Prod.json/Rolloutspec.Dev.json --> RolloutSpec.json)
        - Utilize templating to avoid hardcoding values directly in Ev2 specs
5. SLNM Build and Deploy pipelines will only have to be run once, since the network manager will apply rulesets to any new vnet in the regions that SLNM supports

