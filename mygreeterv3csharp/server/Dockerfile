# Stage 1: Build the .NET application
FROM mcr.microsoft.com/dotnet/sdk:9.0 AS build
WORKDIR /app

# Install the Azure Artifacts Credential Provider
RUN curl -L https://raw.githubusercontent.com/Microsoft/artifacts-credprovider/master/helpers/installcredprovider.sh | sh

# Copy nuget.config
COPY Src/nuget.config .

# Set the PAT argument and environment variable for Azure Artifacts
ARG PAT
ENV VSS_NUGET_EXTERNAL_FEED_ENDPOINTS='{"endpointCredentials": [{"endpoint":"https://pkgs.dev.azure.com/service-hub-flg/service_hub_validation/_packaging/service_hub_validation__PublicPackages/nuget/v3/index.json", "username":"docker", "password":"'${PAT}'"}]}'

# Copy and restore the Client project
COPY Src/Client/*.csproj ./Src/Client/
RUN dotnet restore Src/Client/Client.csproj

# Copy and restore the Server project
COPY Src/Server/*.csproj ./Src/Server/
RUN dotnet restore Src/Server/Server.csproj

# Copy the entire source code for both Client and Server projects
COPY Src/ ./Src/

# Publish the Client project
RUN dotnet publish Src/Client/Client.csproj -c Release -o /app/client

# Publish the Server project
RUN dotnet publish Src/Server/Server.csproj -c Release -o /app/server

# Stage 2: Create a runtime image
FROM mcr.microsoft.com/dotnet/aspnet:9.0 AS runtime
WORKDIR /app

# Copy the published Client and Server binaries
COPY --from=build /app/client ./client
COPY --from=build /app/server ./server
