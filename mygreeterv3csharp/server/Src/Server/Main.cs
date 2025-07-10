using System.CommandLine;
using MyGreeterCsharp.Server.Handler;

namespace MyGreeterCsharp.Server;

public static class ServerMainCommand
{
    public static async Task<int> Main(string[] args)
    {
        var rootCommand = new RootCommand("This sample service demonstrates client-server communication using gRPC and shows how to access and interact with the Azure SDK");
        rootCommand.AddCommand(StartCommand.Init());
        return await rootCommand.InvokeAsync(args);
    }
}
