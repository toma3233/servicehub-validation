using Grpc.Core;
using MyGreeterCsharp.Api.V1;

namespace MyGreeterCsharp.Server.Handler;

public partial class MyGreeterCsharpServer
{
    public override async Task<UpdateStorageAccountResponse> UpdateStorageAccount(UpdateStorageAccountRequest request, ServerCallContext context)
    {
        // TODO:: Implement UpdateStorageAccount
        var response = new UpdateStorageAccountResponse
        {
            StorageAccount = null
        };

        return await Task.FromResult(response);
    }
}
