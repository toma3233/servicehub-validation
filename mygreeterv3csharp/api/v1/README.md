## Overview

This nuget package contains the Api/V1 package. 

### `Proto/api.proto`

Defines a set of requests and responses for the MyGreeterCsharp service. The Api/V1 package includes the generated gRPC service from `api.proto`, including functions like `SayHello()`, `CreateResourceGroup()`, etc. More info can be found [here](https://dev.azure.com/service-hub-flg/service_hub/_wiki/wikis/service_hub.wiki/159/MyGreeterV3-.NET-Documentation?anchor=proto-file).

#### Modify the API

Whenever the API is changed, you need to run the following command to regenerate the code.

```bash
cd api/v1
make service
```

### `Client/Client.cs`, 

`NewClient()` function returns a new client that connects to the specified remote address. 

### Nuget Packages

Instructions on how to build, release, and publish nuget packages can be found [here](https://dev.azure.com/service-hub-flg/service_hub/_wiki/wikis/service_hub.wiki/159/MyGreeterV3-.NET-Documentation?anchor=nuget-packages).

