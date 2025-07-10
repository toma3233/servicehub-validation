using Azure;
using Azure.Core;
using Azure.ResourceManager;
using Azure.ResourceManager.Resources;
using Grpc.Core;
using Moq;
using MyGreeterCsharp.Api.V1;
using MyGreeterCsharp.Server.Handler;
using Serilog;

namespace Server.Tests;

public class UpdateResourceGroupTests
{
    private static readonly string SubscriptionId = "test-subscription";
    private static readonly string ResourceGroupName = "test-resource-group";

    private readonly Mock<ResourceGroupCollection> _mockResourceGroups;
    private readonly Mock<ILogger> _mockLogger;
    private readonly Mock<ArmOperation<ResourceGroupResource>> _mockArmOperation;
    private readonly Mock<ResourceGroupResource> _mockResourceGroupResource;
    private readonly ServerOptions options;

    private MyGreeterCsharpServer _generatedServer;

    public UpdateResourceGroupTests()
    {
        _mockResourceGroups = new Mock<ResourceGroupCollection>();
        _mockLogger = new Mock<ILogger>();
        _mockArmOperation = new Mock<ArmOperation<ResourceGroupResource>>();
        _mockResourceGroupResource = new Mock<ResourceGroupResource>();

        options = new ServerOptions { EnableAzureSDKCalls = false, SubscriptionId = SubscriptionId };
        _generatedServer = new MyGreeterCsharpServer(options, _mockLogger.Object, _mockResourceGroups.Object);
    }

    [Fact]
    public async Task UpdateResourceGroup_ThrowsRpcException_WhenNullResourceGroupsClient()
    {
        // Arrange
        _generatedServer = new MyGreeterCsharpServer(options, _mockLogger.Object, null);
        ServerCallContext serverCallContext = new TestServerCallContext();
        var request = new UpdateResourceGroupRequest { Name = ResourceGroupName, Tags = { { "key", "value" } } };

        // Act & Assert
        var exception = await Assert.ThrowsAsync<RpcException>(
            () => _generatedServer.UpdateResourceGroup(request, serverCallContext));
        Assert.Equal(StatusCode.Unavailable, exception.StatusCode);
        Assert.Contains("ResourceGroupClient is nil", exception.Status.Detail);
    }

    [Fact]
    public async Task UpdateResourceGroup_Success()
    {
        var mockResponse = new Mock<Response>();
        AzureLocation location = AzureLocation.WestUS2;
        ResourceGroupData resourceGroupData = new ResourceGroupData(location);
        _mockResourceGroupResource
            .SetupGet(x => x.Data)
            .Returns(resourceGroupData);
        _mockResourceGroups
            .Setup(x => x.GetAsync(ResourceGroupName, It.IsAny<CancellationToken>()))
            .ReturnsAsync(Response.FromValue(_mockResourceGroupResource.Object, mockResponse.Object));
        _mockResourceGroupResource
            .Setup(x => x.AddTagAsync(It.IsAny<string>(), It.IsAny<string>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync(Response.FromValue(_mockResourceGroupResource.Object, mockResponse.Object));
        ServerCallContext serverCallContext = new TestServerCallContext();
        var request = new UpdateResourceGroupRequest { Name = ResourceGroupName, Tags = { { "key", "value" } } };
        // Act
        var response = await _generatedServer.UpdateResourceGroup(request, serverCallContext);

        // Assert
        Assert.NotNull(response);
        Assert.IsType<UpdateResourceGroupResponse>(response);
        Assert.Equal(location, response.ResourceGroup.Location);
        Assert.Empty(response.ResourceGroup.Name);
        Assert.Empty(response.ResourceGroup.Id);
    }

    [Fact]
    public async Task UpdateResourceGroup_ThrowsRpcException_WhenGetAsyncFail()
    {
        // Arrange
        var mockResponse = new Mock<Response>();
        AzureLocation location = AzureLocation.WestUS2;
        ResourceGroupData resourceGroupData = new ResourceGroupData(location);
        _mockResourceGroupResource
            .SetupGet(x => x.Data)
            .Returns(resourceGroupData);
        _mockResourceGroups
            .Setup(x => x.GetAsync(ResourceGroupName, It.IsAny<CancellationToken>()))
            .ThrowsAsync(new RequestFailedException("error"));
        ServerCallContext serverCallContext = new TestServerCallContext();
        var request = new UpdateResourceGroupRequest { Name = ResourceGroupName, Tags = { { "key", "value" } } };

        // Act & Assert
        var exception = await Assert.ThrowsAsync<RpcException>(
            () => _generatedServer.UpdateResourceGroup(request, serverCallContext));
        Assert.Equal(StatusCode.Unknown, exception.StatusCode);
    }

    [Fact]
    public async Task UpdateResourceGroup_ThrowsRpcException_WhenOthersFail()
    {
        // Arrange
        var mockResponse = new Mock<Response>();
        AzureLocation location = AzureLocation.WestUS2;
        ResourceGroupData resourceGroupData = new ResourceGroupData(location);
        _mockResourceGroupResource
            .SetupGet(x => x.Data)
            .Returns(resourceGroupData);
        _mockResourceGroups
            .Setup(x => x.GetAsync(ResourceGroupName, It.IsAny<CancellationToken>()))
            .ThrowsAsync(new Exception());
        ServerCallContext serverCallContext = new TestServerCallContext();
        var request = new UpdateResourceGroupRequest { Name = ResourceGroupName, Tags = { { "key", "value" } } };
        // Act & Assert
        var exception = await Assert.ThrowsAsync<RpcException>(
            () => _generatedServer.UpdateResourceGroup(request, serverCallContext));
        Assert.Equal(StatusCode.Unknown, exception.StatusCode);
    }
}
