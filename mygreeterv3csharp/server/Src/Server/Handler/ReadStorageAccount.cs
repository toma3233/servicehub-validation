using Grpc.Core;
using MyGreeterCsharp.Api.V1;

namespace MyGreeterCsharp.Server.Handler;

public partial class MyGreeterCsharpServer
{
    public override async Task<ReadStorageAccountResponse> ReadStorageAccount(ReadStorageAccountRequest request, ServerCallContext context)
    {
        // TODO: Implement ReadStorageAccount
        var response = new ReadStorageAccountResponse
        {
            StorageAccount = null
        };

        return await Task.FromResult(response);
    }
}
