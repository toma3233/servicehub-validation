# Setting up Service Hub Generated Code

## Table of Contents

- [Prerequisites](#prerequisites)
  - [Operating System](#operating-system)
  - [Installations](#installations)
  - [Go get for private ADO repo](#go-get-for-private-ado-repo)
  - [Generated Directories](#generated-directories)
- [Terminology](#terminology)
- [Setting up](#creating-your-setup)
  - [Quick setup - Manual](#quick-setup-manually)
  - [Quick setup - Development pipeline](#quick-setup-using-development-pipeline)
- [Deleting resources](#deleting-your-resources)

## Warning

Do not run any commands with sudo.

## Prerequisites

### Operating system

Service Hub runs on Linux.

- If you are on Windows, follow the steps to install [WSL](https://learn.microsoft.com/en-us/windows/wsl/install) if you do not already have it

### Installations

For all of these installations, we recommend checking if they already exist on your WSL set up before installing.

- Follow the steps to install [Go](https://go.dev/doc/install). Make sure to add Go to your path.

- Follow the steps to install [Docker](https://docs.docker.com/engine/install/).  Also turn on [Docker Desktop WSL 2](https://docs.docker.com/desktop/wsl/) such that docker works with WSL.

- Install [Azure CLI](https://learn.microsoft.com/en-us/cli/azure/install-azure-cli).
  - Azure Subscription Requirements

    In order to create all the resources (shared and service specific), you must have role/permissions in your subscription to:

    1. Create, update and delete resources
    2. Assign roles

    Note that the `Contributor` role does not have the permissions to assign roles. The `Owner` role has all the permissions.

  - After installing, make sure to log in and set your subscription

    ```bash
    az login
    az account set --subscription $subscriptionId
    ```

- Check Bicep version to see if it was installed after Azure CLI is installed

  ```bash
  az bicep version
  az bicep upgrade
  ```

- You can choose to install [Kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/), or you can use our pre-built docker container that has everything installed once deploying your service.

- We assume that Bash is used as the default shell. If you are using Zsh, please make the necessary adjustments throughout the instructions accordingly.

### Go get for private ADO repo

If you have not already done the following steps.

Set up access for go build to be able to download modules from your organization for generation.

Creating a Personal Access Token (PAT) is essential to obtain the necessary access. There are two methods to achieve this. The first method involves setting up a manual PAT, which expires in a week and requires manual renewal each time it expires. The preferred method is to use a rotator that automatically fetches and injects a new token for you. Detailed instructions can be found [here](https://dev.azure.com/service-hub-flg/_git/service_hub?path=/developer-guide.md&version=GBmaster&_a=preview). If you choose to create the PAT manually, please follow the steps below.

- Create a personal authentication token under your organization to gain access to modules stored in your repository.
  - At your org's ADO(AzureDevOps e.g. [here](https://service-hub-flg.visualstudio.com/)) page, click the person with settings button next to your initials at the top right.
  - Click on Personal Access Tokens
- Create a new token for org account under "Code" scope with "Read" access. You can specify an expiration date. Make sure Organization is selected to be service-hub-flg, cross organization PAT won't work.
- Save token as environment variable READPAT (either by exporting in terminal or adding to bashrc profile).
- Also do the same for the GOPRIVATE variable below.
- Preferably add the lines to bashrc so that you do not have to set the variables every time you open a new terminal.

    ```bash
    export READPAT=[Your Personal Access Token of Azure Devops of your org]
    export GOPRIVATE="dev.azure.com"
    ```

- If you have the GOPROXY variable, you must do one of the following.

    ```bash
    # You must either comment it out of your bashrc and refresh by running
    source ~/.bashrc
    # Alternatively, you can unset the GOPROXY variable in your shell. Note that you must do this in every new shell session.
    unset GOPROXY
    ```

- Configure git to access repo with PAT instead. This will automatically authenticate downloading from this repository.

    ```bash
    git config --global url."https://$READPAT@dev.azure.com/service-hub-flg/service_hub_validation/_git/service_hub_validation_service".insteadOf "https://dev.azure.com/service-hub-flg/service_hub_validation/_git/service_hub_validation_service"
    ```

- By setting up GOPRIVATE and the git configuration as mentioned above, Go will be able to access modules stored in private Azure DevOps repo. For more information, refer to the [Azure DevOps Git docs](https://learn.microsoft.com/en-us/azure/devops/repos/git/go-get?view=azure-devops).

### [Optional] Using a Remote Proto File from a GitHub Repository

The default assumption is that the proto file (i.e. `api.proto`) exists locally in the `api/v1/proto` directory. Several files in the service's `server` directory are updated to reflect the latest `api.proto` file changes when `make service` in the `api/v1` directory is run.

However, we do add support for the case that the proto file exists remotely in a Github repository. Should you want to use this remote proto file in the Github repo, follow the below steps:

1. Follow the instructions linked [here](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens#creating-a-personal-access-token-classic) to generate a personal access token in Github. Make sure to give it access to repositories (public and/or private). 
2. Run the following line in your current terminal or add the line to your bashrc file such that its applicable to all your new terminals.
    ```bash
    export GITHUB_TOKEN=[your Github PAT value]
    ```
3. Refresh your source file or restart terminal for changes to take effect.

    ```bash
    source ~/.bashrc
    ```
4. Remember you will need to rotate it in accordance to the expiration date you set when creating the PAT.

### Generated Directories

We assume you have generated the below directories:

- **shared-resources**
- **serviceDirectoryName**
- **pipeline-files**

If you have not generated these directories, please return to the `service_hub` repo.

## Terminology

| Term   |  Examples | Definition |
|---|---|---|
| shared resources |---|  Azure resources that are used by multiple services. |
| service resources |---| Azure resources that are just for a single service. |
| generated directory |---|  The directory that contains all the generated content (the directory that holds the following directories). |
| service directory |mygreeterv3| The directory within the generated directory that contains the generated service. The directory holds both service code and service resources.  |
| shared resource directory |shared-resources| The directory within the generated directory that contains  shared resources. |

## Quick Setup (manually)

The following shows you the steps on how to get a working setup quickly.

If you want to understand why we are doing each of the steps, or require more detail, please look at the corresponding README's in the [shared-resources directory](../shared-resources/README.md) and the serviceDirectoryName directory -> serviceDirectoryName/README.md

```bash
# --------------------------------
# Assuming the current directory is the generated directory.
cd README

# Generate environment config (env-config.yaml)
# It will be stored in the generated directory because both shared-resources 
# and serviceDirectoryName will use the file.
# The following makefile target assumes you have an environment variable called $subscriptionId and 
# there is no default value stored.
# The location and serviceImageTag variables have default values that the make command will use if 
# the variables are not set in the environment. location default is set to "westus" and serviceImageTag is "latest".
# Either set the environment variables before running the command,
# or copy and paste the variables into env-config.yaml.
# Make sure the subscriptionId matches the id you used to log into Azure CLI.
# Make sure resourcesName value is present. Otherwise, set a value using only alpha numeric characters.
make genEnvConfig
cd ..

# --------------------------------
# Assuming the current directory is the generated directory.
cd shared-resources

# Templates env-config.yaml values into all the required files. 
# Assuming env-config.yaml exists in your generated directory.
make template-files

# Create shared resources
# It takes about 10 minutes
make deploy-resources

# Return to the generated directory
cd ..

# --------------------------------
# Assuming the current directory is the generated directory.
cd serviceDirectoryName

# Initialize the service. 
# Only need run once for freshly generated code or when something is wrong.
# To only initialize the api module add the parameter "true" when running the following command.
./init.sh

cd server

# Templates env-config.yaml values into all the required files.
# Assuming env-config.yaml exists in your generated directory.
make template-files

# Create service specific resources
# It takes 5-10 minutes
make deploy-resources

# Build image (can be done in parallel with deploy-resources). 
# If you are using macOS, building a Docker image locally won't work
# because we currently don't support cross-platform build. 
# Instead, you need to use DevBox to build the Docker image.
# If you do not have a DevBox set up, please follow the instructions in 
# https://msazure.visualstudio.com/CloudNativeCompute/_wiki/wikis/CloudNativeCompute.wiki/358303/Dev-Box

# Make sure your api module is tagged to the right version in the repo.
# This is the production build process where a specific api module version is used.
make build-image

# If you want to build the image to be multi-architecture (linux/arm64 and linux/amd64) use the following command. It currently uses the fixed tagged api module.
make build-multiarch-image

# This is the debug build process where the latest api code is used. It uses go.work to achieve that.
make build-workspace-image

# Push image to ACR. 
# Need to wait for the completion of `make deploy-resources` of
# the shared resources, as it is where the ACR is created.
make push-image

# (If svc running on aks cluster) Upgrade service on AKS cluster
make upgrade
# (If svc not running on aks cluster) Deploy service to AKS cluster
make install

# Refresh your bashrc such that your terminal has updated kubeconfig
source ~/.bashrc
```

<!-- TODO: Support cross-platform image build -->

To check if your service is deployed correctly, follow the instructions in serviceDirectoryName/README.md

### Command Explanation

- make genEnvConfig
  - Each developer has his/her own instance of the  service. The instance is identified with the resourcesName in env-config.yaml under the generated directory.
  - By running this command, a file named env-config.yaml is created. It stores the developer's instance information such as subscriptionId, resourcesName, and serviceImageTag. This information is used by the make template-files command.
  - Developers can share the same subscriptionId. The combination of subscriptionId and resourcesName determines the developer's instance.
- make template-files
  - By running this command, the developer's resourcesName is generated into the developer's local environment. When the developer runs the following commands, the resources, the images, and the installation will be named with the developer's resourcesName.
    - Under the hood, all files named "template-{blah}" will be used as template and a corresponding file named "{blah}" will be generated. The files named "{blah}" will have the developer's resourcesName. This ensures each developer can create his/her own instance in the same subscription.
  - This command only needs to be run when the content in env-config.yaml changes.
  - For services, it will update the service's all components: service resources, service config, etc.
  - For shared resources, the command updates the name of the shared resources.

## Quick Setup (using development pipeline)

- If you did not have the pipelines be automatically created for you by service_hub, please follow the steps in the [test-and-dev-pipelines_README.md](../pipeline-files/test-and-dev-pipelines_README.md) file to create and setup your test and development pipelines.

### Start the dev pipeline

(TODO: Link dev pipeline)

Go into pipelines, select the `Service Resource And Code Development Pipeline` and manually trigger the pipeline by clicking "Run pipeline". Choose the branch you would like to use.

[Optional] If you would like to change the subscription id that the dev pipeline uses.

- Under `Advanced options` select `Variables`.
  - Change SUBSCRIPTION_ID to your subscription id.

Once the dev pipeline has finished running, quick set up will be different as many of the commands have already been done by the dev pipeline.

### Finding your resources

The development pipeline will display warnings that will point you to the resource group where your resources were created.

Clicking on the warning will direct you to a link to the azure portal for that resource group.

### [Optional] Download adx dashboard file 
- If you want to access the dashboard file for the resources/application you have deployed without using your local setup, you can download it from published artifacts. Each service will have its own dashboard file labelled serviceDirectoryName-dashboard.

### Copy over required files

From the pipeline run, download the following files and move them to their appropriate locations.

- env-config.yaml : move to root of the generated directory.
- .Azuresdk_properties_outputs.yaml : move to serviceDirectoryName/server/artifacts


### Make sure you are logged into azure

```bash
# Log into azure 
az login
# Set you subscription to the same subscription id that was used for the dev pipeline
az account set --subscription $subscriptionId
```

### Template all files

```bash
# Assuming your current direction is the generated directory.
cd shared-resources
# Templates env-config.yaml values into all the required files. We assume env-config.yaml exists in your generated directory.
make template-files
# Return to the generated directory
cd ..
# --------------------------------
# Assuming your current direction is the generated directory.
cd serviceDirectoryName/server
# Templates env-config.yaml values into all the required files. We assume env-config.yaml exists in your generated directory.
make template-files
```

### Connect to your aks cluster

```bash
# Assuming your current direction is the generated directory.
cd serviceDirectoryName/server/generated
# Gets credentials for a managed kubernetes cluster
make connect cluster
# Refresh your bashrc such that your terminal has updated kubeconfig
source ~/.bashrc
```

To check if your service is deployed correctly, follow the instructions in serviceDirectoryName/README.md

## Deleting your resources

### Through the development pipeline

1. Go into pipelines, select the `Service Resource And Code Development Pipeline` and manually trigger the pipeline by clicking "Run pipeline".

2. Under "Advanced options"
    - Select variables.
        - Set RESOURCES_NAME to be the id of your resources. (If you previously created resources through dev pipeline, it would have been output as a warning. If you did it manually, it is the `resourcesName` variable in your env-config.yaml file).
        - Set DELETE to be true.
    - Select stages to run. ONLY select the "Delete all resources" stage. Ignore the warning about the skipped stage.
3. Click `Run pipeline`. This pipeline will delete your resources if the service principal pre-set for the pipeline has access to the subscription the resources were created in. (If you used a different subscription, refer to [Manual deletion](#manual-deletion))

### Manual deletion

Run the following command, where $resourcesName is the `resourcesName` variable in your env-config.yaml file.

```bash
az group delete -n servicehubval-$resourcesName-rg --yes
```

## Making changes to Bicep Resources

### Modifying the generated resources

If you need to adjust the parameter values for your resources, follow these steps:

1. Check out the [Bicep Module Registry](https://github.com/Azure/bicep-registry-modules/tree/main) for available parameters which are indicated by the syntax `param` in the bicep files. These directions do not apply for `aks-managed-cluster` and `subscription-role-assignment` bicep resource. See *note* below.
2. Modify the parameters in the relevant Bicep files in shared-resources/resources or in serviceDirectoryName/server/resources.
3. After making the changes, run `make deploy-resources` to apply the updated configurations.

*Note: For AKS managed cluster or subscription role assignment, refer to the bicep-modules directory in the Service Hub registry. Locate the corresponding directory and review the service-hub-generated-module.bicep for available parameters.*

### Bicep Resources

To explore available Bicep modules and their definitions, check out the following links:

- [AVM Bicep Resource Modules](https://azure.github.io/Azure-Verified-Modules/indexes/bicep/bicep-resource-modules/): This resource provides a list of Bicep modules along with their locations in the Bicep Module Registry.
- [Bicep Module Registry](https://github.com/Azure/bicep-registry-modules/tree/main): The Bicep Module Registry contains modules used by the AVM Bicep Resource Modules.
- [Azure Container Registry Example](https://github.com/Azure/bicep-registry-modules/tree/main/avm/res/container-registry/registry). This links to the azure container registry directory in the bicep module registry. In the README.md, there are examples of how to add the azure container registry.

Keep in mind that we’ve already set up resources for you using the AVM Bicep Resource Modules. If you wish to modify or add resources, refer to the Bicep resource modules and registry for guidance. Detailed examples are provided in their README.md files.

### "Define Once, Reference Everywhere Else" Rule

We introduce the concept of defining a resource once and only referencing it in other locations. We use the `resource` and `existing` syntax to reference the resource. It prevents the modification of the resource from its non-source file.

An example in our repo is the log analytics workspace.

Source file: `shared-resources/resources/Main.SharedResources.Template.bicep`. This defines the resource.

```bicep
module workspace 'br/public:avm/res/operational-insights/workspace:0.3.4' = {
  name: 'servicehubval-${resourcesName}-workspaceDeploy'
  scope: resourceGroup(subscriptionId, resourceGroupName)
  params: {
    name: 'servicehubval-${resourcesName}-workspace'
    location: rg.outputs.location
  }
}
```

Source file: `serviceDirectoryName/server/resources/azuresdk/Azuresdk.ServiceResources.Template.bicep`. This references the existing resource defined in the previous file.

```bicep
resource logAnalyticsWorkspace 'Microsoft.OperationalInsights/workspaces@2022-10-01' existing = {
  name: 'servicehubval-${resourcesName}-workspace'
  scope: resourceGroup(subscriptionId, resourceGroupName)
}
```

# Code Overview

Depending on how the code is generated, it may not have all the components described here.

High level, the repo will store all the code of a service. A service is composed of several microservices. The microservices communicate with each other and provide a service that is visible to external users. One example service is Azure Kubernetes Service (AKS). It provides ARM APIs to external users. Internally, it is composed of multiple microservices, e.g., RP frontend, RP backend, database, etc.

As a result of such microservice concept, the top level directories are classified to the following:

- Microservices. Each microservice directory stores one microservice. It usually will have at least two subdirectories.
  - api: The API of the microservice. Other microservices will use the API to communicate with the microservice.
  - server: The implementation of the microservice. It will contains the source code, the microservice specific resource declaration, deployment definition, etc.
  - See more details from the README.md in each microservice's directory.
- Shared resources. This directory stores resources shared by all microservices. As microservices need to communicate with each other, they usually run on shared platform such as a Kubernetes cluster. Functionalities such as log collection is better handled by one common platform too. Thus these shared resources will be declared in this directory.
  - Shared resources reduce the management overhead of each microservices. They also introduce dependency. During deployment, all microservices potentially need to wait for the deployment of the shared resources. In emergency, if a microservice knows that the old shared resource configuration is still good for their deployment, they don't need to wait for the latest deployment of the shared resources.
- Pipeline. This directory stores shared Azure DevOps (ADO) pipeline yaml files. Each microservice and the shared resource have pipeline yaml files stored in their own directory.
  - Test/development pipeline. This pipeline yaml file is the entry point for the shared resources and all microservices. It calls the pipeline yaml files in each microservice directory and the shared resource directory.
    - Each microservice defines its own work. Usually it runs unit test, generate test coverage, build docker image, push image to ACR, provision service specific resources, deploy the service code, etc.
    - It is used by developers to create their own isolated environment. All microservices will be deployed and work together. It can also be used as a PR gate to ensure changes to the main branch are tested.
  - OneBranch/Ev2 build (MSFT specific). This pipeline yaml file can be shared by all microservices and the shared resources. When being used for a specific microservice, users can create their own pipeline which uses the same yaml file but the pipeline's variable `directoryName` is set to user's directory.
    - It builds docker images for microservices.
    - It builds Ev2 artifacts for both microservices and shared resources.
    - Shared resource can be treated as a special microservice. It only deploys the resources. There is no code related work.
  - OneBranch/Ev2 release (MSFT specific). This pipeline yaml file cannot be shared because of the limitation of the OneBranch template: the build pipeline's name can only be hardcoded into the release pipeline. Thus each microservice has a copy of this file which differs by one line: the name of its build pipeline.
    - It downloads artifacts generated by the build pipeline and calls Ev2 to deploy them to various regions of Azure.
  - See [Ev2_README.md](./Ev2_README.md) for details.
- README. This directory stores information for the whole service. Read the docs in this directory first.
  - Further more, it stores the Makefile which generates the unique configuration for each developer so that they can create their isolated development environment. The Test/development pipeline uses the same mechanism. More details are at [Command Explanation](#command-explanation)
- config-files. This directory stores the configs used to generate the microservices, shared resources, and the pipeline. If users want to regenerate them, they can modify the config files and regenerate.

The following directories exist only when users use Service Hub to generate a complete user environment (ADO project, repo, and pipeline)

- terraform-files. The terraform definitions that create the whole user environment ADO project, repo, and pipeline. If users want to update their user environment, they can modify the terraform definitions rather than modify them directly via GUI.

To help understand the directory structure, please take a look at the `ai-summary.md` files under each directory in repo [service_hub_validation_service](https://service-hub-flg.visualstudio.com/Service_Hub_Validation/_git/service_hub_validation_service). Those files are generated by AI. The `README.md` in various directories are generated by human to capture high level concepts and instructions.
