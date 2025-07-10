using Grpc.Core;

namespace Server.Tests;
public class TestServerCallContext : ServerCallContext
{
    protected override Metadata RequestHeadersCore => new Metadata();
    protected override DateTime DeadlineCore => DateTime.UtcNow.AddMinutes(1);
    protected override string HostCore => "localhost";
    protected override string MethodCore => "TestMethod";
    protected override string PeerCore => "Peer";
    protected override CancellationToken CancellationTokenCore => CancellationToken.None;
    protected override Metadata ResponseTrailersCore => new Metadata();
    protected override Status StatusCore { get; set; }
    protected override WriteOptions? WriteOptionsCore { get; set; } = new WriteOptions();
    protected override AuthContext AuthContextCore => new AuthContext("test", new Dictionary<string, List<AuthProperty>>());
    protected override ContextPropagationToken CreatePropagationTokenCore(ContextPropagationOptions? options)
    {
        throw new NotImplementedException();
    }
    protected override Task WriteResponseHeadersAsyncCore(Metadata responseHeaders) => Task.CompletedTask;
}


