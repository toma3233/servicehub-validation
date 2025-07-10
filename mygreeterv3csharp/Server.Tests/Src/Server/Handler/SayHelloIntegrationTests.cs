using Microsoft.AspNetCore.Builder;
using MyGreeterCsharp.Api.V1;
using MyGreeterCsharp.Server;
using Serilog;
using Serilog.Formatting.Json;

namespace Server.Tests;

[Collection("Sequential")]
public class SayHelloIntegrationTests
{
    private readonly Serilog.ILogger _logger;
    private readonly int _port;

    private readonly string _address;

    private readonly StringWriter _logBuffer;
    private readonly WebApplication? _app;

    public SayHelloIntegrationTests()
    {
        // Start the server
        var args = new string[] { "start", "--port=0" };
        ServerMainCommand.Main(args);

        // Wait for the server to start
        _app = MyGreeterCsharp.Server.Handler.Server.AppServer;
        for (int i = 0; i < 20; i++)
        {
            if (_app != null)
            {
                break;
            }
            else
            {
                Task.Delay(500).Wait();
            }
        }
        _port = GetPort(_app);
        _address = $"localhost:{_port}";

        // Create a StringWriter to capture the logs
        _logBuffer = new StringWriter();
        _logger = new LoggerConfiguration()
            .MinimumLevel.Debug()
            .WriteTo.Console(new JsonFormatter())
            .WriteTo.TextWriter(_logBuffer)
            .CreateLogger();
    }

    [Fact]
    public async Task ShouldReturnSuccessWhenMakingSayHelloCall()
    {
        var client = ClientFactory.NewClient(_address, _logger);
        var request = new HelloRequest { Name = "MyName", Age = 53, Email = "test@test.com" };
        var response = await client.SayHelloAsync(request);
        Assert.Equal("Echo back what you sent me (SayHello): MyName 53 test@test.com", response.Message);

        // Assert the logs captured in the buffer
        var logs = _logBuffer.ToString();
        Assert.Contains("SayHello", logs);
        Assert.Contains("Finished gRPC call", logs);

        await CleanUp();
    }

    [Fact]
    public async Task ShouldReturnErrorWhenNameIsTooShort()
    {
        var client = ClientFactory.NewClient(_address, _logger);
        var request = new HelloRequest { Name = "Z", Age = 53, Email = "test@test.com" };
        var ex = await Assert.ThrowsAsync<AggregateException>(async () => await client.SayHelloAsync(request));
        Assert.Contains("value length must be at least 2 characters", ex.Message);

        // Assert the logs captured in the buffer
        var logs = _logBuffer.ToString();
        Assert.Contains("Call failed with gRPC error status", logs);
        Assert.Contains("value length must be at least 2 characters", logs);

        await CleanUp();
    }

    [Fact]
    public async Task ShouldReturnErrorWhenAgeIsTooBig()
    {
        var client = ClientFactory.NewClient(_address, _logger);
        var request = new HelloRequest { Name = "MyName", Age = 530, Email = "test@test.com" };
        var expectedErrorMessage = "value must be greater than or equal to 1 and less than 150";
        var ex = await Assert.ThrowsAsync<AggregateException>(async () => await client.SayHelloAsync(request));
        Assert.Contains(expectedErrorMessage, ex.Message);

        // Assert the logs captured in the buffer
        var logs = _logBuffer.ToString();
        Assert.Contains("Call failed with gRPC error status", logs);
        Assert.Contains(expectedErrorMessage, logs);

        await CleanUp();
    }

    [Fact]
    public async Task ShouldReturnErrorWhenEmailIsInvalid()
    {
        var client = ClientFactory.NewClient(_address, _logger);
        var request = new HelloRequest { Name = "MyName", Age = 53, Email = "test" };
        var expectedErrorMessage = "value does not match regex pattern";
        var ex = await Assert.ThrowsAsync<AggregateException>(async () => await client.SayHelloAsync(request));
        Assert.Contains(expectedErrorMessage, ex.Message);

        // Assert the logs captured in the buffer
        var logs = _logBuffer.ToString();
        Assert.Contains("Call failed with gRPC error status", logs);
        Assert.Contains(expectedErrorMessage, logs);

        await CleanUp();
    }

    [Fact]
    public async Task ShouldReturnErrorWhenServerIsUnavailable()
    {
        var client = ClientFactory.NewClient($"localhost:0", _logger);
        var request = new HelloRequest { Name = "MyName", Age = 53, Email = "test" };
        var expectedErrorMessage = "error";
        var ex = await Assert.ThrowsAsync<AggregateException>(async () => await client.SayHelloAsync(request));
        Assert.Contains(expectedErrorMessage, ex.Message);

        // Assert the logs captured in the buffer
        var logs = _logBuffer.ToString();
        Assert.Contains("Call failed with gRPC error status", logs);
        Assert.Contains(expectedErrorMessage, logs);

        // Assert that retry occurred
        var occurrences =
            logs.Split(new[] { "Call failed with gRPC error status" }, StringSplitOptions.None).Length - 1;
        Assert.True(occurrences > 1, $"Expected retris, but found only {occurrences}");

        await CleanUp();
    }

    private int GetPort(WebApplication? app)
    {
        if (app != null)
        {
            var boundUrl = app.Urls.FirstOrDefault(); // Get the first bound URL
            if (boundUrl != null)
            {
                var uri = new Uri(boundUrl);
                int port = uri.Port;
                return port;
            }
        }
        throw new InvalidOperationException("Unable to retrieve the port from the server.");
    }

    private async Task CleanUp()
    {
        Log.CloseAndFlush();
        if (_app != null)
        {
            await _app.StopAsync();
        }
    }

    // TODO: Add rest tests when rest sdk is implemented
}
