using AKSMiddleware;
using Azure.Core;
using Azure.Identity;
using Azure.ResourceManager;
using Azure.ResourceManager.Resources;

namespace MyGreeterCsharp.Server.Handler;

public partial class MyGreeterCsharpServer : MyGreeterCsharp.Api.V1.MyGreeterCsharp.MyGreeterCsharpBase
{
    private ResourceGroupCollection? _resourceGroups;
    private readonly Serilog.ILogger _logger;
    public MyGreeterCsharpServer(ServerOptions options, Serilog.ILogger logger, ResourceGroupCollection? resourceGroups = null)
    {
        _logger = logger;
        // Used only for testing
        _resourceGroups = resourceGroups;

        if (resourceGroups != null)
        {
            _resourceGroups = resourceGroups;
        }

        if (options.EnableAzureSDKCalls)
        {
            // ArmPolicy is defined in aks-middleware-csharp
            // Ref: https://github.com/Azure/aks-middleware-csharp/blob/t-aduke/logProtoInMiddleware/policy/Policy.cs#L76
            var clientOptions = ArmPolicy.GetDefaultArmClientOptions(_logger);

            TokenCredential credential;
            if (!string.IsNullOrEmpty(options.IdentityResourceId))
            {
                ResourceIdentifier ResourceId = new ResourceIdentifier(options.IdentityResourceId);
                credential = new ManagedIdentityCredential(ResourceId);
            }
            else
            {
                credential = new DefaultAzureCredential();
            }
            try
            {
                var armClient = new ArmClient(credential, options.SubscriptionId, clientOptions);
                SubscriptionResource subscription = armClient.GetSubscriptionResource(new ResourceIdentifier($"/subscriptions/{options.SubscriptionId}"));
                _resourceGroups = subscription.GetResourceGroups();

            }
            catch (Exception ex)
            {
                _logger.Error(ex.Message);
                throw new ApplicationException("Failed to initialize resource groups.", ex);
            }
        }
    }
}
