using Google.Protobuf.WellKnownTypes;
using Grpc.Core;
using Moq;
using MyGreeterCsharp.Api.V1;
using MyGreeterCsharp.Server.Handler;
using Serilog;

namespace Server.Tests;

public class DeleteStorageAccountTests
{
    private static readonly string SubscriptionId = "test-subscription";
    private static readonly string ResourceGroupName = "test-rg";
    private static readonly string ServiceAccountName = "test-service-account";

    private readonly Mock<ILogger> _mockLogger;
    private readonly MyGreeterCsharpServer _generatedServer;

    public DeleteStorageAccountTests()
    {
        _mockLogger = new Mock<ILogger>();
        ServerOptions options = new ServerOptions { EnableAzureSDKCalls = false, SubscriptionId = SubscriptionId };
        _generatedServer = new MyGreeterCsharpServer(options, _mockLogger.Object);
    }

    [Fact]
    public async Task DeleteStorageAccount_Success()
    {
        // Arrange
        ServerCallContext serverCallContext = new TestServerCallContext();
        DeleteStorageAccountRequest request =
            new DeleteStorageAccountRequest { RgName = ResourceGroupName, SaName = ServiceAccountName };

        // Act
        var response = await _generatedServer.DeleteStorageAccount(request, serverCallContext);

        // Assert
        Assert.NotNull(response);
        Assert.IsType<Empty>(response);
    }
}
