using AKSMiddleware;
using Azure;
using Grpc.Core;
using MyGreeterCsharp.Api.V1;
using Serilog;

namespace MyGreeterCsharp.Server.Handler;

public partial class MyGreeterCsharpServer
{
    public override async Task<ReadResourceGroupResponse> ReadResourceGroup(ReadResourceGroupRequest request, ServerCallContext context)
    {
        if (_resourceGroups == null)
        {
            Log.Logger.WithCtx(context).Warning("ResourceGroupClient is nil in ReadResourceGroup(), azuresdk feature is likely disabled");
            throw new RpcException(new Status(StatusCode.Unavailable, "ResourceGroupClient is nil in ReadResourceGroup(), azuresdk feature is likely disabled"));
        }

        try
        {
            var resourceGroup = await _resourceGroups.GetAsync(request.Name);

            // This should never happen because GetAsync method should never return null, 
            // but just in case we check for it here and make the subsequuent operations cleaner.   
            if (resourceGroup == null)
            {
                Log.Logger.WithCtx(context).Warning("Resource group not found: {ResourceName}", request.Name);
                throw new RpcException(new Status(StatusCode.Internal, $"Internal error while getting resource group '{request.Name}'"));
            }

            var readResourceGroup = new ResourceGroup
            {
                Id = resourceGroup.Value.Data.Id?.ToString() ?? string.Empty,
                Name = resourceGroup.Value.Data.Name ?? string.Empty,
                Location = resourceGroup.Value.Data.Location
            };

            Log.Logger.WithCtx(context).Information("Read resource group: {ResourceName} in {Location}", readResourceGroup.Name, readResourceGroup.Location);

            return new ReadResourceGroupResponse { ResourceGroup = readResourceGroup };
        }
        catch (RequestFailedException ex)
        {
            Log.Logger.WithCtx(context).Error(ex, "GetAsync() error: {ErrorMessage}", ex.Message);
            throw Server.HandleError(ex, "GetAsync");
        }
        catch (Exception ex)
        {
            Log.Logger.WithCtx(context).Error(ex, "An unexpected error occurred: {ErrorMessage}", ex.Message);
            throw Server.HandleError(ex, "ReadResourceGroup");
        }
    }
}
