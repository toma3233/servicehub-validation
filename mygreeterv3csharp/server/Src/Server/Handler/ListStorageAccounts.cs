using Grpc.Core;
using MyGreeterCsharp.Api.V1;

namespace MyGreeterCsharp.Server.Handler;

public partial class MyGreeterCsharpServer
{
    public override async Task<ListStorageAccountResponse> ListStorageAccounts(ListStorageAccountRequest request, ServerCallContext context)
    {
        // TODO: Implement ListStorageAccounts
        var response = new ListStorageAccountResponse();

        return await Task.FromResult(response);
    }
}
