#!/bin/bash
# Check if the correct number of arguments is provided
if [ $# -ne 2 ]; then
    echo "Usage: $0 <property_output_json_file> <property_output_yaml_file>"
    exit 1
fi
# Check if the input JSON file exists
PROPERTY_OUTPUT_JSON_FILE="$1"
PROPERTY_OUTPUT_YAML_FILE="$2"
if [ ! -e "$PROPERTY_OUTPUT_JSON_FILE" ]; then
    echo "Error: Input JSON file '$PROPERTY_OUTPUT_JSON_FILE' does not exist."
    exit 1
fi
# Read the values from the JSON file
MANAGED_IDENTITY_CLIENT_ID=$(jq -r '.managedIdentityClientId.value' "$PROPERTY_OUTPUT_JSON_FILE")
KEYVAULT_NAME=$(jq -r '.keyVaultName.value' "$PROPERTY_OUTPUT_JSON_FILE")
TENANT_ID=$(jq -r '.tenantId.value' "$PROPERTY_OUTPUT_JSON_FILE")
AKV_SECRETS_PROVIDER_CLIENT_ID=$(jq -r '.akvSecretsProviderClientId.value' "$PROPERTY_OUTPUT_JSON_FILE")
CERT_NAME=$(jq -r '.certName.value' "$PROPERTY_OUTPUT_JSON_FILE")

# Generate the YAML configuration directly
YAML_CONFIG="serviceAccount:
  annotations:
    azure.workload.identity/client-id: \"$MANAGED_IDENTITY_CLIENT_ID\"
keyVaultName: \"$KEYVAULT_NAME\"
tenantId: \"$TENANT_ID\"
akvSecretsProviderClientId: \"$AKV_SECRETS_PROVIDER_CLIENT_ID\"
certName: \"$CERT_NAME\""

# Write the YAML configuration to the specified output file
echo "$YAML_CONFIG" > "$PROPERTY_OUTPUT_YAML_FILE"
echo "$PROPERTY_OUTPUT_YAML_FILE generated successfully!"
