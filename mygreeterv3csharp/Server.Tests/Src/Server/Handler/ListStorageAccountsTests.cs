using Grpc.Core;
using Moq;
using MyGreeterCsharp.Api.V1;
using MyGreeterCsharp.Server.Handler;
using Serilog;

namespace Server.Tests;

public class ListStorageAccountsTests
{
    private static readonly string SubscriptionId = "test-subscription";
    private static readonly string ResourceGroupName = "test-rg";

    private readonly Mock<ILogger> _mockLogger;
    private readonly MyGreeterCsharpServer _generatedServer;

    public ListStorageAccountsTests()
    {
        _mockLogger = new Mock<ILogger>();
        ServerOptions options = new ServerOptions { EnableAzureSDKCalls = false, SubscriptionId = SubscriptionId };
        _generatedServer = new MyGreeterCsharpServer(options, _mockLogger.Object);
    }

    [Fact]
    public async Task ListStorageAccounts_Success()
    {
        // Arrange
        ServerCallContext serverCallContext = new TestServerCallContext();
        ListStorageAccountRequest request = new ListStorageAccountRequest { RgName = ResourceGroupName };

        // Act
        var response = await _generatedServer.ListStorageAccounts(request, serverCallContext);

        // Assert
        Assert.NotNull(response);
        Assert.IsType<ListStorageAccountResponse>(response);
    }
}
