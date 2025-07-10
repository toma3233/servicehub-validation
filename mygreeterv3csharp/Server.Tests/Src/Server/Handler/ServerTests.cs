using Azure;
using Grpc.Core;

namespace Server.Tests;

public class ServerTests
{
    [Fact]
    public void HandleError_ReturnsCorrectRpcException_ForRequestFailedException()
    {
        // Arrange
        var exception = new RequestFailedException(404, "Not Found");
        var operation = "TestOperation";

        // Act
        var result = MyGreeterCsharp.Server.Handler.Server.HandleError(exception, operation);

        // Assert
        Assert.NotNull(result);
        Assert.Equal(StatusCode.NotFound, result.StatusCode);
        Assert.Contains("call error: Not Found", result.Status.Detail);
    }

    [Fact]
    public void HandleError_ReturnsCorrectRpcException_ForGenericException()
    {
        // Arrange
        var exception = new Exception("Generic error");
        var operation = "TestOperation";

        // Act
        var result = MyGreeterCsharp.Server.Handler.Server.HandleError(exception, operation);

        // Assert
        Assert.NotNull(result);
        Assert.Equal(StatusCode.Unknown, result.StatusCode);
        Assert.Contains("An unexpected error occurred during TestOperation: Generic error", result.Status.Detail);
    }
}
