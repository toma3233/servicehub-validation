//go:debug x509negativeserial=1
//go:build testcontainers

package server

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"dev.azure.com/service-hub-flg/service_hub_validation/_git/service_hub_validation_service.git/operationcontainer/server/internal/logattrs"
	pb "github.com/Azure/OperationContainer/api/v1"
	"github.com/Azure/OperationContainer/api/v1/client"
	"github.com/Azure/OperationContainer/api/v1/restsdk"
	"github.com/Azure/aks-middleware/grpc/interceptor"
	"github.com/Azure/aks-middleware/http/client/direct/restlogger"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mssql"

	log "log/slog"

	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// This test only works when running in a VM, not a docker container since a docker container might not be able to create
// another container, that's why we have the "// +build testcontainers" at the beginning of this file. If you want to
// ignore this file, simply run "go test ./...", but if you want to test it with all the other test cases, run:
// "go test -tags=testcontainers ./..."
func createOperationStatus(buf *bytes.Buffer, port int, req *pb.CreateOperationStatusRequest) {
	logger := log.New(log.NewTextHandler(buf, nil))
	host := fmt.Sprintf("localhost:%d", port)
	options := interceptor.GetClientInterceptorLogOptions(logger, logattrs.GetAttrs())
	options.APIOutput = buf
	client, err := client.NewClient(host, options)
	// logging the error for transparency, but not failing the test since retry interceptor will handle it
	if err != nil {
		log.Error("did not connect: " + err.Error())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	client.CreateOperationStatus(ctx, req)
}

func getOperationStatus(buf *bytes.Buffer, port int, req *pb.GetOperationStatusRequest) {
	logger := log.New(log.NewTextHandler(buf, nil))
	host := fmt.Sprintf("localhost:%d", port)
	options := interceptor.GetClientInterceptorLogOptions(logger, logattrs.GetAttrs())
	options.APIOutput = buf
	client, err := client.NewClient(host, options)
	// logging the error for transparency, but not failing the test since retry interceptor will handle it
	if err != nil {
		log.Error("did not connect: " + err.Error())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	client.GetOperationStatus(ctx, req)
}

func updateOperationStatus(buf *bytes.Buffer, port int, req *pb.UpdateOperationStatusRequest) {
	logger := log.New(log.NewTextHandler(buf, nil))
	host := fmt.Sprintf("localhost:%d", port)
	options := interceptor.GetClientInterceptorLogOptions(logger, logattrs.GetAttrs())
	options.APIOutput = buf
	client, err := client.NewClient(host, options)
	// logging the error for transparency, but not failing the test since retry interceptor will handle it
	if err != nil {
		log.Error("did not connect: " + err.Error())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	client.UpdateOperationStatus(ctx, req)
}

var _ = Describe("Interceptor test", func() {

	var server *Server
	var serverPort int
	var httpPort int

	var in *pb.CreateOperationStatusRequest
	var ctx context.Context

	// Sql server for integration tests.
	var mssqlContainer *mssql.MSSQLServerContainer
	var dbConnectionString string

	var operationTableName string

	BeforeEach(func() {
		operationTableName = "operations"

		ctx = context.TODO()

		operationId, err := uuid.NewV4()
		if err != nil {
			return
		}

		in = &pb.CreateOperationStatusRequest{
			OperationName:       "LongRunning",
			EntityId:            "1",
			ExpirationTimestamp: timestamppb.New(time.Now().Add(1 * time.Hour)),
			OperationId:         operationId.String(),
		}

		password := "yourStrong(!)Password"
		mssqlContainer, err = mssql.Run(ctx,
			"mcr.microsoft.com/mssql/server:2022-CU14-ubuntu-22.04",
			mssql.WithAcceptEULA(),
			mssql.WithPassword(password),
		)
		if err != nil {
			log.Error("Failed to start container.")
			return
		}

		dbConnectionString, err = mssqlContainer.ConnectionString(ctx, "")
		if err != nil {
			log.Error("connection string not retrieved")
			return
		}
	})

	AfterEach(func() {
		if err := testcontainers.TerminateContainer(mssqlContainer); err != nil {
			log.Info("Failed to terminate container.")
		}

		if server != nil && server.IsRunning() {
			server.Cleanup()
		}
		server = nil
		serverPort = 0
	})

	Context("when the server is available", func() {
		BeforeEach(func() {

			options := Options{
				Port:                     0, // Use 0 to let the system assign an available port
				HTTPPort:                 0, // Same for HTTP port
				JsonLog:                  true,
				DatabaseConnectionString: dbConnectionString,
				OperationTableName:       operationTableName,
			}
			server = NewServer()
			server.Init(options)
			server.Serve(options)

			serverPort = server.GrpcPort
			httpPort = server.HttpPort
			// Explicitly testing state of server
			// Continue with tests once server and grpc-gateway are up and running
			Eventually(func() bool {
				return server.IsRunning() //&& IsServerRunning(demoserverPort)
			}, 10*time.Second).Should(BeTrue())
		})

		It("createOperationStatus should validate the operationName length", func() {
			var buf bytes.Buffer
			in.OperationName = ""
			createOperationStatus(&buf, serverPort, in)
			Expect(strings.Count(buf.String(), "headers")).To(Equal(1))
			Expect(buf.String()).To(ContainSubstring("value length must be at least 1 characters")) // must return error b/c name < 2 letters
		})

		It("createOperationStatus should validate the entityId length", func() {
			var buf bytes.Buffer
			in.EntityId = ""
			createOperationStatus(&buf, serverPort, in)
			Expect(strings.Count(buf.String(), "headers")).To(Equal(1))
			Expect(buf.String()).To(ContainSubstring("value length must be at least 1 characters")) // must return error b/c name < 2 letters
		})

		It("createOperationStatus should validate the operationId length", func() {
			var buf bytes.Buffer
			in.OperationId = ""
			createOperationStatus(&buf, serverPort, in)
			Expect(strings.Count(buf.String(), "headers")).To(Equal(1))
			Expect(buf.String()).To(ContainSubstring("value length must be at least 1 characters")) // must return error b/c name < 2 letters
		})

		It("updateOperationStatus should validate the operationId length", func() {
			var buf bytes.Buffer
			in := &pb.UpdateOperationStatusRequest{
				OperationId: "",
				Status:      0,
			}
			updateOperationStatus(&buf, serverPort, in)
			Expect(strings.Count(buf.String(), "headers")).To(Equal(1))
			Expect(buf.String()).To(ContainSubstring("value length must be at least 1 characters")) // must return error b/c name < 2 letters
		})

		It("getOperationStatus should validate the operationId length", func() {
			var buf bytes.Buffer
			in := &pb.GetOperationStatusRequest{
				OperationId: "",
			}
			getOperationStatus(&buf, serverPort, in)
			Expect(strings.Count(buf.String(), "headers")).To(Equal(1))
			Expect(buf.String()).To(ContainSubstring("value length must be at least 1 characters")) // must return error b/c name < 2 letters
		})

		It("should not retry the request", func() {
			var buf bytes.Buffer
			createOperationStatus(&buf, serverPort, in)
			Expect(strings.Count(buf.String(), "headers")).To(Equal(1))
			Expect(strings.Count(buf.String(), "OK")).To(Equal(1)) // must not retry
		})

		It("should successfully send a REST request", func() {
			logger := log.New(log.NewTextHandler(os.Stdout, nil))
			// Create a new Configuration instance
			cfg := &restsdk.Configuration{
				BasePath:      fmt.Sprintf("http://0.0.0.0:%d", httpPort),
				DefaultHeader: make(map[string]string),
				UserAgent:     "Swagger-Codegen/1.0.0/go",
				HTTPClient:    restlogger.NewLoggingClient(logger),
			}

			apiClient := restsdk.NewAPIClient(cfg)

			service := apiClient.OperationContainerApi

			operationId, err := uuid.NewV4()
			if err != nil {
				return
			}

			in := restsdk.OperationContainerCreateOperationStatusBody{
				OperationName:       "LongRunning",
				EntityId:            "1",
				ExpirationTimestamp: time.Now().Add(1 * time.Hour),
			}
			_, _, err = service.OperationContainerCreateOperationStatus(context.Background(), in, operationId.String())
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when the server is unavailable", func() {
		It("should retry the request", func() {
			var buf bytes.Buffer
			createOperationStatus(&buf, serverPort, in)
			Expect(strings.Count(buf.String(), "headers")).To(BeNumerically(">", 1)) // must retry
		})
	})
})
