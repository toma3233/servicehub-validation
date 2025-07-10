using Grpc.Core;
using Moq;
using MyGreeterCsharp.Api.V1;
using MyGreeterCsharp.Server.Handler;
using Serilog;

namespace Server.Tests;

public class CreateStorageAccountTests
{
    private static readonly string SubscriptionId = "test-subscription";
    private static readonly string ResourceGroupName = "test-rg";
    private static readonly string Region = "east-us";

    private readonly Mock<ILogger> _mockLogger;
    private readonly MyGreeterCsharpServer _generatedServer;

    public CreateStorageAccountTests()
    {
        _mockLogger = new Mock<ILogger>();
        ServerOptions options = new ServerOptions { EnableAzureSDKCalls = false, SubscriptionId = SubscriptionId };
        _generatedServer = new MyGreeterCsharpServer(options, _mockLogger.Object);
    }

    [Fact]
    public async Task CreateStorageAccount_Success()
    {
        // Arrange
        ServerCallContext serverCallContext = new TestServerCallContext();
        CreateStorageAccountRequest request = new CreateStorageAccountRequest { RgName = ResourceGroupName, Region = Region };

        // Act
        var response = await _generatedServer.CreateStorageAccount(request, serverCallContext);

        // Assert
        Assert.NotNull(response);
        Assert.IsType<CreateStorageAccountResponse>(response);
        Assert.Equal("StorageName", response.Name);
    }
}
