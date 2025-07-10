DIR_NAME=shared-resources
MAIN_DIR := $(shell dirname -- $(shell pwd))
USER_ID := $(shell id -u):$(shell id -g)
#The config file used to populate resourceName and subscriptionId, default assumes there is a config file
#in the main directory of project.
ENVCONFIG?="env-config.yaml"
REGISTRY_NAME=servicehubregistry
IMG_NAME=service_hub_environment
IMG_DATE_TAG=20250228

.PHONY: template-files deploy-resources

template-files:
	@docker run --rm -u $(USER_ID) --mount src=$(MAIN_DIR),target=/app/binded-data,type=bind $(REGISTRY_NAME).azurecr.io/$(IMG_NAME):$(IMG_DATE_TAG) \
	./maintain_service executeTemplatesInFolder --templatePrefix "template-" --templatePath binded-data/$(DIR_NAME) --envConfig binded-data/$(ENVCONFIG) --leftDelim "{{" --rightDelim "}}"

deploy-resources: template-files
	echo "Starting resource provisioning."
	RESOURCES_NAME=$(shell jq -r '.parameters.resourcesName.value' resources/Main.SharedResources.Parameters.json); \
	az deployment sub create --name "servicehubval-$$RESOURCES_NAME-deploy" --location EastUS --template-file resources/Main.SharedResources.Template.bicep --parameters resources/Main.SharedResources.Parameters.json -o json > .$(DIR_NAME)_tmp.json
	@if cat .$(DIR_NAME)_tmp.json | grep '"provisioningState": "Succeeded"' > /dev/null 2>&1; then \
		echo "Resource provisioning succeeded."; \
		cp .$(DIR_NAME)_tmp.json .$(DIR_NAME)_output.json; \
	else \
		echo "Resource provisioning did not succeed."; \
		cat .$(DIR_NAME)_tmp.json; \
		exit 1; \
	fi
	@rm .$(DIR_NAME)_tmp.json
	cd generated; make connect-cluster
	@docker run --rm -u $(USER_ID) --mount src=$(MAIN_DIR),target=/app/binded-data,type=bind $(REGISTRY_NAME).azurecr.io/$(IMG_NAME):$(IMG_DATE_TAG) /bin/bash -c \
	"./maintain_service generateResourceFiles --svcDirName=binded-data/$(DIR_NAME); \
	cd binded-data/$(DIR_NAME); export KUBECONFIG=\$$(pwd)/generated/.kube/config; \
	kubectl create -f deployments/argo-rollouts-namespace.yaml; \
	kubectl apply -n argo-rollouts -f https://github.com/argoproj/argo-rollouts/releases/latest/download/install.yaml;"; \

