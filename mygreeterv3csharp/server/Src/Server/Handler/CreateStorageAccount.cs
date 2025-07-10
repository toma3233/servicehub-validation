using Grpc.Core;
using MyGreeterCsharp.Api.V1;

namespace MyGreeterCsharp.Server.Handler;

public partial class MyGreeterCsharpServer
{
    public override async Task<CreateStorageAccountResponse> CreateStorageAccount(CreateStorageAccountRequest request, ServerCallContext context)
    {
        // TODO: implement CreateStorageAccount
        var response = new CreateStorageAccountResponse
        {
            Name = "StorageName"
        };

        return await Task.FromResult(response);
    }
}
