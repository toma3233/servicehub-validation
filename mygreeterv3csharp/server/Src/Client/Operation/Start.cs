using System.CommandLine;
using System.CommandLine.NamingConventionBinder;
using Google.Protobuf.WellKnownTypes;
using Grpc.Core;
using MyGreeterCsharp.Api.V1;
using Serilog;
using Serilog.Templates;

namespace MyGreeterCsharp.Client.Operation;

public class ClientOptions
{
    public string? RemoteAddr { get; set; }
    public string? HttpAddr { get; set; }
    public bool JsonLog { get; set; }
    public string? Name { get; set; }
    public int Age { get; set; }
    public string? Email { get; set; }
    public string? Address { get; set; }
    public long IntervalMilliSec { get; set; }
    public string? RgName { get; set; }
    public string? RgRegion { get; set; }
    public bool CallAllRgOps { get; set; }
}


public static class StartCommand
{
    public static Command Init()
    {
        var remoteAddrOption = new Option<string>(
            "--remote-addr",
            description: "The remote server's address for this client to connect to",
            getDefaultValue: () => "localhost:50051");

        var httpAddrOption = new Option<string>(
            "--http-addr",
            description: "The remote HTTP gateway address",
            getDefaultValue: () => "http://localhost:50061");

        var jsonLogOption = new Option<bool>(
            "--json-log",
            description: "Enables JSON format for logs (human readable key-value pairs)",
            getDefaultValue: () => false);

        var nameOption = new Option<string>(
            "--name",
            description: "The name to send in Hello request",
            getDefaultValue: () => "MyName");

        var ageOption = new Option<int>(
            "--age",
            description: "The age to send in Hello request",
            getDefaultValue: () => 53);

        var emailOption = new Option<string>(
            "--email",
            description: "The email to send in Hello request",
            getDefaultValue: () => "test@test.com");

        var addressOption = new Option<string>(
            "--address",
            description: "The address to send in Hello request",
            getDefaultValue: () => "123 Main St, Seattle, WA 98101");

        var intervalMilliSecOption = new Option<long>(
            "--interval-milli-sec",
            description: "The interval between two requests. Negative numbers mean sending one request.",
            getDefaultValue: () => -1);

        var rgNameOption = new Option<string>(
            "--rg-name",
            description: "The name of the resource group",
            getDefaultValue: () => "MyGreeterCsharp-resource-group");

        var rgRegionOption = new Option<string>(
            "--rg-region",
            description: "The region of the resource group",
            getDefaultValue: () => "eastus");

        var callAllRgOpsOption = new Option<bool>(
            "--call-all-rg-ops",
            description: "Call all resource group operations",
            getDefaultValue: () => true);

        var startCommand = new Command("hello", "Call SayHello")
        {
            remoteAddrOption,
            httpAddrOption,
            jsonLogOption,
            nameOption,
            ageOption,
            emailOption,
            addressOption,
            intervalMilliSecOption,
            rgNameOption,
            rgRegionOption,
            callAllRgOpsOption
        };

        startCommand.Handler = CommandHandler.Create<ClientOptions>(hello);

        return startCommand;
    }

    // hello is a client function that configures logging, creates a new client, and calls the SayHello function
    public static async Task hello(ClientOptions options)
    {

        // Serilog configuration
        var loggerConfiguration = new LoggerConfiguration()
            .Enrich.FromLogContext()
            .Enrich.With<LogAttrs.CustomAttributeEnricher>();

        if (options.JsonLog)
        {
            loggerConfiguration = loggerConfiguration.WriteTo.Console(new ExpressionTemplate(
                "{ {time: @t, level: if @l = 'Information' then 'INFO' else if @l = 'Error' then 'ERROR' else if @l = 'Warning' then 'WARN' else if @l = 'Debug' then 'DEBUG' else if @l = 'Verbose' then 'VERBOSE' else if @l = 'Fatal' then 'FATAL' else @l, msg: @m, EX: @x, location: @Location, ..@p} }\n"));
        }
        else
        {
            loggerConfiguration = loggerConfiguration.WriteTo.Console(outputTemplate: "{Timestamp} [{Level}] {Message} {CustomAttributes:lj}{Properties}{NewLine}{Exception}");
        }

        Log.Logger = loggerConfiguration.CreateLogger();

        var client = ClientFactory.NewClient(options.RemoteAddr!, Log.Logger);

        if (options.IntervalMilliSec < 0)
        {
            await SayHello(client, options.Name!, options.Age!, options.Email!, options.Address!, options);
        }
        else
        {
            while (true)
            {
                await SayHello(client, options.Name!, options.Age!, options.Email!, options.Address!, options);
                await Task.Delay((int)options.IntervalMilliSec!);
            }
        }
    }

