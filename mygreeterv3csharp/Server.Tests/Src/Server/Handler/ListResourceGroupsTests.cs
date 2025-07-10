using Azure;
using Azure.Core;
using Azure.ResourceManager;
using Azure.ResourceManager.Resources;
using Google.Protobuf.WellKnownTypes;
using Grpc.Core;
using Moq;
using MyGreeterCsharp.Api.V1;
using MyGreeterCsharp.Server.Handler;
using Serilog;

namespace Server.Tests;

public class ListResourceGroupsTests
{
    private static readonly string SubscriptionId = "test-subscription";
    private static readonly string ResourceGroupDataId = "/subscriptions/123/resourceGroups/TestId";

    private readonly Mock<ResourceGroupCollection> _mockResourceGroups;
    private readonly Mock<ILogger> _mockLogger;
    private readonly Mock<ArmOperation<ResourceGroupResource>> _mockArmOperation;
    private readonly Mock<ResourceGroupResource> _mockResourceGroupResource;
    private readonly ServerOptions options;

    private MyGreeterCsharpServer _generatedServer;

    public ListResourceGroupsTests()
    {
        _mockResourceGroups = new Mock<ResourceGroupCollection>();
        _mockLogger = new Mock<ILogger>();
        _mockArmOperation = new Mock<ArmOperation<ResourceGroupResource>>();
        _mockResourceGroupResource = new Mock<ResourceGroupResource>();

        options = new ServerOptions { EnableAzureSDKCalls = false, SubscriptionId = SubscriptionId };
        _generatedServer = new MyGreeterCsharpServer(options, _mockLogger.Object, _mockResourceGroups.Object);
    }

    [Fact]
    public async Task ListResourceGroups_ThrowsRpcException_WhenNullResourceGroupsClient()
    {
        // Arrange
        _generatedServer = new MyGreeterCsharpServer(options, _mockLogger.Object, null);
        ServerCallContext serverCallContext = new TestServerCallContext();
        var request = new Empty();

        // Act & Assert
        var exception = await Assert.ThrowsAsync<RpcException>(
            () => _generatedServer.ListResourceGroups(request, serverCallContext));
        Assert.Equal(StatusCode.Unavailable, exception.StatusCode);
        Assert.Contains("ResourceGroupClient is nil", exception.Status.Detail);
    }

    [Fact]
    public async Task ListResourceGroups_Success()
    {
        // Arrange
        var mockResponse = new Mock<Response>();
        AzureLocation location = AzureLocation.WestUS2;
        ResourceGroupData resourceGroupData = new ResourceGroupData(location);
        _mockResourceGroupResource
            .SetupGet(x => x.Id)
            .Returns(new ResourceIdentifier(ResourceGroupDataId));
        _mockResourceGroupResource
            .SetupGet(x => x.Data)
            .Returns(resourceGroupData);
        var resourceGroupList = new List<ResourceGroupResource> { _mockResourceGroupResource.Object };
        Page<ResourceGroupResource> page =
            Page<ResourceGroupResource>.FromValues(resourceGroupList, null, mockResponse.Object);
        AsyncPageable<ResourceGroupResource> pageable = AsyncPageable<ResourceGroupResource>.FromPages(new[] { page });

        _mockResourceGroups
            .Setup(x => x.GetAllAsync(
                It.IsAny<String>(),
                It.IsAny<Nullable<Int32>>(),
                It.IsAny<CancellationToken>()))
            .Returns(pageable);
        ServerCallContext serverCallContext = new TestServerCallContext();
        var request = new Empty();

        // Act
        var response = await _generatedServer.ListResourceGroups(request, serverCallContext);

        // Assert
        Assert.NotNull(response);
        Assert.IsType<ListResourceGroupResponse>(response);
        Assert.Single(response.RgList);
        Assert.Equal(ResourceGroupDataId, response.RgList[0].Id);
        Assert.Equal(location.ToString(), response.RgList[0].Location);
        Assert.Empty(response.RgList[0].Name);
    }

    [Fact]
    public async Task ListResourceGroups_ThrowsRpcException_WhenListResourceGroupsFail()
    {
        // Arrange
        _mockResourceGroups
          .Setup(x => x.GetAllAsync(
                It.IsAny<String>(),
                It.IsAny<Nullable<Int32>>(),
                It.IsAny<CancellationToken>()))
          .Throws(new Exception());
        ServerCallContext serverCallContext = new TestServerCallContext();
        var request = new Empty();

        // Act & Assert
        var exception = await Assert.ThrowsAsync<RpcException>(
            () => _generatedServer.ListResourceGroups(request, serverCallContext));
        Assert.Equal(StatusCode.Unknown, exception.StatusCode);
    }
}
