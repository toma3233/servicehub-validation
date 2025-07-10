using Google.Protobuf.WellKnownTypes;
using Grpc.Core;
using MyGreeterCsharp.Api.V1;

namespace MyGreeterCsharp.Server.Handler;

public partial class MyGreeterCsharpServer
{
    public override async Task<Empty> DeleteStorageAccount(DeleteStorageAccountRequest request, ServerCallContext context)
    {
        // TODO: Implement DeleteStorageAccount
        var response = new Empty();

        return await Task.FromResult(response);
    }
}
