using Azure.ResourceManager.Resources;
using Grpc.Core;
using Moq;
using MyGreeterCsharp.Api.V1;
using MyGreeterCsharp.Server.Handler;
using Serilog;

namespace Server.Tests;

public class SayHelloTests
{
    private static readonly string SubscriptionId = "test-subscription";

    private readonly Mock<ResourceGroupCollection> _mockResourceGroups;
    private readonly Mock<ILogger> _mockLogger;
    private readonly MyGreeterCsharpServer _generatedServer;

    public SayHelloTests()
    {
        _mockResourceGroups = new Mock<ResourceGroupCollection>();
        _mockLogger = new Mock<ILogger>();

        ServerOptions options = new ServerOptions { EnableAzureSDKCalls = false, SubscriptionId = SubscriptionId };
        _generatedServer = new MyGreeterCsharpServer(options, _mockLogger.Object, _mockResourceGroups.Object);
    }

    [Fact]
    public async Task SayHello_Success()
    {
        // Arrange
        ServerCallContext serverCallContext = new TestServerCallContext();
        var request = new HelloRequest { Name = "test", Age = 1, Email = "test@test.com" };

        // Act
        var response = await _generatedServer.SayHello(request, serverCallContext);

        // Assert
        Assert.NotNull(response);
        Assert.IsType<HelloReply>(response);
    }
}
