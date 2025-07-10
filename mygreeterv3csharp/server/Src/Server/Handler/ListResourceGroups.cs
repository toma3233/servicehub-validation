using AKSMiddleware;
using Google.Protobuf.WellKnownTypes;
using Grpc.Core;
using MyGreeterCsharp.Api.V1;
using Serilog;

namespace MyGreeterCsharp.Server.Handler;

public partial class MyGreeterCsharpServer
{
    public override async Task<ListResourceGroupResponse> ListResourceGroups(Empty request, ServerCallContext context)
    {
        if (_resourceGroups == null)
        {
            Log.Logger.WithCtx(context).Warning("ResourceGroupClient is nil in ListResourceGroups(), azuresdk feature is likely disabled");
            throw new RpcException(new Status(StatusCode.Unavailable, "ResourceGroupClient is nil in ListResourceGroups(), azuresdk feature is likely disabled"));
        }

        var resourceGroupList = new List<ResourceGroup>();

        try
        {
            await foreach (var resourceGroup in _resourceGroups.GetAllAsync())
            {
                var resourceGroupProto = new ResourceGroup
                {
                    Id = resourceGroup.Id.ToString(),
                    Name = resourceGroup.Data.Name ?? string.Empty,
                    Location = resourceGroup.Data.Location
                };
                resourceGroupList.Add(resourceGroupProto);
            }

            Log.Logger.WithCtx(context).Information("Resource groups found: {Count}", resourceGroupList.Count);
        }
        catch (Exception ex)
        {
            var grpcError = Server.HandleError(ex, "ListResourceGroups");
            Log.Logger.WithCtx(context).Error(grpcError, "Error occurred during resource group listing");
            throw grpcError;
        }

        return new ListResourceGroupResponse
        {
            RgList = { resourceGroupList }
        };
    }
}
