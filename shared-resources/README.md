# Provision Shared Resources

## Create or Update Shared Resources

```bash
make deploy-resources
```

[Optional] Should you want to modify the parameter values, change `resources/Main.SharedResources.Template.bicep` and run `make deploy-resources`. Follow the instructions in [Making changes to Bicep Resources](../README/README.md).

## View All Resources and Dependencies

After you have run `make deploy-resources`, file [shared-resources_resources.md](shared-resources_resources.md) will be generated. It provides a high-level overview of all your deployments.

To see the resources and their dependencies, click the different links in the file. Each link is a different markdown file that is associated with a bicep deployment. Each bicep deployment associated file has:

- list of resources you have created via bicep file
- links to the resources in Azure portal
- the dependencies of each resource

## Argo Controller Installation Information

As a part of our shared-resources deployment, we perform two steps required for the Argo Rollouts - Kubernetes Progressive Delivery Controller's Installation.

1. Create namespace using [argo-rollouts-namespace.yaml](deployments/argo-rollouts-namespace.yaml)
2. Install the controller through an externally hosted yaml file.

How is it done in the two deployments.

- Regular deployment: in the above command to deploy-resources, after shared-resources deployment is successful, we connect to the aks managed cluster and perform the steps using kubectl commands in our created "environment" docker container to avoid performing unnecessary installations.
- EV2 deployment: We use Kubectl application modeling to perform the steps as a part of the rollout.

More information and manual installation steps can be found [here](https://argoproj.github.io/argo-rollouts/installation/)