    /// <summary>
    /// Sends a greeting request to the gRPC server and performs various resource group operations.
    /// </summary>
    /// <param name="client">The gRPC client used to send requests.</param>
    /// <param name="name">The name of the user to greet.</param>
    /// <param name="age">The age of the user to greet.</param>
    /// <param name="email">The email of the user to greet.</param>
    /// <param name="address">The address of the user to greet, in the format "Street, City, State Zip".</param>
    /// <param name="options">The client options containing resource group information.</param>
    /// <returns>A task representing the asynchronous operation.</returns>
    /// <remarks>
    /// This method performs the following operations:
    /// <list type="bullet">
    /// <item><description>Sends a greeting request to the gRPC server.</description></item>
    /// <item><description>Creates a resource group.</description></item>
    /// <item><description>Lists resource groups.</description></item>
    /// <item><description>Reads a resource group.</description></item>
    /// <item><description>Updates a resource group.</description></item>
    /// <item><description>Deletes a resource group.</description></item>
    /// </list>
    /// </remarks>    
    private static async Task SayHello(MyGreeterCsharp.Api.V1.MyGreeterCsharp.MyGreeterCsharpClient client, string name, int age, string email, string address, ClientOptions options)
    {
        if (address.Split(',').Length != 3 || address.Split(',')[2].Trim().Split(' ').Length != 2)
        {
            throw new ArgumentException("Address format is incorrect. Expected format: 'Street, City, State Zip'.");
        }

        string[] addressParts = address.Split(',');
        string street = addressParts[0].Trim();
        string city = addressParts[1].Trim();
        string[] stateAndZip = addressParts[2].Trim().Split(' ');
        string state = stateAndZip[0];
        string zipString = stateAndZip[1];
        if (!int.TryParse(zipString, out int zipCode))
        {
            throw new ArgumentException("Invalid zip code format.");
        }

        var addr = new Address
        {
            Street = street,
            City = city,
            State = state,
            Zipcode = zipCode
        };

        var helloRequest = new HelloRequest
        {
            Name = name,
            Age = age,
            Email = email,
            Address = addr
        };

        try
        {
            var reply = await client.SayHelloAsync(helloRequest);
            Log.Information("Greeting: {Message}", reply.Message);
        }
        catch (RpcException ex)
        {
            Log.Error($"gRPC Error calling SayHello: {ex.Status.Detail}");
        }
        catch (Exception ex)
        {
            Log.Error("Error: {Message}", ex.Message);
        }

        try
        {
            var reply = await client.CreateResourceGroupAsync(new CreateResourceGroupRequest
            {
                Name = options.RgName,
                Region = options.RgRegion
            });
        }
        catch (RpcException ex)
        {
            Log.Error($"gRPC Error calling CreateResourceGroup: {ex.Status.Detail}");
        }
        catch (Exception ex)
        {
            Log.Error("Error: {Message}", ex.Message);
        }

        try
        {
            var response = await client.ListResourceGroupsAsync(new Empty());
        }
        catch (RpcException ex)
        {
            Log.Error($"gRPC Error calling ListResourceGroup: {ex.Status.Detail}");
        }
        catch (Exception ex)
        {
            Log.Error($"Error calling ListResourceGroup: {ex.Message}");
        }

        try
        {
            var response = await client.ReadResourceGroupAsync(new ReadResourceGroupRequest
            {
                Name = options.RgName
            });
        }
        catch (RpcException ex)
        {
            Log.Error($"gRPC Error calling ReadResourceGroup: {ex.Status.Detail}");
        }
        catch (Exception ex)
        {
            Log.Error($"Error calling ReadResourceGroup: {ex.Message}");
        }

        try
        {
            var tags = new Dictionary<string, string>
            {
                { "key1", "value1" },
                { "key2", "value2" }
            };

            var response = await client.UpdateResourceGroupAsync(new UpdateResourceGroupRequest
            {
                Name = options.RgName,
                Tags = { tags }
            });
        }
        catch (RpcException ex)
        {
            Log.Error($"gRPC Error calling UpdateResourceGroup: {ex.Status.Detail}");
        }
        catch (Exception ex)
        {
            Log.Error($"Error calling UpdateResourceGroup: {ex.Message}");
        }

        try
        {
            var response = await client.DeleteResourceGroupAsync(new DeleteResourceGroupRequest
            {
                Name = options.RgName
            });
        }
        catch (RpcException ex)
        {
            Log.Error($"gRPC Error calling DeleteResourceGroup: {ex.Status.Detail}");
        }
        catch (Exception ex)
        {
            Log.Error($"Error calling DeleteResourceGroup: {ex.Message}");
        }

        // TODO: Add rest API calls when RestSDK is implemented 

        // try
        // {
        //     var createReply = await client.CreateStorageAccountAsync(new CreateStorageAccountRequest
        //     {
        //         RgName = options.RgName,
        //         Region = options.RgRegion
        //     });
        // }
        // catch (RpcException ex)
        // {
        //     Log.Error($"gRPC Error calling CreateStorageAccount: {ex.Status.Detail}");
        // }
        // catch (Exception ex)
        // {
        //     Log.Error("Error: {Message}", ex.Message);
        // }

        // try
        // {
        //     var readReply = await client.ReadStorageAccountAsync(new ReadStorageAccountRequest
        //     {
        //         RgName = options.RgName,
        //         SaName = "StorageName"
        //     });
        // }
        // catch (RpcException ex)
        // {
        //     Log.Error($"gRPC Error calling ReadStorageAccount: {ex.Status.Detail}");
        // }
        // catch (Exception ex)
        // {
        //     Log.Error("Error: {Message}", ex.Message);
        // }

        // try
        // {
        //     await client.DeleteStorageAccountAsync(new DeleteStorageAccountRequest
        //     {
        //         RgName = options.RgName,
        //         SaName = "StorageName"
        //     });
        // }
        // catch (RpcException ex)
        // {
        //     Log.Error($"gRPC Error calling DeleteStorageAccount: {ex.Status.Detail}");
        // }
        // catch (Exception ex)
        // {
        //     Log.Error("Error: {Message}", ex.Message);
        // }

        // try
        // {
        //     var updateReply = await client.UpdateStorageAccountAsync(new UpdateStorageAccountRequest
        //     {
        //         RgName = options.RgName,
        //         SaName = "StorageName",
        //         Tags = { { "tag1", "value1" }, { "tag2", "value2" } }
        //     });
        // }
        // catch (RpcException ex)
        // {
        //     Log.Error($"gRPC Error calling UpdateStorageAccount: {ex.Status.Detail}");
        // }
        // catch (Exception ex)
        // {
        //     Log.Error("Error: {Message}", ex.Message);
        // }

        // try
        // {
        //     var listReply = await client.ListStorageAccountsAsync(new ListStorageAccountRequest
        //     {
        //         RgName = options.RgName
        //     });
        // }
        // catch (RpcException ex)
        // {
        //     Log.Error($"gRPC Error calling ListStorageAccounts: {ex.Status.Detail}");
        // }
        // catch (Exception ex)
        // {
        //     Log.Error("Error: {Message}", ex.Message);
        // }
    }
}
