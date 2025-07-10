uuid := $(shell LC_ALL=C tr -dc 'a-z0-9' </dev/urandom | head -c 10; echo)
envConfig=../env-config.yaml
location?=westus
serviceImageTag?=latest
deletionDate := $(shell date +%Y-%m-%d -d "+3 days")

#This assumes we have an environment variable called subscriptionId
genEnvConfig:
	echo "# A string that will be used for naming all resources that are created for this project." > ${envConfig}
	echo "resourcesName: $(uuid)" >> ${envConfig}
	echo "# The subscription Id to which we deploy our resources." >> ${envConfig}
	echo "subscriptionId: $$subscriptionId" >> ${envConfig}
	echo "# The location of the resource group to which we deploy our resources." >> ${envConfig}
	echo "location: $(location)" >> ${envConfig}
	echo "# The resource group to which we deploy our resources." >> ${envConfig}
	echo "resourceGroupName: servicehubval-$(uuid)-rg" >> ${envConfig}
	echo "# The tag used to push the services' images to ACR." >> ${envConfig}
	echo "serviceImageTag: $(serviceImageTag)" >> ${envConfig}
	echo "# Flag to indicate whether to use Geneva environment." >> ${envConfig}
	echo "useGenevaMonitoring: false" >> ${envConfig}
	echo "# The date when the resources will be deleted. Users need to update this if they want to keep the resources longer than the default of 3 days." >> ${envConfig}
	echo "deletionDate: $(deletionDate)" >> ${envConfig}
