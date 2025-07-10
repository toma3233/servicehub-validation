@sys.description('The kusto cluster name that the data connection will be created in.')
param clusterName string

@sys.description('The name of the data connection.')
@minLength(1)
@maxLength(12) // TODO: calculate the max length of the mds account name. This depends on the length of the cluster name and the data connection name. The max length of the mds account name is 40 characters. The mds account name is a combination of the cluster name and the data connection name.
param dataConnectionName string

@sys.description('The name of the geneva environment.')
param genevaEnvironment string

@sys.description('The list of MDS accounts to be used in the data connection.')
param mdsAccounts array

@sys.description('Whether the data connection is scrubbed or not.')
param isScrubbed bool = true // Clarify the purpose of 'isScrubbed' with the Geneva team or remove the TODO if it has been addressed.

@sys.description('The location of the data connection.')
param location string

// TODO: Note that this is a legacy data connection. We are working on migrating to the new data connection model.
// This resource can be updated, but you cannot update the genevaEnvironment property. If you need to change the Geneva environment, you must delete and recreate the data connection.
// The data connection cannot be deleted the typical means. You must manually delete the Geneva Data Connection through an HTTP request. Details are here: https://kusto.azurewebsites.net/docs/kusto/ops/manage-geneva-dataconnections.html#create-or-update-geneva-data-connection
resource dataConnection 'Microsoft.Kusto/clusters/dataconnections@2019-11-09' = {
  name: concat(clusterName, '/', dataConnectionName) // Name of the Geneva data connection. The name of your data connection. Data connection names can contain only alphanumeric, dash and dot characters and can be up to 40 characters in length. The name must be unique at the cluster level."
  location: location
  kind: 'GenevaLegacy'
  properties: {
    genevaEnvironment: genevaEnvironment // If you decide to use a different Geneva environment, you must delete and recreate the data connection, or change the name of the data connection.
    mdsAccounts: mdsAccounts
    isScrubbed: isScrubbed
  }

}
