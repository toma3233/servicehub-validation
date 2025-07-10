using AKSMiddleware;
using Azure;
using Azure.ResourceManager;
using Azure.ResourceManager.Resources;
using Google.Protobuf.WellKnownTypes;
using Grpc.Core;
using MyGreeterCsharp.Api.V1;
using Serilog;

namespace MyGreeterCsharp.Server.Handler;

public partial class MyGreeterCsharpServer
{
    public override async Task<Empty> CreateResourceGroup(CreateResourceGroupRequest request, ServerCallContext context)
    {
        if (_resourceGroups == null)
        {
            Log.Logger.WithCtx(context).Warning("ResourceGroupClient is nil in CreateResourceGroup(), azuresdk feature is likely disabled");
            throw new RpcException(new Status(StatusCode.Unavailable, "ResourceGroupClient is nil, azuresdk feature is likely disabled"));
        }

        try
        {
            ResourceGroupData resourceGroupData = new ResourceGroupData(request.Region);
            ArmOperation<ResourceGroupResource> operation = await _resourceGroups.CreateOrUpdateAsync(WaitUntil.Completed, request.Name, resourceGroupData);
            ResourceGroupResource resourceGroup = operation.Value;

            Log.Logger.WithCtx(context).Information("Created resource group: {ResourceId}", resourceGroup.Id);
        }
        catch (RequestFailedException ex)
        {
            Log.Logger.WithCtx(context).Error(ex, "CreateOrUpdateAsync() error: {ErrorMessage}", ex.Message);
            throw Server.HandleError(ex, "CreateOrUpdateAsync");
        }
        catch (Exception ex)
        {
            Log.Logger.WithCtx(context).Error(ex, "An unexpected error occurred: {ErrorMessage}", ex.Message);
            throw Server.HandleError(ex, "CreateResourceGroup");
        }

        return new Empty();
    }
}
