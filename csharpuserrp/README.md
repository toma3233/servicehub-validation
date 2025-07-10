# csharpuserrp

## Overview
This template generates rest service code based on TypeSpec ProviderHub template, and then add deployment and monitoring templates. It allows you to define API spec using Typespec. Models and controllers will automatically be generated based on Typespec configurations.

All files under server/src/ and server/setup.sh are Typespec ProviderHub template generated code. ProviderHub template uses Typespec as a single source of truth to generate ARM OpenAPI Swagger spec. Although the generated code is a UserRP service, the reason to have an ARM Typespec+Swagger spec is that they can check in to the azure-rest-api-specs or azure-rest-api-specs-pr repos. For RPaaS APIs, it is required to have the Swagger hosted in one of these repos for manifest/registration and live validation of the APIs. 
TODO: Clean up the Swagger files since in the template when possible as it seems not needed by UserRP service owner. 

Learn more about the template in [Reference](#reference) section.

### Installations

- Follow the steps to install [.NET](https://dotnet.microsoft.com/en-us/download) if you do not already have it.

- Follow the steps to install [Docker](https://docs.docker.com/engine/install/) if you do not already have it.

- Follow the steps to install [Typespec](https://azure.github.io/typespec-azure/docs/typespec-getting-started/) if you don't already have it.

- Setup credentials for authentication to Azure Artifacts Feed

  - Follow the steps to install [Azure Artifacts Credential Provider](https://github.com/Microsoft/artifacts-credprovider) if you do not already have it.
  - To avoid manually adding --interactive to every dotnet command simply set the following variable in your `~/.bashrc` file such that the credential provider uses that endpoint.

    ```bash
    export VSS_NUGET_EXTERNAL_FEED_ENDPOINTS='{"endpointCredentials": [{"endpoint":"https://pkgs.dev.azure.com/service-hub-flg/service_hub_validation/_packaging/service_hub_validation__PublicPackages/nuget/v3/index.json", "username":"user", "password":"'$READPAT'"}]}'
    ```

## Setup and Development

### Initialize service

```bash
./init.sh
```

### Run Service Locally

There is a simple way to run the CsharpUserRp service, after everything has been properly generated. Inside the CsharpUserRp directory, you can run the server.

Make sure you have installed the Azure Artifacts Credential Provider and set the associated endpoint variable as mentioned in the [Prerequisites](#prerequisites) section. Without this, the dotnet commands mentioned below will fail.

### Server Configuration

#### Server

To run the server:

```bash
cd server/src/csharpuserrp
dotnet run
```

By default the server starts in port `localhost:6020`.

To change API definitions, go to server/src/typespec/typespec/main.tsp and make modifications, rebuild the .NET project and the new specs will be automatically refreshed.


#### Help

You can run help on every command in order to get more information on how to use them.

Examples:

```bash
dotnet run --help
```

### Deployment

Refer to README/README.md

## Reference
[Typespec Azure Introduction](https://azure.github.io/typespec-azure/docs/typespec-getting-started/)

[Typespec ProviderHub User Guide](https://github.com/Azure/typespec-azure-pr/blob/providerhub/docs/getstarted/providerhub/step01-create-userrp-project.md)

[Typespec ProviderHub Intro](https://github.com/Azure/typespec-azure-pr/blob/providerhub/docs/intro.md)

[TypeSpec ProviderHub Template](https://github.com/Azure/typespec-azure-pr/tree/providerhub/packages/typespec-providerhub-templates)

[Typespec ProviderHub Controller](https://github.com/Azure/typespec-azure-pr/tree/providerhub/packages/typespec-providerhub-controller)

[RPaaS Introduction](https://armwiki.azurewebsites.net/rpaas/gettingstarted.html)
