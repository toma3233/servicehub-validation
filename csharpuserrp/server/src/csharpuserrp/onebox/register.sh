#!/bin/bash

set -e

ONEBOX_ENDPOINT=${ONEBOX_ENDPOINT:-http://host.docker.internal:5000}
ONEBOX_SUBSCRIPTION=${ONEBOX_SUBSCRIPTION:-6f53185c-ea09-4fc3-9075-318dec805303}
ONEBOX_API_VERSION=${ONEBOX_API_VERSION:-2021-05-01-preview}

function getLocation {
  local location=${ONEBOX_ENDPOINT}/subscriptions/${ONEBOX_SUBSCRIPTION}/providers/Microsoft.ProviderHub
  local isProvider=1

  IFS=/

  for seg in $1; do
    if [ $isProvider -eq 1 ]; then
      location=${location}/providerRegistrations/$seg
      isProvider=0
    else
      location=${location}/resourcetypeRegistrations/$seg
    fi
  done

  unset IFS

  echo $location?api-version=${ONEBOX_API_VERSION}
}

registrations=${1:-./registrations}

if [ ! -d $registrations ]; then
  echo "Directory $registrations does not exist"
  exit 1
fi

for file in $(find $registrations -type f -name "*.json" | sort); do
  echo "Registering $file"
  fileName=${file#*/registrations/}
  fqrt=${fileName%.json}

  location=$(getLocation $fqrt)

  curl -v -m 1 -X PUT -H "Content-Type: application/json" -d @$file $location

  echo -e "\nSuccessfully registered: $fqrt\n"
done
