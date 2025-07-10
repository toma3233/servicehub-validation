#!/bin/bash -x

# Execute this file under the service directory.
# For the server

SERVICEHUB_AKSMIDDLEWARE_VERSION="0.0.40"

onlyApi=$1
if [ -z "$onlyApi" ]
then
    onlyApi=false
fi
if [ "$onlyApi" = "true" ]
then
    echo "No api module to initialize, exiting early."
    exit 0
fi
cd server
make alert-files
if [ $? -ne 0 ]
then
    echo "Make alert-files failed."
    exit 1
fi
go mod init dev.azure.com/service-hub-flg/service_hub_validation/_git/service_hub_validation_service.git/operationcontainer/server
go mod edit -require github.com/Azure/OperationContainer/api@v0.0.12
go mod edit -require github.com/Azure/aks-middleware@v$SERVICEHUB_AKSMIDDLEWARE_VERSION
go mod edit -require github.com/golang-jwt/jwt/v5@v5.2.2
go get google.golang.org/genproto@latest
go mod tidy
# The following command must be run AFTER go mod tidy. If ran before, building the server module
# will fail as go mod tidy removes the indirect dependency with google.golang.org/genproto
# and go work sum will pull in an older version that causes an ambiguous import error.
# For more information refer to: https://github.com/googleapis/go-genproto/issues/1015
go get google.golang.org/genproto@latest
cd ..

go work init
go work use ./server

cd server
go build ./...
if [ $? -ne 0 ]
then
    echo "Building the server module failed."
    echo "If downloading the server module failed, you might have to wait for the api module to be available or the tag to settle then rerun again"
    exit 1
fi
go test -tags=testcontainers ./...
if [ $? -ne 0 ]
then
    echo "Testing the server module failed."
    exit 1
fi
gofmt -s -w .
cd ..

cat << EOM

After the service can run properly on your local machine, you can use the commands in
the Makefile in this directory to run the service on your standalone.

!!! Rename/delete your go.work file as aksbuilder doesn't work with go.work.

Remember to commit your modules to the repo and use the right version of the module.
Local changes won't be used by aksbuilder.

EOM
