set -e

outputDir=$1
deleteConfigType=$2

if [[ "$deleteConfigType" == *Daily* ]]; then
    export deleteConfigType="Daily"
elif [[ "$deleteConfigType" == *Manual* ]]; then
    export deleteConfigType="Manual"
else
    echo "Invalid deleteConfigType provided. Please use 'Daily' or 'Manual'."
    exit 1
fi

cd $outputDir/Ev2Specs

# Packaging everything together: script and crane
tar -cvf delete-resources.tar ./Shell/*

jq '.Settings.deleteConfigType'=env.deleteConfigType Configurations/Test/Configuration.json > Configurations/Test/Configuration.json.tmp
mv Configurations/Test/Configuration.json.tmp Configurations/Test/Configuration.json
jq '.Settings.deleteConfigType'=env.deleteConfigType Configurations/Prod/Configuration.json > Configurations/Prod/Configuration.json.tmp
mv Configurations/Prod/Configuration.json.tmp Configurations/Prod/Configuration.json

testGlobalSubscriptionId=$(jq -r '.Settings.globalSubscriptionId' Configurations/Test/Configuration.json)
prodGlobalSubscriptionId=$(jq -r '.Settings.globalSubscriptionId' Configurations/Prod/Configuration.json)
if [ -z "$testGlobalSubscriptionId" ] || [ -z "$prodGlobalSubscriptionId" ]; then
    echo "Error: Global subscription IDs are not set in the configuration files."
    exit 1
fi