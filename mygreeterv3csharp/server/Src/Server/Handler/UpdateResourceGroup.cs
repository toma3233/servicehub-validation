using AKSMiddleware;
using Azure;
using Grpc.Core;
using MyGreeterCsharp.Api.V1;
using Serilog;

namespace MyGreeterCsharp.Server.Handler;

public partial class MyGreeterCsharpServer
{
    public override async Task<UpdateResourceGroupResponse> UpdateResourceGroup(UpdateResourceGroupRequest request, ServerCallContext context)
    {
        var logger = Log.Logger.WithCtx(context);

        if (_resourceGroups == null)
        {
            logger.Error("ResourceGroupClient is nil in UpdateResourceGroup(), azuresdk feature is likely disabled");
            throw new RpcException(new Status(StatusCode.Unavailable, "ResourceGroupClient is nil, azuresdk feature is likely disabled"));
        }

        try
        {
            // Read the existing resource group
            var existingResourceGroupResponse = await _resourceGroups.GetAsync(request.Name);
            var existingResourceGroup = existingResourceGroupResponse.Value;

            // Loop through and update each tag
            // The tags field will not be null. Ref: https://protobuf.dev/programming-guides/proto3/#default
            foreach (var tag in request.Tags)
            {
                await existingResourceGroup.AddTagAsync(tag.Key, tag.Value);
            }

            // Return the existing resource group in the response
            var response = new UpdateResourceGroupResponse
            {
                ResourceGroup = new ResourceGroup
                {
                    Id = existingResourceGroup.Id?.ToString() ?? string.Empty,
                    Name = existingResourceGroup.Data.Name ?? string.Empty,
                    Location = existingResourceGroup.Data.Location
                }
            };

            logger.Information("Updated resource group: {ResourceName}", response.ResourceGroup.Name);

            return response;
        }
        catch (RequestFailedException ex)
        {
            logger.Error(ex, "AddTagAsync() error: {ErrorMessage}", ex.Message);
            throw Server.HandleError(ex, "AddTagAsync");
        }
        catch (Exception ex)
        {
            logger.Error(ex, "An unexpected error occurred: {ErrorMessage}", ex.Message);
            throw Server.HandleError(ex, "UpdateResourceGroup");
        }
    }
}

