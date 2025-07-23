variable "org_url" {
  description = "The Azure DevOps organization URL"
  default     = "https://dev.azure.com/service-hub-flg/"
}

variable "project_name" {
  description = "The Azure DevOps project name"
  default     = "service_hub_validation"
}

variable "repo_name" {
  description = "The Azure DevOps repository name"
  default     = "service_hub_validation_service"
}

variable "pipeline_path" {
  #This variable is extracted automatically from the config files and added to environment in the user environment creation script.
  description = "The path to the pipeline yaml file"
  default     = "pipeline-files/testServiceResourceAndCode.yaml"
}

variable "application_name" {
  description = "The application name for aad app"
  default =  "ServiceHubValidation_App"
}

variable "service_tree_id" {
  description = "The service id."
  // Aks Service hub validation service tree id.
  default = "dfe3a810-861c-44da-9b00-71e57a865194"
}

variable "serviceConnectionName" {
  description = "The service endpoint name used for service connection creation. Will be created at the subscription scope and require owner permissions"
  default = "ServiceHubValidation_ServiceConnection"
}

// If you want to set up a PAT rotator for your key vault 
// refer to: https://eng.ms/docs/experiences-devices/opg/office-es365/idc-fundamentals-security/oe-secret-rotator/secret-rotator-tool/guide
variable "pat_key_vault_name" {
  description = "The name of the key vault that stores the PATs"
  default = "servicehubkv"
}

variable "key_vault_rg_name" {
  description = "The name of the resource group that stores the key vault"
  default = "servicehubRg"
}

variable "key_vault_subscription_id" {
  description = "The subscription id that stores the key vault. If blank, uses the current subscription you are logged in with."
  // Subscription id of AKS Long Running things
  default = "359833f5-8592-40b6-8175-edc664e2196a"
}

variable "subscription_id" {
  description = "The subscription id used to create all the resources"
  // Subscription id of Azure Container Service - Test (AKS Standalone)
  default = "8ecadfc9-d1a3-4ea4-b844-0d9f87e4d7c8"
}
