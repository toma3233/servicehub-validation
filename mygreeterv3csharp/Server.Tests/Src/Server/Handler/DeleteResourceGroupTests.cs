using Azure;
using Azure.ResourceManager;
using Azure.ResourceManager.Resources;
using Google.Protobuf.WellKnownTypes;
using Grpc.Core;
using Moq;
using MyGreeterCsharp.Api.V1;
using MyGreeterCsharp.Server.Handler;
using Serilog;

namespace Server.Tests;

public class DeleteResourceGroupTests
{
    private static readonly string SubscriptionId = "test-subscription";
    private static readonly string ResourceGroupName = "test-rg";

    private readonly Mock<ResourceGroupCollection> _mockResourceGroups;
    private readonly Mock<ILogger> _mockLogger;
    private readonly Mock<ArmOperation<ResourceGroupResource>> _mockArmOperation;
    private readonly Mock<ResourceGroupResource> _mockResourceGroupResource;
    private readonly ServerOptions options;

    private MyGreeterCsharpServer _generatedServer;

    public DeleteResourceGroupTests()
    {
        _mockResourceGroups = new Mock<ResourceGroupCollection>();
        _mockLogger = new Mock<ILogger>();
        _mockArmOperation = new Mock<ArmOperation<ResourceGroupResource>>();
        _mockResourceGroupResource = new Mock<ResourceGroupResource>();

        options = new ServerOptions { EnableAzureSDKCalls = false, SubscriptionId = SubscriptionId };
        _generatedServer = new MyGreeterCsharpServer(
            options,
            _mockLogger.Object,
            _mockResourceGroups.Object);
    }

    [Fact]
    public async Task DeleteResourceGroup_ThrowsRpcException_WhenNullResourceGroupsClient()
    {
        // Arrange
        _generatedServer = new MyGreeterCsharpServer(options, _mockLogger.Object, null);
        ServerCallContext serverCallContext = new TestServerCallContext();
        var request = new DeleteResourceGroupRequest { Name = ResourceGroupName };

        // Act & Assert
        var exception = await Assert.ThrowsAsync<RpcException>(
            () => _generatedServer.DeleteResourceGroup(request, serverCallContext));
        Assert.Equal(StatusCode.Unavailable, exception.StatusCode);
        Assert.Contains("ResourceGroupClient is nil", exception.Status.Detail);
    }

    [Fact]
    public async Task DeleteResourceGroup_Success()
    {
        // Arrange
        var mockResponse = new Mock<Response>();
        _mockResourceGroups
            .Setup(x => x.GetAsync(
                ResourceGroupName,
                It.IsAny<CancellationToken>()))
            .ReturnsAsync(Response.FromValue(_mockResourceGroupResource.Object, mockResponse.Object));
        ServerCallContext serverCallContext = new TestServerCallContext();
        var request = new DeleteResourceGroupRequest { Name = ResourceGroupName };

        // Act
        var response = await _generatedServer.DeleteResourceGroup(request, serverCallContext);

        // Assert
        Assert.NotNull(response);
        Assert.IsType<Empty>(response);
    }

    [Fact]
    public async Task DeleteResourceGroup_ThrowsRpcException_WhenDeleteResourceGroupFail()
    {
        // Arrange
        _mockResourceGroups
            .Setup(x => x.GetAsync(
                ResourceGroupName,
                It.IsAny<CancellationToken>()))
            .ThrowsAsync(new RequestFailedException("error"));
        ServerCallContext serverCallContext = new TestServerCallContext();
        var request = new DeleteResourceGroupRequest { Name = ResourceGroupName };

        // Act & Assert
        var exception = await Assert.ThrowsAsync<RpcException>(
            () => _generatedServer.DeleteResourceGroup(request, serverCallContext));
        Assert.Equal(StatusCode.Unknown, exception.StatusCode);
    }

    [Fact]
    public async Task DeleteResourceGroup_ThrowsRpcException_WhenOthersFail()
    {
        // Arrange
        _mockResourceGroups
            .Setup(x => x.GetAsync(
                ResourceGroupName,
                It.IsAny<CancellationToken>()))
            .ThrowsAsync(new Exception());
        ServerCallContext serverCallContext = new TestServerCallContext();
        var request = new DeleteResourceGroupRequest { Name = ResourceGroupName };

        // Act & Assert
        var exception = await Assert.ThrowsAsync<RpcException>(
            () => _generatedServer.DeleteResourceGroup(request, serverCallContext));
        Assert.Equal(StatusCode.Unknown, exception.StatusCode);
    }
}
