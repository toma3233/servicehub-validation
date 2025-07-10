#!/bin/bash -x

# The api version is fixed based on the value of the SERVICEHUB_APIV1_VERSION variable.
# It must be specified in double quotes
# The automated package versioning logic bumps the PATCH version only
SERVICEHUB_APIV1_VERSION="1.0.37"
SERVICEHUB_AKSMIDDLEWARE_VERSION="1.0.6"

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

# Set the endpoint for the Azure Artifacts feed
export VSS_NUGET_EXTERNAL_FEED_ENDPOINTS='{"endpointCredentials": [{"endpoint":"https://pkgs.dev.azure.com/service-hub-flg/service_hub_validation/_packaging/service_hub_validation__PublicPackages/nuget/v3/index.json", "username":"user", "password":"'$READPAT'"}]}'

# Generic function to update a package's version in a .csproj file
update_package_version() {
    local csproj_file=$1
    local package_name=$2
    local package_version=$3

    if [ -f "$csproj_file" ]; then
        echo "Setting $package_name version to $package_version in $csproj_file"
        sed -i.bak "s|<PackageReference Include=\"$package_name\" Version=\".*\" />|<PackageReference Include=\"$package_name\" Version=\"$package_version\" />|" "$csproj_file"
        # Remove the backup file created by sed
        rm -f "${csproj_file}.bak"
    else
        echo "Warning: $csproj_file not found."
    fi
}

update_package_version server/Src/Server/Server.csproj "ServiceHub.ApiV1" "$SERVICEHUB_APIV1_VERSION"
update_package_version server/Src/Server/Server.csproj "ServiceHub.AKSMiddleware" "$SERVICEHUB_AKSMIDDLEWARE_VERSION"
update_package_version server/Src/Client/Client.csproj "ServiceHub.ApiV1" "$SERVICEHUB_APIV1_VERSION"
update_package_version server/Src/Client/Client.csproj "ServiceHub.AKSMiddleware" "$SERVICEHUB_AKSMIDDLEWARE_VERSION"
update_package_version api/v1/ApiV1.csproj "ServiceHub.AKSMiddleware" "$SERVICEHUB_AKSMIDDLEWARE_VERSION"


# Build and test
cd api/v1
make service
if [ $? -ne 0 ]; then
    echo "Make service failed."
    exit 1
fi
if [ "$onlyApi" = "true" ]
then
    echo "Only api module was initialized."
    exit 0
fi
cd ../..
dotnet test Api.Tests
if [ $? -ne 0 ]; then
    echo "Unit tests for api module failed."
    exit 1
fi

cd server
dotnet restore Src/Server/Server.csproj
dotnet restore Src/Client/Client.csproj
cd ..
dotnet test Server.Tests
if [ $? -ne 0 ]; then
    echo "Unit tests for server modules failed."
    exit 1
fi
