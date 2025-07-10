using MyGreeterCsharp.Server.Handler;

namespace Server.Tests;

public class StartCommandTests
{
    [Fact]
    public void Init_ShouldReturnCommandWithExpectedOptions()
    {
        // Act
        var command = StartCommand.Init();

        // Assert
        Assert.NotNull(command);
        Assert.Equal("start", command.Name);
        Assert.Equal(8, command.Options.Count);
        Assert.Contains(command.Options, o => o.Name == "port");
        Assert.Contains(command.Options, o => o.Name == "json-log");
        Assert.Contains(command.Options, o => o.Name == "subscription-id");
        Assert.Contains(command.Options, o => o.Name == "enable-azureSDK-calls");
        Assert.Contains(command.Options, o => o.Name == "http-port");
        Assert.Contains(command.Options, o => o.Name == "remote-addr");
        Assert.Contains(command.Options, o => o.Name == "interval-milli-sec");
        Assert.Contains(command.Options, o => o.Name == "identity-resource-id");
    }
}
