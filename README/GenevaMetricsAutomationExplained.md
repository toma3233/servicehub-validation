# Table of Contents

- [Table of Contents](#table-of-contents)
- [Geneva Metrics Automation for 1P](#geneva-metrics-automation-for-1p)
  - [Resource/Module Table](#resourcemodule-table)
- [Managed Prometheus Metrics Flow to Azure Monitor Workspace to Grafana](#managed-prometheus-metrics-flow-to-azure-monitor-workspace-to-grafana)
  - [Detailed Flow Description](#detailed-flow-description)
    - [How the Data Collection Rule (DCR) Works with the Data Collection Endpoint (DCE)](#how-the-data-collection-rule-dcr-works-with-the-data-collection-endpoint-dce)
  - [Summary](#summary)
  - [Notes](#notes)

# Geneva Metrics Automation for 1P

This document is for 1P teams.

Geneva Metrics is now encouraging alignment with the 3P offering. Most steps are well-documented by 3P Microsoft documentation.

The steps below are taken from this [wiki](https://eng.ms/docs/products/geneva/metrics/prometheus/promgetstarted). Read through this if needed, but we will cover each step in this document.

1. Create your 1P Azure Monitor workspace and link it to your MDM account. Scenario 3 is followed.
   1. In Scenario 3, Azure Monitor workspace creates the Geneva Metrics (MDM) account in the default shared MDM stamp of that region.
   2. We choose this scenario because "Managed metrics" account is created by the Azure monitor workspace, and we do not need to create our own stamp.
   3. The creation of this internal azure monitor workspace is only supported in certain regions.
2. Deploy the addon to your AKS cluster for metrics collection
3. Configure metrics collection
4. Set up Azure Managed Grafana
5. Configure Prometheus data source
6. Set up IcM integration for your alert rules using IcM connector [TODO]
7. Set up alerts and recording rules [TODO]

We automate these steps and separate them out logically.

## Resource/Module Table

Each module automates a step in setting up Geneva metrics monitoring, from identity and permissions to workspace and dashboard provisioning.

| Resource/Module Name                              | Purpose                                                                                                                      | Step Covered                                                                                  | Needed by Step(s)         | 3P Docs Covered? | Documentation | Scope   |
|---------------------------------------------------|------------------------------------------------------------------------------------------------------------------------------|-----------------------------------------------------------------------------------------------|-----------------------------|------------------|---------------|---------|
| `rg` (resource group)                             | References the existing resource group where all resources will be deployed.                                                 |                                                                                      | All steps                   | Yes              | -             | Global  |
| `scriptIdentity`                                  | Creates a managed identity used to run scripts for enabling required Azure Monitor features.                                 |                                                       | Step 1 (Create Azure Monitor workspace)                      | Yes              | -             | Global  |
| `registrationScriptRoleAssignment`                | Assigns Contributor role to the managed identity at the subscription level so it can register features.                      |                                                        | Step 1 (Create Azure Monitor workspace)                       | Yes              | -             | Global  |
| `registrationScriptDeployment`                    | Runs an Azure CLI script to register the `EnableFirstPartyFeatures` feature for Microsoft.Monitor, required for internal Azure Monitor workspace creation. |                                                        | Step 1 (Create Azure Monitor workspace)             | No               | [Microsoft Docs - Enable First Party Features](https://eng.ms/docs/products/geneva/metrics/prometheus/mac#prerequisites) | Global  |
| `azureMonitorWorkspaces`                | Deploys all Azure Monitor workspaces for metrics ingestion and storage.                                               | Step 1 (Create Azure Monitor workspace)                | Step 3 (Configure metrics collection), Step 5  (Configure Prometheus data source)            | No               | [Microsoft Docs - Internal Azure Monitor Workspace](https://eng.ms/docs/products/geneva/metrics/prometheus/mac#scenario-3) | Global  |
| `grafanaWorkspace`                                | Creates an Azure Managed Grafana workspace and assigns admin/viewer roles to specified security groups. Configures azure monitor workspace as a data source.                      | Step 4 (Set up Azure Managed Grafana), Step 5 (Configure Prometheus data source)              |                       | Yes              | [Azure Managed Grafana](https://learn.microsoft.com/en-us/azure/managed-grafana/overview)             | Global  |
| `grafanaAMWRoleAssignment`                        | Assigns the Grafana workspace managed identity access to the Azure Monitor workspace/resource group as required.             | Step 4 (Set up Azure Managed Grafana), Step 5 (Configure Prometheus data source)              |                      | Yes              |             | Global  |
| `dataCollectionEndpoint`                          | Provisions a Data Collection Endpoint (DCE) for secure ingestion of Prometheus metrics.                                     | Step 2 (Deploy addon to AKS cluster for metrics collection)                                                         |       Step 3 (Configure metrics collection)                | Yes              | [Microsoft Docs - Enable Azure Monitor Profile Metrics Add On](https://learn.microsoft.com/en-us/azure/azure-monitor/containers/kubernetes-monitoring-enable?tabs=arm#enable-prometheus-and-grafana) | Shared  |
| `dataCollectionRule`                              | Provisions a Data Collection Rule (DCR) to define processing and routing of metrics.                                        | Step 2 (Deploy addon)                                                         | Step 2 (Deploy addon to AKS cluster for metrics collection)                      | Yes              | [Microsoft Docs - Enable Azure Monitor Profile Metrics Add On](https://learn.microsoft.com/en-us/azure/azure-monitor/containers/kubernetes-monitoring-enable?tabs=arm#enable-prometheus-and-grafana) | Shared  |
| `aksAzureMonitorDataCollectionRuleAssociation`     | Associates the DCR with the AKS cluster to enforce metrics collection and routing.                                           | Step 2 (Deploy addon to AKS cluster for metrics collection)                                                         |       Step 3 (Configure metrics collection)               | Yes              | [Microsoft Docs - Enable Azure Monitor Profile Metrics Add On](https://learn.microsoft.com/en-us/azure/azure-monitor/containers/kubernetes-monitoring-enable?tabs=arm#enable-prometheus-and-grafana) | Shared  |
| `enableAzureMonitorProfileMetrics`              | Enables the Azure Monitor Profile Metrics feature in the AKS cluster to allow managed Prometheus metrics collection.         | Step 2 (Deploy addon to AKS cluster for metrics collection)                                   | Step 3 (Configure metrics collection)                      | Yes              | [Microsoft Docs - Enable Azure Monitor Profile Metrics Add On](https://learn.microsoft.com/en-us/azure/azure-monitor/containers/kubernetes-monitoring-enable?tabs=arm#enable-prometheus-and-grafana) | Shared  |

**Permission Model and Required Steps:**

1. **Enable AMW Feature:** Enable `EnableFirstPartyFeatures` via script.
2. **Create AMW:** Deploy the Azure Monitor Workspace.
3. **Apply AMW Policy:** Apply an AMW policy so that principals from the CorpTenant can access this AMW, even if it is in a different *ME tenant.
4. **Create Grafana Workspace:** Deploy the Grafana workspace.
5. **Assign Grafana Roles:** Create role assignments for principals from CorpTenant to access the Grafana workspace.
6. **Identity Flow:** Grafana passes the user's identity to AMW, which relies on the AMW policy to check if the user can access metrics.
7. **Register Prometheus Endpoint:** Manually register the Prometheus endpoint exposed by AMW in Grafana.

**End-to-end flow:**
User (Corp) → Grafana (Corp tenant) → Prometheus endpoint exposed by AMW (Corp Tenant and *ME tenants)

Note that we did not cover step 3, since this covers [Customize scraping of Prometheus metrics in Azure Monitor managed service for Prometheus](https://learn.microsoft.com/en-us/azure/azure-monitor/containers/prometheus-metrics-scrape-configuration?tabs=CRDConfig%2CCRDScrapeConfig%2CConfigFileScrapeConfigBasicAuth%2CConfigFileScrapeConfigTLSAuth). Feel free to follow this and customize Prometheus metrics as needed.

# Managed Prometheus Metrics Flow to Azure Monitor Workspace to Grafana

This section provides a detailed, step-by-step explanation of how Prometheus metrics are ingested from an AKS cluster into an Azure Monitor Workspace using managed Prometheus and the resources defined in your Bicep template.

## Detailed Flow Description

1. **Workloads in AKS Expose Prometheus Metrics**
   - Applications and system components running in your AKS (Azure Kubernetes Service) cluster expose metrics in Prometheus format via HTTP endpoints.
   - These endpoints are typically annotated or labeled so that Prometheus scrapers can discover and collect metrics from them.

2. **Managed Prometheus Integration on AKS**
   - The AKS cluster is configured with `enableAzureMonitorProfileMetrics: true`.
   - This setting enables Azure's managed Prometheus integration, which automatically deploys and manages the necessary agents on the cluster.
   - These agents are responsible for discovering and scraping Prometheus metrics endpoints from the workloads running in the cluster.

3. **Data Collection Endpoint (DCE) Deployment**
   - The Bicep template provisions a Data Collection Endpoint (DCE) resource in Azure using the `dataCollectionEndpoint` module.
   - The DCE acts as a secure and scalable ingestion point for metrics data coming from the AKS cluster.
   - The managed Prometheus agents on the AKS cluster are configured to send the scraped Prometheus metrics to this DCE.

4. **Data Collection Rule (DCR) Deployment**
   - The Bicep template provisions a Data Collection Rule (DCR) resource using the `dataCollectionRule` module.
   - The DCR defines how incoming metrics data should be processed, filtered, and routed.
   - It specifies the data sources (Prometheus metrics from the AKS cluster via the DCE), the data flows (how the data should be handled), and the destinations (where the data should be sent).
   - In this case, the DCR is configured to forward Prometheus metrics to the Azure Monitor Workspace, as seen in the `destinations` and `dataFlows` properties.

5. **Data Collection Rule Association (DCRA)**
   - The Bicep template provisions a Data Collection Rule Association (DCRA) using the `aksAzureMonitorDataCollectionRuleAssociation` module.
   - The DCRA resource associates the DCR with the AKS cluster.
   - This association ensures that the DCR is applied to the metrics data coming from the specific AKS cluster, enabling the defined processing and routing rules.
   - Without this association, the DCR would not be enforced for the cluster's metrics data.

6. **Metrics Ingestion into Azure Monitor Workspace and Visualization in Grafana**
   - After passing through the DCE and being processed by the DCR (as enforced by the DCRA), the Prometheus metrics are ingested into the Azure Monitor Workspace.
   - The Azure Monitor Workspace provides storage, querying, visualization, and alerting capabilities for the ingested metrics.
   - Azure Managed Grafana is configured to use the Azure Monitor Workspace as a data source.
   - This allows you to build dashboards and visualize Prometheus metrics directly in Grafana, leveraging Azure Monitor's integration.
   - You can use Azure Monitor features such as dashboards, alerts, and workbooks, as well as Grafana's advanced visualization capabilities, to analyze and act on the collected Prometheus metrics.

### How the Data Collection Rule (DCR) Works with the Data Collection Endpoint (DCE)

The Data Collection Endpoint (DCE) acts as a secure network entry point in Azure, receiving Prometheus metrics from the managed Prometheus agent in your AKS cluster. The Data Collection Rule (DCR) defines which data to collect from the DCE, how to process it, and where to send it (such as the Azure Monitor Workspace). The DCR and DCE together ensure only the required telemetry is securely routed from your AKS cluster to Azure Monitor.

---

## Summary
- Workloads in AKS expose Prometheus metrics.
- Managed Prometheus integration scrapes these metrics.
- Scraped metrics are sent to the Data Collection Endpoint (DCE).
- The Data Collection Rule (DCR) defines how metrics are processed and routed.
- The Data Collection Rule Association (DCRA) links the DCR to the AKS cluster.
- Metrics are ingested into the Azure Monitor Workspace for monitoring and analysis.

## Notes

> **Note 1:** No manual deployment of Prometheus collectors is required. The managed integration and the DCE/DCR/DCRA resources handle the entire ingestion pipeline securely and efficiently.

> **Note 2:** The internal Azure Monitor workspace is provisioned according to Scenario 3 (Managed Metrics account on a shared default stamp). Only certain Azure regions are supported for this managed metrics account. Shared default 1P stamps are available only in regions where the "GenevaMetricsFirstPartyStamp" setting is not null in the deployment settings file. Attempting to create an Azure Monitor workspace in unsupported regions will result in the error: "Metrics account creation is not yet supported". For more details, see the [Scenario 3 documentation](https://eng.ms/docs/products/geneva/metrics/prometheus/mac#scenario-3) and the [deployment settings file](https://msazure.visualstudio.com/One/_git/EngSys-MDA-AMCS?path=/Deployment/oaAppSettings.json&version=GBmaster&_a=contents).
