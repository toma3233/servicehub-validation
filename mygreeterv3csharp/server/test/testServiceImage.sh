#!/bin/bash
#Define color codes for printing to help analysis.
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'
#---------
echo "${GREEN}This is a placeholder for real test.${NC}"
cd mygreeterv3csharp

# Install the credential provider for Azure Artifacts feed if it is not already installed
if [ ! -f ~/.nuget/plugins/netcore/CredentialProvider.Microsoft/CredentialProvider.Microsoft.dll ]; then
    echo "Credential provider not found. Installing..."
    curl -L https://raw.githubusercontent.com/Microsoft/artifacts-credprovider/master/helpers/installcredprovider.sh | sh
else
    echo "Credential provider already installed."
fi

# Set the endpoint for the Azure Artifacts feed
export VSS_NUGET_EXTERNAL_FEED_ENDPOINTS='{"endpointCredentials": [{"endpoint":"https://pkgs.dev.azure.com/service-hub-flg/service_hub_validation/_packaging/service_hub_validation__PublicPackages/nuget/v3/index.json", "username":"user", "password":"'$READPAT'"}]}' 

cd server
dotnet restore Src/Server/Server.csproj
dotnet restore Src/Client/Client.csproj
cd ..
dotnet test Server.Tests
if [ $? -ne 0 ]; then
    echo "Unit tests for server modules failed."
    exit 1
fi
echo "${GREEN}Server tests were successful.${NC}"


