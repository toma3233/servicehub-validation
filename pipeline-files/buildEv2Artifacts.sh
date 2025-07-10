# Create Artifacts Files
# Assumptions for the script:
# - If isService is true, 
#   - the directory being used to build artifacts is $directoryName/server
#   - there is a testServiceImage.sh file in the $directoryName/server/test being used to test the service.
#   - there is a deployments folder in the $directoryName/server directory that contains the helm chart for the service.
# - There is an folder labelled "Ev2" in the directory being used to build artifacts
# - There is a resources folder in the directory being used to build artifacts that stores the bicep files for the resources and any corresponding parameters files (starting with "template-")

set -e
directoryName=$1
outputDir=$2
isService=$3
buildNumber=$4
isLocal=$5
buildReason=$6
sourceBranch=$7
pullRequestId=$8
restrictToPR=$9

# We are artifically restricting this build pipeline to either only run on the default protected branch or from a pull request.
# This is to ensure that the build pipeline is not run on any other branches as the unique id is tied to either the pull request id or the default branch.
if [ "$restrictToPR" = "true" ]; then
  if [ "$buildReason" != "PullRequest" ] && [ "$isLocal" = "false" ]; then
    if [ "$sourceBranch" != "refs/heads/master" ]; then
      echo "This build is not being triggered by a PR. Please only run this pipeline through a Pull Request for resources to work as expected. Exiting."
      exit 1
    else 
      echo "This build is being triggered by master."
    fi
  fi
fi

if [ "$isService" = "true" ]; then
  directory=$directoryName/server
else 
  directory=$directoryName
fi

cd $directory
currPath=$(pwd)
echo "Copy Ev2 folder to out directory"
cp -rT Ev2 $outputDir

mkdir -p $outputDir/Ev2Specs/Build


echo "Test and Build Directory Specific Code"
if [ "$isService" = "true" ]; then
  cd ..
  cd ..
  echo "Test Service"
  ./${directory}/test/testServiceImage.sh
  
  cd $directory

  if [ "$isLocal" = "true" ]; then
    echo "Building Docker Image"
    cd generated
    docker build --build-arg PAT=$READPAT -t $directoryName:$buildNumber -f ../Dockerfile ./../
    docker save -o ${directoryName}-image.tar $directoryName:$buildNumber
    cp ${directoryName}-image.tar $outputDir/Ev2Specs/Build
    cd ..
  fi

  # Run customized build script for the service if it does exist.
  if [ -f "buildCustomEv2.sh" ]; then
    echo "Running build script for service"
    ./buildCustomEv2.sh
  fi

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
  mv deployments/$directoryName-0.1.0.tgz $outputDir/Ev2Specs/Build

  echo "Copy Helm Values"
  find . -name "*values*.yaml" -exec cp {} $outputDir/Ev2Specs/Build \;

  echo "Rename Helm Values files"
  for file in $outputDir/Ev2Specs/Build/template-*.yaml; do
    if [ -f "$file" ]; then
      mv "$file" "${file/template-/}"
    fi
  done

  cd $outputDir/Ev2Specs

  # Packaging everything together: script and crane
  tar -cvf push-image-to-acr.tar ./Shell/*
  
else
  echo "Building non-service artifacts"
  if [ -f "buildCustomEv2.sh" ]; then
    echo "Running build script"
    ./buildCustomEv2.sh "$outputDir"
  fi
fi

cd $currPath

# copy bicep templates and parameters only if resources folder exists
resourceDir="resources"
if [ -d "$resourceDir" ]; then
    mkdir -p $outputDir/Ev2Specs/Templates
    mkdir -p $outputDir/Ev2Specs/Parameters

    echo "Copy all bicep files in $resourceDir to $outputDir/Ev2Specs/Templates"
    find "$resourceDir" -name "*.bicep" -exec cp {} $outputDir/Ev2Specs/Templates \;

    echo "Copy parameters file to $outputDir/Ev2Specs/Parameters"
    find "$resourceDir" -name "*Parameters.json" -exec cp {} $outputDir/Ev2Specs/Parameters \;

    echo "Rename parameters file"
    if compgen -G "$outputDir/Ev2Specs/Parameters/template-*.json" > /dev/null; then
        for file in $outputDir/Ev2Specs/Parameters/template-*.json; do
            mv "$file" "${file/template-/}"
        done
    fi
else
    echo "No resources folder in $directory; skipping bicep and parameters copy"
fi

echo "Convert Bicep Templates to json"
bicepVersionBefore=$(az bicep version --query "version" -o tsv)
az bicep upgrade
bicepVersionAfter=$(az bicep version --query "version" -o tsv)
echo "Bicep version upgrade: $bicepVersionBefore -> $bicepVersionAfter"
templatesDir="${outputDir}/Ev2Specs/Templates"
if [ -d "$templatesDir" ]; then
    echo "Convert Bicep Templates to json in $templatesDir"
    cd "$templatesDir"
    if compgen -G "*.bicep" > /dev/null; then
        for f in *.bicep; do az bicep build --file "$f"; done
    else
        echo "No Bicep templates found in $templatesDir; skipping conversion"
    fi
    cd ..
else
    echo "No Templates directory at $templatesDir; skipping Bicep conversion"
fi

echo "Package Script and Set Build Version"
versionContent=$buildNumber
versionFileName="./Version.txt"

# If the file already exists, delete it so you can recreate it and repopulate with the build number
if [ -f "$versionFileName" ]; then
  rm "$versionFileName"
fi

# Create the version file with the build number
echo -n "$versionContent" > "$versionFileName"

cd ../..

# TODO: Adjust this when introducing ring configuration
echo "Pull request source branch is set. Updating Configuration.json with pull request ID."
if [ "$sourceBranch" != "refs/heads/master" ] && [ "$restrictToPR" = "true" ]; then  
  export resourcesName="${pullRequestId}TestCorp"
  export deletionDate=$(date +%Y-%m-%d -d "+3 days")
  jq '.Settings.resourcesName'=env.resourcesName $outputDir/Ev2Specs/Configurations/Test/Configuration.json > $outputDir/Ev2Specs/Configurations/Test/Configuration.json.tmp
  jq '.Settings.deletionDate'=env.deletionDate $outputDir/Ev2Specs/Configurations/Test/Configuration.json.tmp > $outputDir/Ev2Specs/Configurations/Test/Configuration.json
  echo "##vso[task.logissue type=warning]Your resources created in the test environment will have '$resourcesName' as the unique id and '$deletionDate' as the deletion date."
fi
