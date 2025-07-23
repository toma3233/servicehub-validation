outputDir=$1


echo "Test Shared Resources"
# TODO generalise this test more.
#./resources/testResourceNames.sh ev2

  echo "Package Helm"
  # Install helm if not already installed
  if ! command -v helm &> /dev/null; then
      echo "Helm not found. Installing Helm..."
      curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3
      chmod 700 get_helm.sh
      ./get_helm.sh
  fi
  # Package service
  cd deployments
  helm package .
  cd ..

  echo "Copy Helm Package"
  mv deployments/geneva-0.1.0.tgz $outputDir/Ev2Specs/Build

  echo "Copy Helm Values"
  find . -name "*values*.yaml" -exec cp {} $outputDir/Ev2Specs/Build;

  echo "Rename Helm Values files"
  for file in $outputDir/Ev2Specs/Build/template-*.yaml; do
    if [ -f "$file" ]; then
      mv "$file" "${file/template-/}"
    fi
  done

echo "Copy Kubectl Yaml files to Build Folder"
find deployments -name "*.yaml" -exec cp {} $outputDir/Ev2Specs/Build \;

cd $outputDir/Ev2Specs/Build

echo "Get Argo controller installation yaml"
curl -fsSL -o install.yaml https://github.com/argoproj/argo-rollouts/releases/latest/download/install.yaml
cd ..

cd $outputDir/Ev2Specs

# Make sure the global subscription IDs are set in the configuration files
testGlobalSubscriptionId=$(jq -r '.Settings.globalSubscriptionId' Configurations/Test/Configuration.json)
prodGlobalSubscriptionId=$(jq -r '.Settings.globalSubscriptionId' Configurations/Prod/Configuration.json)
if [ -z "$testGlobalSubscriptionId" ] || [ -z "$prodGlobalSubscriptionId" ]; then
    echo "Error: Global subscription IDs (globalSubscriptionId) are not set in the configuration files."
    exit 1
fi