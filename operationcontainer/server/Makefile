
REGISTRY_NAME=servicehubregistry
IMG_NAME=service_hub_environment
IMG_DATE_TAG=20250228
MAIN_DIR := $(shell dirname -- $(shell dirname -- $(shell pwd)))
DIR_NAME=operationcontainer
#The config file used to populate resourceName and subscriptionId, default assumes there is a config file
#in the main directory of project.
ENVCONFIG?="env-config.yaml"

USER_ID := $(shell id -u):$(shell id -g)
GOPATH := $(shell go env GOPATH)
all: tidy test build alert-files

template-files:
	@if [ -f ../../$(ENVCONFIG) ]; then \
		docker run --mount src=$(MAIN_DIR),target=/app/binded-data,type=bind $(REGISTRY_NAME).azurecr.io/$(IMG_NAME):$(IMG_DATE_TAG) \
		./maintain_service executeTemplatesInFolder --templatePrefix "template-" --templatePath binded-data/$(DIR_NAME) --envConfig binded-data/$(ENVCONFIG) --leftDelim "{{" --rightDelim "}}"; \
		echo "Templating server files was successful"; \
	else \
		echo "../../$(ENVCONFIG) does not exist. Please make sure either the path exists or rerun the command with the flag --ENVCONFIG 'pathtoconfigfile'"; \
		exit 1; \
	fi

# (Prerequisite) To deploy the shared resources needed, go to the directory of the shared resources and run the Makefile.
# Creates all the service specific resources and generates markdown files
# TODO: Investigate how to express dependency across different directories rather than write imperative code.
deploy-resources: template-files
	cd resources; make all
	docker run --mount src=$(MAIN_DIR),target=/app/binded-data,type=bind $(REGISTRY_NAME).azurecr.io/$(IMG_NAME):$(IMG_DATE_TAG) ./maintain_service generateResourceFiles --svcDirName=binded-data/$(DIR_NAME); \

tidy: template-files
	go mod tidy

test:
	go test ./...

build: template-files
	go build -o bin/ dev.azure.com/service-hub-flg/service_hub_validation/_git/service_hub_validation_service.git/operationcontainer/server/cmd/server
	
build-image: template-files
	cd generated; make build-image

build-multiarch-image: template-files
	cd generated; make build-multiarch-image

push-image: template-files
	cd generated; make push-image

alert-files:
	docker run --mount src=$(MAIN_DIR),target=/app/binded-data,type=bind \
		-e GITHUB_TOKEN=$(GITHUB_TOKEN) \
		$(REGISTRY_NAME).azurecr.io/$(IMG_NAME):$(IMG_DATE_TAG) \
		/bin/bash -c \
	"cd /app; ./maintain_service populateMethodFiles \
		--useRemoteProto true \
		--owner Azure \
		--repo OperationContainer \
		--protoFilePath api/v1/proto/operationcontainer.proto \
		--svcDirName binded-data/operationcontainer --deleteGarbageFiles=false;"

AZURESDK_PATH := artifacts/.Azuresdk_properties_outputs.yaml

# The install and uninstall information (k8s secret) is stored in the current namespace.
install: template-files
	cd generated; make connect-cluster
	if [ -f $(AZURESDK_PATH) ]; then \
		docker run --mount src=$(MAIN_DIR),target=/app/binded-data,type=bind $(REGISTRY_NAME).azurecr.io/$(IMG_NAME):$(IMG_DATE_TAG) /bin/bash -c \
		"cd binded-data/$(DIR_NAME)/server; export KUBECONFIG=\$$(pwd)/generated/.kube/config; \
		cd deployments; helm install servicehubval-operationcontainer-server . -f values-common.yaml -f values-server.yaml -f ../$(AZURESDK_PATH);"; \
	else \
		echo ".Azuresdk_properties_outputs.yaml file does not exist. Run the make deploy-resources cmd"; \
		docker run --mount src=$(MAIN_DIR),target=/app/binded-data,type=bind $(REGISTRY_NAME).azurecr.io/$(IMG_NAME):$(IMG_DATE_TAG) /bin/bash -c \
		"cd binded-data/$(DIR_NAME)/server; export KUBECONFIG=\$$(pwd)/generated/.kube/config; \
		cd deployments; helm install servicehubval-operationcontainer-server . -f values-common.yaml -f values-server.yaml;"; \
	fi

upgrade: template-files
	cd generated; make connect-cluster
	if [ -f $(AZURESDK_PATH) ]; then \
		docker run --mount src=$(MAIN_DIR),target=/app/binded-data,type=bind $(REGISTRY_NAME).azurecr.io/$(IMG_NAME):$(IMG_DATE_TAG) /bin/bash -c \
		"cd binded-data/$(DIR_NAME)/server; export KUBECONFIG=\$$(pwd)/generated/.kube/config; \
		cd deployments; helm upgrade servicehubval-operationcontainer-server . -f values-common.yaml -f values-server.yaml -f ../$(AZURESDK_PATH);"; \
	else \
		echo ".Azuresdk_properties_outputs.yaml file does not exist. Run the make deploy-resources cmd"; \
		docker run --mount src=$(MAIN_DIR),target=/app/binded-data,type=bind $(REGISTRY_NAME).azurecr.io/$(IMG_NAME):$(IMG_DATE_TAG) /bin/bash -c \
		"cd binded-data/$(DIR_NAME)/server; export KUBECONFIG=\$$(pwd)/generated/.kube/config; \
		cd deployments; helm upgrade servicehubval-operationcontainer-server . -f values-common.yaml -f values-server.yaml;"; \
	fi

uninstall: template-files
	docker run --mount src=$(MAIN_DIR),target=/app/binded-data,type=bind $(REGISTRY_NAME).azurecr.io/$(IMG_NAME):$(IMG_DATE_TAG) /bin/bash -c \
	"cd binded-data/$(DIR_NAME)/server; export KUBECONFIG=\$$(pwd)/generated/.kube/config; \
	helm uninstall servicehubval-operationcontainer-server;";

