#!/bin/bash

set -e

ONEBOX_ENDPOINT=${ONEBOX_ENDPOINT:-http://host.docker.internal:5000}

swaggerRootDir=$(dirname $0)

if [ ! -d $swaggerRootDir/.apitest ]; then
  mkdir $swaggerRootDir/.apitest
fi

function generateApiScenario {
  local swaggerFile=${1:-openapi.json}

  if [ ! -f $swaggerFile ]; then
    swaggerFile=$(find $swaggerRootDir -type f -name "$swaggerFile")
  fi

  if [ -z $swaggerFile ]; then
    echo "Swagger file \"$1\" not found"
    exit 1
  fi

  echo "Found swagger file: $swaggerFile"

  # Compile Swagger into RESTler dependencies
  docker run --rm -v $(realpath $swaggerRootDir):/swagger -w /swagger/.apitest/restler_output \
    mcr.microsoft.com/restlerfuzzer/restler \
    dotnet /RESTler/restler/Restler.dll compile \
    --api_spec /swagger/$(realpath --relative-to=$swaggerRootDir $swaggerFile)

  # Generate API Scenario with full operation coverage
  oav generate-api-scenario static \
    --specs $swaggerFile \
    --dependency $swaggerRootDir/.apitest/restler_output/Compile/dependencies.json \
    -o $(dirname $swaggerFile)/scenarios

}

function runApiScenario {
  local scenarioFile=${1:-basic.yaml}
  local swaggerFile=${2:-openapi.json}

  if [ ! -f scenarioFile ]; then
    scenarioFile=$(find $swaggerRootDir -type f -name "$scenarioFile")
  fi

  if [ -z $scenarioFile ]; then
    echo "Scenario file \"$1\" not found"
    exit 1
  fi

  echo "Found scenario file: $scenarioFile"

  if [ ! -f $swaggerFile ]; then
    swaggerFile=$(find $swaggerRootDir -type f -name "$swaggerFile")
  fi

  if [ -z $swaggerFile ]; then
    echo "Swagger file \"$2\" not found"
    exit 1
  fi

  echo "Found swagger file: $swaggerFile"

  local envFile=$swaggerRootDir/.apitest/env.json

  if [ ! -f $envFile ]; then
    # Create environment file
    echo -e '{\n  "tenantId": "<AAD app tenantId>",\n  "client_id": "<AAD app client_id>",\n  "client_secret": "<AAD app client_secret>",\n  "subscriptionId": "6f53185c-ea09-4fc3-9075-318dec805303",\n  "location": "westus"\n}' >$envFile
  fi

  # Run API Scenario test with local OneBox
  oav run $scenarioFile \
    --specs $swaggerFile \
    --armEndpoint $ONEBOX_ENDPOINT \
    --envFile $envFile \
    --devMode \
    --output $swaggerRootDir/.apitest \
    --generateExample \
    --logLevel verbose

}

case "$1" in
"generate")
  generateApiScenario $2
  ;;

"run")
  runApiScenario $2 $3
  ;;

*)
  echo "Usage: $0 <command> [<args>]"
  echo "Commands: generate, run"
  echo "  - generate [<swagger file>]: Generate API Scenario"
  echo "  - run [<scenario file> <swagger file>]: Run API Scenario"
  exit 1
  ;;
esac
