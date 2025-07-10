using System.Diagnostics;
using Moq;
using MyGreeterCsharp.LogAttrs;
using Serilog.Core;
using Serilog.Events;
using Serilog.Parsing;

namespace Server.Tests;

public class LogAttrsTests
{
    private static readonly string Key = "test-key";
    private static readonly string Value = "test-value";

    [Fact]
    public void LogAttributes_AddAndGetAttr_Success()
    {
        // Arrange
        LogAttributes.AddAttr(Key, Value);

        // Act
        var attrs = LogAttributes.GetAttrs();

        // Assert
        Assert.Contains(new KeyValuePair<string, object>(Key, Value), attrs);
    }

    [Fact]
    public void CustomAttributeEnricher_Enrich_ShouldAddCustomProperty()
    {
        // Arrange
        LogAttributes.AddAttr(Key, Value);

        var logEvent = new LogEvent(
            DateTimeOffset.Now,
            LogEventLevel.Information,
            exception: null,
            messageTemplate: new MessageTemplate("Test message", Enumerable.Empty<MessageTemplateToken>()),
            properties: new List<LogEventProperty>(), traceId: ActivityTraceId.CreateRandom(),
            spanId: ActivitySpanId.CreateRandom());

        var propertyFactoryMock = new Mock<ILogEventPropertyFactory>();
        propertyFactoryMock.Setup(factory => factory.CreateProperty(Key, Value, false))
            .Returns(new LogEventProperty(Key, new ScalarValue(Value)));

        var enricher = new CustomAttributeEnricher();

        // Act
        enricher.Enrich(logEvent, propertyFactoryMock.Object);

        // Assert
        Assert.Contains(logEvent.Properties, p => p.Key == Key && p.Value.ToString() == $"\"{Value}\"");
    }
}
