// HTTP proxy server with OTEL audit middleware that forwards requests to gRPC-Gateway
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/Azure/aks-middleware/http/server/logging"
	"github.com/Azure/aks-middleware/http/server/otelaudit"
	"github.com/microsoft/go-otel-audit/audit"
	"github.com/microsoft/go-otel-audit/audit/conn"
	"github.com/microsoft/go-otel-audit/audit/msgs"
)

type HTTPProxyWithOtelAudit struct {
	server         *http.Server
	logger         *slog.Logger
	otelConfig     *otelaudit.OtelConfig
	port           int
	grpcGatewayURL string
	proxy          *httputil.ReverseProxy
}

// NewHTTPProxyWithOtelAudit creates a new HTTP proxy server with OTEL audit middleware
func NewHTTPProxyWithOtelAudit(logger *slog.Logger, port int, grpcGatewayPort int) *HTTPProxyWithOtelAudit {
	// Initialize OTEL audit client
	// Helper function to create a real audit client
	createAuditClient := func() *audit.Client {
		clienConn := func() (conn.Audit, error) {
			// MDSD_SOCKET_PATH environment variable is set in cluster deployment files
			ops := conn.DSPath(os.Getenv("MDSD_SOCKET_PATH"))
			if os.Getenv("MDSD_SOCKET_PATH") != "" {
				return conn.NewDomainSocket(ops)
			}
			return conn.NewNoOP(), nil

		}
		client, err := audit.New(clienConn)
		if err != nil {
			logger.Error("Failed to create audit client", "error", err)
			return nil
		}
		return client
	}
	otelConfig := &otelaudit.OtelConfig{
		Client: createAuditClient(),
		CustomOperationDescs: map[string]string{
			"POST /subscriptions/*/resourceGroups/*":                                                 "Create Azure resource group",
			"GET /subscriptions/*/resourceGroups/*":                                                  "Read Azure resource group",
			"DELETE /subscriptions/*/resourceGroups/*":                                               "Delete Azure resource group",
			"PUT /subscriptions/*/resourceGroups/*":                                                  "Update Azure resource group",
			"GET /subscriptions/*/resourceGroups":                                                    "List Azure resource groups",
			"POST /subscriptions/*/resourceGroups/*/providers/Microsoft.Storage/storageAccounts/*":   "Create Azure storage account",
			"GET /subscriptions/*/resourceGroups/*/providers/Microsoft.Storage/storageAccounts/*":    "Read Azure storage account",
			"DELETE /subscriptions/*/resourceGroups/*/providers/Microsoft.Storage/storageAccounts/*": "Delete Azure storage account",
			"PUT /subscriptions/*/resourceGroups/*/providers/Microsoft.Storage/storageAccounts/*":    "Update Azure storage account",
			"GET /subscriptions/*/resourceGroups/*/providers/Microsoft.Storage/storageAccounts":      "List Azure storage accounts",
		},
		CustomOperationCategories: map[string]msgs.OperationCategory{
			"POST /subscriptions/*/resourceGroups/*":                                                 msgs.ResourceManagement,
			"GET /subscriptions/*/resourceGroups/*":                                                  msgs.ResourceManagement,
			"DELETE /subscriptions/*/resourceGroups/*":                                               msgs.ResourceManagement,
			"PUT /subscriptions/*/resourceGroups/*":                                                  msgs.ResourceManagement,
			"GET /subscriptions/*/resourceGroups":                                                    msgs.ResourceManagement,
			"POST /subscriptions/*/resourceGroups/*/providers/Microsoft.Storage/storageAccounts/*":   msgs.ResourceManagement,
			"GET /subscriptions/*/resourceGroups/*/providers/Microsoft.Storage/storageAccounts/*":    msgs.ResourceManagement,
			"DELETE /subscriptions/*/resourceGroups/*/providers/Microsoft.Storage/storageAccounts/*": msgs.ResourceManagement,
			"PUT /subscriptions/*/resourceGroups/*/providers/Microsoft.Storage/storageAccounts/*":    msgs.ResourceManagement,
			"GET /subscriptions/*/resourceGroups/*/providers/Microsoft.Storage/storageAccounts":      msgs.ResourceManagement,
		},
		OperationAccessLevel: "Internal",
	}

	grpcGatewayURL := fmt.Sprintf("http://localhost:%d", grpcGatewayPort)

	// Parse the gRPC-Gateway URL
	targetURL, err := url.Parse(grpcGatewayURL)
	if err != nil {
		logger.Error("Failed to parse gRPC-Gateway URL", "url", grpcGatewayURL, "error", err)
		return nil
	}

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	return &HTTPProxyWithOtelAudit{
		logger:         logger,
		otelConfig:     otelConfig,
		port:           port,
		grpcGatewayURL: grpcGatewayURL,
		proxy:          proxy,
	}
}

// setupHandler creates the HTTP handler with logging, OTEL audit middleware and proxy
func (h *HTTPProxyWithOtelAudit) setupHandler() http.Handler {
	// Start with the proxy as the base handler
	var handler http.Handler = h.proxy

	// Apply OTEL audit middleware first (closest to the proxy)
	if h.otelConfig != nil && h.otelConfig.Client != nil {
		handler = otelaudit.NewOtelAuditLogging(h.logger, h.otelConfig)(handler)
		h.logger.Info("OTEL audit middleware enabled")
	} else {
		h.logger.Warn("OTEL audit middleware disabled - audit client unavailable")
	}

	// Apply logging middleware last (outermost layer)
	loggingMiddleware := logging.NewLogging(h.logger)
	handler = loggingMiddleware(handler)

	return handler
}

// Start starts the HTTP proxy server
func (h *HTTPProxyWithOtelAudit) Start() error {
	handler := h.setupHandler()

	h.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", h.port),
		Handler: handler,
	}

	h.logger.Info("Starting HTTP proxy server with OTEL audit middleware",
		"port", h.port,
		"target", h.grpcGatewayURL)

	go func() {
		if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			h.logger.Error("HTTP proxy server failed", "error", err)
		}
	}()

	return nil
}

// Stop gracefully stops the HTTP proxy server
func (h *HTTPProxyWithOtelAudit) Stop() error {
	if h.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	h.logger.Info("Stopping HTTP proxy server")
	return h.server.Shutdown(ctx)
}
