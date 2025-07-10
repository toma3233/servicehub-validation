using AKSMiddleware;
using Azure;
using Grpc.Core;
using Microsoft.Extensions.Diagnostics.HealthChecks;
using Serilog;
using Serilog.Core;
using Serilog.Events;
using Serilog.Templates;

namespace MyGreeterCsharp.Server.Handler;

// Some properties are automatically included in the log message, so we must remove them from the properties list
// https://stackoverflow.com/questions/47176191/how-to-remove-properties-from-log-entries-in-asp-net-core
class RemovePropertiesEnricher : ILogEventEnricher
{
    public void Enrich(LogEvent le, ILogEventPropertyFactory lepf)
    {
        le.RemovePropertyIfPresent("SourceContext");
        le.RemovePropertyIfPresent("RequestId");
        le.RemovePropertyIfPresent("RequestPath");
        le.RemovePropertyIfPresent("ConnectionId");
    }
}

public static class Server
{

    private static WebApplication? _appServer;

    public static WebApplication? AppServer => _appServer;

    // Serve configures and starts the gRPC server, including logging, interceptors, and health checks, 
    // it starts a gRPC server with defined or default port, but allows rest-like calls to be made through HTTP/2 by adding JSON transcoding.
    public static async Task Serve(ServerOptions options)
    {
        var builder = WebApplication.CreateBuilder(new WebApplicationOptions
        {
            Args = new[] { "--urls", $"http://0.0.0.0:{options.Port}" }
        });

        // Enforce HTTP/2
        // https://learn.microsoft.com/en-us/aspnet/core/fundamentals/servers/kestrel/endpoints?view=aspnetcore-8.0#configure-http-protocols-in-code
        builder.WebHost.ConfigureKestrel(serverOptions =>
        {
            serverOptions.ListenAnyIP(options.Port, listenOptions =>
            {
                listenOptions.Protocols = Microsoft.AspNetCore.Server.Kestrel.Core.HttpProtocols.Http2;
            });
        });

        // Serilog configuration; enriches with LogContext (link below), removes properties, and adds custom attributes
        var loggerConfiguration = new LoggerConfiguration()
            .Enrich.FromLogContext()
            .Enrich.With(new RemovePropertiesEnricher())
            .Enrich.With<LogAttrs.CustomAttributeEnricher>();

        // using Serilog ExpressionTemplate
        // https://github.com/serilog/serilog-expressions?tab=readme-ov-file#formatting-with-expressiontemplate
        if (options.JsonLog)
        {
            loggerConfiguration = loggerConfiguration.WriteTo.Console(new ExpressionTemplate(
                "{ {time: @t, level: if @l = 'Information' then 'INFO' else if @l = 'Error' then 'ERROR' else if @l = 'Warning' then 'WARN' else if @l = 'Debug' then 'DEBUG' else if @l = 'Verbose' then 'VERBOSE' else if @l = 'Fatal' then 'FATAL' else @l, msg: @m, EX: @x, location: @Location, ..@p} }\n"));
        }
        // https://github.com/serilog/serilog/wiki/Formatting-Output#formatting-plain-text
        else
        {
            loggerConfiguration = loggerConfiguration.WriteTo.Console(outputTemplate: "{Timestamp} [{Level}] {Message} {CustomAttributes:lj}{Properties}{NewLine}{Exception}");
        }
        Log.Logger = loggerConfiguration.CreateLogger();

        builder.Logging.ClearProviders();
        builder.Logging.AddSerilog(Log.Logger);

        // Add Serilog logger to the DI container
        // https://learn.microsoft.com/en-us/aspnet/core/fundamentals/dependency-injection?view=aspnetcore-8.0
        builder.Services.AddScoped<Serilog.ILogger>(provider =>
        {
            var logger = Log.Logger;
            return logger;
        });

        // Add the ServerOptions to the DI for access in the MyGreeterCsharpServer constructor (Api.cs)
        builder.Services.AddSingleton(options);

        // Add interceptors to the gRPC server, injected into the DI container
        // https://learn.microsoft.com/en-us/aspnet/core/grpc/interceptors?view=aspnetcore-8.0#configure-server-interceptors
        builder.Services.AddGrpc(grpcOptions =>
        {
            var serverInterceptors = InterceptorFactory.DefaultServerInterceptors(Log.Logger);
            foreach (var interceptor in serverInterceptors)
            {
                grpcOptions.Interceptors.Add(interceptor.GetType());
            }
            // Allows rest calls through http/2
            // https://learn.microsoft.com/en-us/aspnet/core/grpc/json-transcoding-openapi?view=aspnetcore-8.0
        }).AddJsonTranscoding();


        // https://learn.microsoft.com/en-us/aspnet/core/grpc/health-checks?view=aspnetcore-8.0
        builder.Services.AddGrpcHealthChecks()
                        .AddCheck("MyGreeterCsharpServer", () => HealthCheckResult.Healthy());

        var app = builder.Build();

        app.MapGrpcService<MyGreeterCsharpServer>();
        app.MapGrpcHealthChecksService();
        app.MapGet("/", () => "Communication with gRPC endpoints must be made through a gRPC client. To learn how to create a client, visit: https://go.microsoft.com/fwlink/?linkid=2086909");

        _appServer = app;

        await app.RunAsync();
    }

    public static RpcException HandleError(Exception ex, string operation)
    {
        if (ex is RequestFailedException requestFailedException)
        {
            var code = ArmPolicy.ConvertHTTPStatusToGRPCError(requestFailedException.Status);
            var grpcError = new RpcException(new Status(code, $"call error: {ex.Message}"));
            return grpcError;
        }
        else
        {
            return new RpcException(new Status(StatusCode.Unknown, $"An unexpected error occurred during {operation}: {ex.Message}"));
        }
    }
}
