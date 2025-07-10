using MyGreeterCsharp.Client;

namespace Server.Tests;

public class ClientMainCommandTests
{
    [Fact]
    public async Task Main_ShouldInvokeStartCommand()
    {
        // Arrange
        var args = new string[] { "start" };

        // Act
        var result = await ClientMainCommand.Main(args);

        // Assert
        Assert.Equal(1, result);
    }
}
