#!/bin/bash -x

onlyApi=$1
if [ -z "$onlyApi" ]
then
    onlyApi=false
fi
# Install the credential provider for Azure Artifacts feed if it is not already installed
if [ ! -f ~/.nuget/plugins/netcore/CredentialProvider.Microsoft/CredentialProvider.Microsoft.dll ]; then
    echo "Credential provider not found. Installing..."
    curl -L https://raw.githubusercontent.com/Microsoft/artifacts-credprovider/master/helpers/installcredprovider.sh | sh
else
    echo "Credential provider already installed."
fi

# TODO: Switch to git credential manager
# Set the endpoint for the Azure Artifacts feed
export VSS_NUGET_EXTERNAL_FEED_ENDPOINTS='{"endpointCredentials": [{"endpoint":"https://pkgs.dev.azure.com/service-hub-flg/service_hub_validation/_packaging/service_hub_validation__PublicPackages/nuget/v3/index.json", "username":"user", "password":"'$READPAT'"}]}'

# Install and Build
# TODO: Uncomment the following line once ProviderHub template has expected AKS middleware
# dotnet new install Microsoft.TypeSpec.ProviderHub.Templates
# dotnet new typespec-providerhub -P CsharpUserRp -n csharpuserrp -o server --allow-scripts yes --force

cd server
./setup.sh
cd src/csharpuserrp
dotnet build
if [ $? -ne 0 ]; then
    echo "Service initialization failed."
    exit 1
fi
