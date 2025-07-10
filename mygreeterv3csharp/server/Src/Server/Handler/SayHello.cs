using AKSMiddleware;
using Grpc.Core;
using MyGreeterCsharp.Api.V1;
using Newtonsoft.Json;
using Serilog;

namespace MyGreeterCsharp.Server.Handler;

public partial class MyGreeterCsharpServer
{
    // Unlike all other operations, SayHello is a synchronized operation that echoes back the request information,
    // although wrapped in Task.FromResult to conform to the asynchronous pattern expected by gRPC. 
    public override Task<HelloReply> SayHello(HelloRequest request, ServerCallContext context)
    {
        try
        {
            string reqJson = JsonConvert.SerializeObject(request);
            Log.Logger.WithCtx(context).Information($"API handler logger output. req: {reqJson}");
        }
        catch (Exception ex)
        {
            Log.Logger.WithCtx(context).Error($"Error serializing request: {ex}");
        }

        // TODO: Add a call to demo server
        return Task.FromResult(new HelloReply
        {
            Message = "Echo back what you sent me (SayHello): " + request.Name + " " + request.Age.ToString() + " " + request.Email
        });
    }
}
