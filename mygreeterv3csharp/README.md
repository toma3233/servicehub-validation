# mygreeterv3csharp

## Prerequisites

### Installations

- Follow the steps to install [.NET](https://dotnet.microsoft.com/en-us/download) if you do not already have it.

- Follow the steps to install [Docker](https://docs.docker.com/engine/install/) if you do not already have it.

- Setup credentials for authentication to Azure Artifacts Feed

  - Follow the steps to install [Azure Artifacts Credential Provider](https://github.com/Microsoft/artifacts-credprovider) if you do not already have it.
  - To avoid manually adding --interactive to every dotnet command simply set the following variable in your `~/.bashrc` file such that the credential provider uses that endpoint.

    ```bash
    export VSS_NUGET_EXTERNAL_FEED_ENDPOINTS='{"endpointCredentials": [{"endpoint":"https://pkgs.dev.azure.com/service-hub-flg/service_hub_validation/_packaging/service_hub_validation__PublicPackages/nuget/v3/index.json", "username":"user", "password":"'$READPAT'"}]}'
    ```

## Setup and Development

### `Proto/api.proto`

Defines a set of requests and responses for the MyGreeterCsharp service. The Api/V1 package includes the generated Grpc service from `api.proto`, including functions like `SayHello()`, `CreateResourceGroup()`, etc.

Note that we use the remote aks middleware. This middleware is responsible for features such as logging, retry, and input validation. To learn more, please visit the [repo](https://github.com/Azure/aks-middleware-csharp).

### Initialize service

```bash
./init.sh
```

### Run Service Locally

There is a simple way to run the MyGreeterCsharp service, after everything has been properly generated. Inside the MyGreeterCsharp directory, you can run the client and server.

Make sure you have installed the Azure Artifacts Credential Provider and set the associated endpoint variable as mentioned in the [Prerequisites](#prerequisites) section. Without this, the dotnet commands mentioned below will fail.

### Client/Server Configuration

Client and Server are configured to be different projects and have their own config files. Since they are closedly related to each other, tests are written in one project to cover all classes in both projects.

#### Server

To run the server:

```bash
cd server/Src/Server
dotnet run start
```

By default the server starts in port `localhost:50051` and the enable-azureSDK-calls flag is set to false.

To run the server with the azureSDK calls enabled:

```bash
cd server/Src/Server
dotnet run start --enable-azureSDK-calls true --subscription-id <sub_id>
```

#### Client

To run the client:

```bash
cd server/Src/Client
dotnet run hello
```

By default the client sends messages to port `localhost:50051`. This can be changed by running

```bash
cd server/Src/Client
dotnet run hello --remote-addr <remote_addr>
```

#### Run Unit tests

Unit tests are organized into separate test projects within the solution. Each test project corresponds to a specific project in the main solution. To run unit tests, you can use the .NET CLI.

Example:

```bash
cd mygreeterv3csharp
dotnet test Server.Tests 
```

#### Help

You can run help on every command in order to get more information on how to use them.

Examples:

```bash
dotnet run hello --help
```

### Deployment

Refer to README/README.md

## Reference

[Service Tech Details](https://dev.azure.com/service-hub-flg/service_hub/_wiki/wikis/service_hub.wiki/159/MyGreeterV3-.NET-Documentation)

[Unit Test Best Practices](https://learn.microsoft.com/en-us/dotnet/core/testing/unit-testing-best-practices)
