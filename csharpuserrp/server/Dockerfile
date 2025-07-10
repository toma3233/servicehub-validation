# Stage 1: Build the .NET application
FROM mcr.microsoft.com/dotnet/sdk:6.0 AS build
WORKDIR /app

# Install the Azure Artifacts Credential Provider
RUN curl -L https://raw.githubusercontent.com/Microsoft/artifacts-credprovider/master/helpers/installcredprovider.sh | sh

# Install Node.js (required for TypeSpec)
RUN apt-get update && apt-get install -y curl && \
    curl -fsSL https://deb.nodesource.com/setup_20.x | bash - && \
    apt-get install -y nodejs

# TODO: Install curl for testing only, remove curl once we have a client calling APIs
RUN apt-get update && apt-get install -y curl

# Copy nuget.config
COPY nuget.config .

# Set the PAT argument and environment variable for Azure Artifacts
ARG PAT
ENV VSS_NUGET_EXTERNAL_FEED_ENDPOINTS='{"endpointCredentials": [{"endpoint":"https://pkgs.dev.azure.com/service-hub-flg/service_hub_validation/_packaging/service_hub_validation__PublicPackages/nuget/v3/index.json", "username":"docker", "password":"'${PAT}'"}]}'

# Copy the entire source code for the project
COPY src/ ./src/

# Install TypeSpec dependencies
WORKDIR /app/src/typespec/typespec
RUN npm install

# Publish the project
WORKDIR /app
RUN dotnet publish src/csharpuserrp/csharpuserrp.csproj -c Release -o /app/csharpuserrp

# Stage 2: Create a runtime image using the ASP.NET runtime
FROM mcr.microsoft.com/dotnet/aspnet:6.0 AS runtime
WORKDIR /app

# TODO: Install curl for testing only, remove curl once we have a client calling APIs
RUN apt-get update && apt-get install -y curl

# Copy the published binaries
COPY --from=build /app/csharpuserrp ./csharpuserrp
