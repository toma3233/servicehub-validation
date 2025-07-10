@sys.description('A unique string. If changed the script will be applied again. By default, we generate a unique string to ensure the script will be applied again.')
param forceUpdateTag string = utcNow()

@sys.description('If true, continues the script execution even if the script fails.')
param continueOnErrors bool = false

@sys.description('The name of the Kusto cluster.')
param clusterName string

@sys.description('The name of the Kusto database.')
param databaseName string

@sys.description('The name of the Kusto table creation script.')
param scriptName string

@sys.description('The name of the Kusto table to be created.')
param tableName string

resource cluster 'Microsoft.Kusto/clusters@2022-02-01' existing = {
    name: clusterName
}

resource db 'Microsoft.Kusto/clusters/databases@2022-02-01' existing = {
    name: databaseName
    parent: cluster
}

resource createTableScript 'Microsoft.Kusto/clusters/databases/scripts@2022-02-01' = {
    name: scriptName
    parent: db
    properties: {
        scriptContent: '.create table ${tableName} (source: string)'
        continueOnErrors: continueOnErrors
        forceUpdateTag: forceUpdateTag
    }
}
