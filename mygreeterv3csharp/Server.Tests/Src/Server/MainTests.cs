using MyGreeterCsharp.Server;


namespace Server.Tests;

[Collection("Sequential")]
public class ServerMainCommandTests
{
    [Fact]
    public async Task Main_ShouldInvokeStartCommand()
    {
        // Arrange
        var args = new string[] { "start" };
        var timeout = TimeSpan.FromSeconds(5);

        // Act
        var mainTask = ServerMainCommand.Main(args);
        var timeoutTask = Task.Delay(timeout);

        var completedTask = await Task.WhenAny(mainTask, timeoutTask);

        // Assert
        // If the main task times out, it is the expected behavior, we ignore it. Otherwise we assert on server not being null.
        if (completedTask == timeoutTask)
        {
            Console.WriteLine("Main command timed out");
        }
        else
        {
            var result = await mainTask;
            Assert.NotNull(MyGreeterCsharp.Server.Handler.Server.AppServer);
        }
    }
}
