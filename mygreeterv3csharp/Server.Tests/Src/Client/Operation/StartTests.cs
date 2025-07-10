using System.CommandLine;
using MyGreeterCsharp.Client.Operation;

namespace Server.Tests;

public class StartTests
{
    [Fact]
    public void Init_Success()
    {
        // Arrange & Act
        var command = StartCommand.Init();

        // Assert
        Assert.NotNull(command);
        Assert.IsType<Command>(command);
    }

    [Fact]
    public async Task Hello_Success()
    {
        // Arrange
        var options = new ClientOptions
        {
            JsonLog = true,
            RemoteAddr = "localhost:50051",
            IntervalMilliSec = -1,
            Name = "TestName",
            Age = 30,
            Email = "test@example.com",
            Address = "123 Test St, Seattle, WA 98101"
        };

        // Act & Assert
        // Tests that it finishes without error. Server is not started,
        // but a new client is created, failures will be caught in catch block.
        await StartCommand.hello(options);
    }

    [Fact]
    public async Task Hello_ThrowsException_WhenInvalidInputAddress()
    {
        // Arrange
        var options = new ClientOptions
        {
            JsonLog = true,
            RemoteAddr = "localhost:50051",
            IntervalMilliSec = -1,
            Name = "TestName",
            Age = 30,
            Email = "test@example.com",
            Address = "123 Test St"
        };

        // Act & Assert
        var exception = await Assert.ThrowsAsync<ArgumentException>(() => StartCommand.hello(options));
        Assert.Contains("Address format is incorrect", exception.Message);
    }

    [Fact]
    public async Task Hello_ThrowsException_WhenInvalidInput()
    {
        // Arrange
        var options = new ClientOptions
        {
            JsonLog = true,
            RemoteAddr = "",
            IntervalMilliSec = -1,
            Name = "TestName",
            Age = 30,
            Email = "test@example.com",
            Address = "123 Test St, Seattle, WA 98101"
        };

        // Act & Assert
        var exception = await Assert.ThrowsAsync<UriFormatException>(() => StartCommand.hello(options));
        Assert.Contains("Invalid URI", exception.Message);
    }
}
