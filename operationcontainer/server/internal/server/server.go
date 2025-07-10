// Auto generated. Don't modify.
package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"dev.azure.com/service-hub-flg/service_hub_validation/_git/service_hub_validation_service.git/operationcontainer/server/internal/logattrs"
	pb "github.com/Azure/OperationContainer/api/v1"

	"github.com/Azure/aks-middleware/grpc/interceptor"
	"github.com/Azure/aks-middleware/http/common"
	aksMiddlewareMetadata "github.com/Azure/aks-middleware/http/server/metadata"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	log "log/slog"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

func (s *Server) Serve(options Options) {
	logger := log.New(log.NewTextHandler(os.Stdout, nil).WithAttrs(logattrs.GetAttrs()))
	if options.JsonLog {
		logger = log.New(log.NewJSONHandler(os.Stdout, nil).WithAttrs(logattrs.GetAttrs()))
	}

	log.SetDefault(logger)

	s.Init(options)

	s.GrpcServer = grpc.NewServer(grpc.ChainUnaryInterceptor(
		interceptor.DefaultServerInterceptors(interceptor.GetServerInterceptorLogOptions(logger, logattrs.GetAttrs()))...,
	))
	pb.RegisterOperationContainerServer(s.GrpcServer, s)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s.GrpcServer, healthServer)

	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", options.Port))
	if err != nil {
		logger.Error("failed to listen: " + err.Error())
		os.Exit(1)
	}

	logger.Info(fmt.Sprintf("server listening at %s", grpcListener.Addr().String()))
	go func() {
		if err := s.GrpcServer.Serve(grpcListener); err != nil {
			logger.Error("failed to serve: " + err.Error())
			os.Exit(1)
		}
	}()

	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	s.GrpcPort = grpcListener.Addr().(*net.TCPAddr).Port

	serverAddress := grpcListener.Addr().String()

	conn, err := grpc.DialContext(
		context.Background(),
		serverAddress,
		grpc.WithInsecure(),
	)
	if err != nil {
		logger.Error("Failed to dial server: " + err.Error())
		os.Exit(1)
	}

	gwmux := runtime.NewServeMux(
		aksMiddlewareMetadata.NewMetadataMiddleware(common.HeaderToMetadata, common.MetadataToHeader)...,
	)
	err = pb.RegisterOperationContainerHandler(context.Background(), gwmux, conn)
	if err != nil {
		logger.Error("Failed to register gateway: " + err.Error())
		os.Exit(1)
	}

	httpListener, err := net.Listen("tcp", fmt.Sprintf(":%d", options.HTTPPort))
	if err != nil {
		logger.Error("failed to listen HTTP: " + err.Error())
		os.Exit(1)
	}

	s.GwServer = &http.Server{
		Handler:           gwmux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	logger.Info("serving gRPC-Gateway on " + httpListener.Addr().String())

	go func() {
		if err := s.GwServer.Serve(httpListener); err != nil {
			if err != http.ErrServerClosed {
				logger.Error("failed to serve HTTP: " + err.Error())
				os.Exit(1)
			} else {
				logger.Info("HTTP server closed")
			}
		}
	}()
	s.HttpPort = httpListener.Addr().(*net.TCPAddr).Port
}

func (s *Server) IsRunning() bool {
	timeout := 1 * time.Second

	grpcAddress := net.JoinHostPort("localhost", strconv.Itoa(s.GrpcPort))
	httpAddress := net.JoinHostPort("localhost", strconv.Itoa(s.HttpPort))

	// Check gRPC port
	grpcConn, err := net.DialTimeout("tcp", grpcAddress, timeout)
	if err != nil {
		return false
	}
	if grpcConn != nil {
		defer grpcConn.Close()
	} else {
		return false
	}

	// Check HTTP port
	httpConn, err := net.DialTimeout("tcp", httpAddress, timeout)
	if err != nil {
		return false
	}
	if httpConn != nil {
		defer httpConn.Close()
	} else {
		return false
	}

	return true
}

// Close the db connection for graceful shutdown.
func (s *Server) closeDb() {
	if s.dbClient != nil {
		if err := s.dbClient.Close(); err != nil {
			log.Error("Something went wrong closing the db connection")
		} else {
			log.Info("Database connection closed")
		}
	}
}

func (s *Server) Cleanup() {
	s.closeDb()
	if s.GrpcServer != nil {
		s.GrpcServer.GracefulStop()
	}
	if s.GwServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.GwServer.Shutdown(ctx); err != nil {
			log.Error("HTTP server Shutdown: " + err.Error())
		}
	}
}
