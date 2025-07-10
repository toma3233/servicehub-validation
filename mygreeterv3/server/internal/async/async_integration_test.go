//go:debug x509negativeserial=1
//go:build testcontainers

package async

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	log "log/slog"
	"strings"

	"dev.azure.com/service-hub-flg/service_hub_validation/_git/service_hub_validation_service.git/mygreeterv3/server/internal/async/operations"
	"dev.azure.com/service-hub-flg/service_hub_validation/_git/service_hub_validation_service.git/mygreeterv3/server/internal/async/operations/longRunningOperation"
	oc "github.com/Azure/OperationContainer/api/v1"
	ocMock "github.com/Azure/OperationContainer/api/v1/mock"
	"github.com/Azure/aks-async/database"
	asyncErrors "github.com/Azure/aks-async/runtime/errors"
	"github.com/Azure/aks-async/runtime/matcher"
	"github.com/Azure/aks-async/runtime/operation"
	"github.com/Azure/aks-async/runtime/testutils/toolkit/convert"
	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/go-shuttle/v2"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mssql"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

// This test only works when running in a VM, not a docker container since a docker container might not be able to create
// another container, that's why we have the "// +build testcontainers" at the beginning of this file. If you want to
// ignore this file, simply run "go test ./...", but if you want to test it with all the other test cases, run:
// "go test -tags=testcontainers ./..."
var _ = Describe("Async Testing", func() {
	Context("Integration tests", func() {
		var (
			ctrl               *gomock.Controller
			ctx                context.Context
			entityTableName    string
			options            Options
			err                error
			mssqlContainer     *mssql.MSSQLServerContainer
			dbConnectionString string
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			ctx = context.Background()

			entityTableName = "hcp"

			password := "yourStrong(!)Password"
			mssqlContainer, err = mssql.Run(ctx,
				"mcr.microsoft.com/mssql/server:2022-CU14-ubuntu-22.04",
				mssql.WithAcceptEULA(),
				mssql.WithPassword(password),
			)
			Expect(err).To(BeNil())

			dbConnectionString, err = mssqlContainer.ConnectionString(ctx, "")
			Expect(err).To(BeNil())

			// Can only test the connection string, since testcontainers-go doesn't allow to retrieve
			// ServerUrl or ServerName.
			options = Options{
				EntityTableName:          entityTableName,
				DatabaseConnectionString: dbConnectionString,
				DatabasePort:             1433,
				DatabaseServerUrl:        "",
				DatabaseName:             "",
			}
		})

		AfterEach(func() {
			if err := testcontainers.TerminateContainer(mssqlContainer); err != nil {
				log.Error("Failed to terminate container.")
			}
			ctrl.Finish()
		})

		Context("async validation testing", func() {
			It("should fail if no entity table name provided", func() {

				options.EntityTableName = ""
				_, err := NewAsync(ctx, options, false)
				Expect(err).ToNot(BeNil())
			})
			It("should fail if entity table name doesn't meet regex requirements", func() {

				options.EntityTableName = "sample_entity_table!"
				_, err := NewAsync(ctx, options, false)
				Expect(err).ToNot(BeNil())
			})
			It("should fail to connect to db if no db options are provided", func() {

				options.DatabaseConnectionString = "" // The other options are already "".
				_, err := NewAsync(ctx, options, false)
				Expect(err).ToNot(BeNil())
			})
			// As of today, we can't fully create an async in a test, because we can't mock the servicebus client that is used to create a receiver passed into the processor.
			// However, if the failure is that no service bus receiver was received, then we know it will be due to this issue.
			// TODO(mheberling): Add mocking to the servicebus client
			It("async creation should fail due to no service bus receiver", func() {
				_, err := NewAsync(ctx, options, false)
				Expect(err).ToNot(BeNil())
				Expect(strings.Count(err.Error(), "No serviceBusReceiver received")).To(Equal(1))
			})
		})

		Context("entityController validation testing", func() {
			It("should be able to create an entityController", func() {
				dbClient, err := database.NewDbClientWithConnectionString(ctx, options.DatabaseConnectionString)
				Expect(err).To(BeNil())

				m := matcher.NewMatcher()
				_, err = NewEntityController(ctx, options, m, dbClient)
				Expect(err).To(BeNil())
			})
		})

		Context("entityController query testing", func() {
			var (
				dbClient         *sql.DB
				m                *matcher.Matcher
				operationRequest *operation.OperationRequest
				ec               *EntityController
			)
			BeforeEach(func() {
				operationRequest = &operation.OperationRequest{
					OperationName:       operations.LroName,
					ApiVersion:          "",
					OperationId:         "test_operation_id",
					Body:                nil,
					HttpMethod:          "",
					RetryCount:          0,
					EntityId:            "test_entity_id",
					EntityType:          "test_entity_type",
					ExpirationTimestamp: nil,
				}

				m = matcher.NewMatcher()
				lro := &longRunningOperation.LongRunningOperation{}
				m.Register(ctx, operations.LroName, lro)
				m.RegisterEntity(ctx, operations.LroName, longRunningOperation.CreateLroEntityFunc)

				dbClient, err = database.NewDbClientWithConnectionString(ctx, options.DatabaseConnectionString)
				Expect(err).To(BeNil())

				// Create the table for the entity.
				entityListCreateTableQuery := fmt.Sprintf("CREATE TABLE %s (entity_type VARCHAR(255), entity_id VARCHAR(255), last_operation_id VARCHAR(255), operation_name VARCHAR (255), operation_status VARCHAR(255))", entityTableName)
				_, err = database.QueryDb(context.Background(), dbClient, entityListCreateTableQuery)
				Expect(err).To(BeNil())

				ec, err = NewEntityController(ctx, options, m, dbClient)
				Expect(err).To(BeNil())
			})

			AfterEach(func() {
				dbClient.Close()
			})

			It("should be able to retrieve an entity", func() {
				// Insert sample value
				initialOperationStatus := oc.Status_PENDING.String()
				insertToEntityTable := fmt.Sprintf("INSERT INTO %s (entity_type, entity_id, last_operation_id, operation_name, operation_status) VALUES ('%s', '%s', '%s', '%s', '%s');", entityTableName, operationRequest.EntityType, operationRequest.EntityId, operationRequest.OperationId, operations.LroName, initialOperationStatus)
				_, err = database.ExecDb(ctx, dbClient, insertToEntityTable)

				entity, err := ec.GetEntity(ctx, operationRequest)
				Expect(err).To(BeNil())
				Expect(entity).ToNot(BeNil())
			})
			It("should fail to retrieve entity if there is no correspoding entity in the database", func() {
				_, err := ec.GetEntity(ctx, operationRequest)
				Expect(err).ToNot(BeNil())
				Expect(strings.Count(err.Error(), "EntityId not found in database.")).To(Equal(1))
			})
		})

		Context("Deadletterqueuehandler query testing", func() {
			var (
				buf                          bytes.Buffer
				dbClient                     *sql.DB
				operationRequest             *operation.OperationRequest
				operationContainerClient     *ocMock.MockOperationContainerClient
				handler                      shuttle.HandlerFunc
				settler                      shuttle.MessageSettler
				message                      *azservicebus.ReceivedMessage
				updateOperationStatusRequest *oc.UpdateOperationStatusRequest
				failedOperationStatus        oc.Status
			)

			BeforeEach(func() {
				buf.Reset()
				logger := log.New(log.NewTextHandler(&buf, nil))
				ctx = ctxlogger.WithLogger(ctx, logger)
				operationContainerClient = ocMock.NewMockOperationContainerClient(ctrl)

				operationRequest = &operation.OperationRequest{
					OperationName:       operations.LroName,
					ApiVersion:          "",
					OperationId:         "test_operation_id",
					Body:                nil,
					HttpMethod:          "",
					RetryCount:          0,
					EntityId:            "test_entity_id",
					EntityType:          "test_entity_type",
					ExpirationTimestamp: nil,
				}

				marshaller := &shuttle.DefaultProtoMarshaller{}
				marshalledOperation, err := marshaller.Marshal(operationRequest)
				Expect(err).To(BeNil())

				message = convert.ConvertToReceivedMessage(marshalledOperation)

				failedOperationStatus = oc.Status_FAILED
				updateOperationStatusRequest = &oc.UpdateOperationStatusRequest{
					OperationId: operationRequest.OperationId,
					Status:      failedOperationStatus,
				}

				settler = &fakeMessageSettler{}

				dbClient, err = database.NewDbClientWithConnectionString(ctx, options.DatabaseConnectionString)
				Expect(err).To(BeNil())

				// Create the table for the entity.
				entityListCreateTableQuery := fmt.Sprintf("CREATE TABLE %s (entity_type VARCHAR(255), entity_id VARCHAR(255), last_operation_id VARCHAR(255), operation_name VARCHAR (255), operation_status VARCHAR(255))", entityTableName)
				_, err = database.QueryDb(ctx, dbClient, entityListCreateTableQuery)
				Expect(err).To(BeNil())

				handler = NewDeadLetterQueueHandler(options, operationContainerClient, dbClient, nil)
			})

			It("should update the operation status", func() {
				// Insert sample value
				initialOperationStatus := oc.Status_PENDING.String()
				insertToEntityTable := fmt.Sprintf("INSERT INTO %s (entity_type, entity_id, last_operation_id, operation_name, operation_status) VALUES ('%s', '%s', '%s', '%s', '%s');", entityTableName, operationRequest.EntityType, operationRequest.EntityId, operationRequest.OperationId, operations.LroName, initialOperationStatus)
				_, err = database.ExecDb(ctx, dbClient, insertToEntityTable)

				// Run handler
				operationContainerClient.EXPECT().UpdateOperationStatus(gomock.Any(), updateOperationStatusRequest).Return(nil, nil)
				handler(ctx, settler, message)
				Expect(strings.Count(buf.String(), "DeadLetterQueueHandler: Successfully set the operation")).To(Equal(1))

				// Verify result
				checkNewStatusQuery := fmt.Sprintf("SELECT operation_status FROM %s WHERE last_operation_id = '%s';", entityTableName, operationRequest.OperationId)
				rows, err := database.QueryDb(ctx, dbClient, checkNewStatusQuery)
				Expect(err).To(BeNil())

				var operationStatus string
				if rows.Next() {
					err = rows.Scan(&operationStatus)
				} else {
					err = errors.New("No rows return for OperationId: " + operationRequest.OperationId)
				}

				Expect(err).To(BeNil())
				Expect(operationStatus).To(Equal(oc.Status_FAILED.String()))
			})
		})

		Context("hooks query testing", func() {
			var (
				hookedOperation          *OperationStatusHook
				dbClient                 *sql.DB
				operationRequest         *operation.OperationRequest
				op                       *longRunningOperation.LongRunningOperation
				sampleError              *asyncErrors.AsyncError
				insertToEntityTableQuery string
			)
			BeforeEach(func() {
				operationRequest = &operation.OperationRequest{
					OperationName:       operations.LroName,
					ApiVersion:          "",
					OperationId:         "test_operation_id",
					Body:                nil,
					HttpMethod:          "",
					RetryCount:          0,
					EntityId:            "test_entity_id",
					EntityType:          "test_entity_type",
					ExpirationTimestamp: nil,
				}

				op = &longRunningOperation.LongRunningOperation{
					Operation: operationRequest,
				}

				errorMessage := "Error message"
				tempErr := errors.New(errorMessage)
				sampleError = &asyncErrors.AsyncError{
					OriginalError: tempErr,
					Message:       errorMessage,
				}

				dbClient, err = database.NewDbClientWithConnectionString(ctx, options.DatabaseConnectionString)
				if err != nil {
					return
				}

				// Create the table for the entity.
				createEntityTableQuery := fmt.Sprintf("CREATE TABLE %s (entity_type VARCHAR(255), entity_id VARCHAR(255), last_operation_id VARCHAR(255), operation_name VARCHAR (255), operation_status VARCHAR(255))", entityTableName)
				_, err = database.QueryDb(context.Background(), dbClient, createEntityTableQuery)
				if err != nil {
					log.Error("Error creating table: " + err.Error())
					return
				}

				insertToEntityTableQuery = fmt.Sprintf("INSERT INTO %s (entity_type, entity_id, last_operation_id, operation_name, operation_status) VALUES ('%s', '%s', '%s', '%s', @p1);", entityTableName, operationRequest.EntityType, operationRequest.EntityId, operationRequest.OperationId, operations.LroName)

				hookedOperation = &OperationStatusHook{
					dbClient:        dbClient,
					EntityTableName: entityTableName,
				}
			})

			Describe("BeforeInitOperation", func() {
				BeforeEach(func() {
					// Insert sample value
					initialOperationStatus := oc.Status_PENDING.String()
					_, err = database.ExecDb(ctx, dbClient, insertToEntityTableQuery, initialOperationStatus)
					if err != nil {
						log.Error("Error setting sample value: " + err.Error())
						return
					}
				})

				It("should update entity database with IN_PROGRESS status", func() {
					err := hookedOperation.BeforeInitOperation(ctx, operationRequest)
					Expect(err).To(BeNil())

					newStatus, checkErr := checkCurrentEntityStatus(ctx, dbClient, entityTableName, operationRequest)
					Expect(checkErr).To(BeNil())
					Expect(newStatus).To(Equal(oc.Status_IN_PROGRESS.String()))
				})
			})

			Describe("AfterInitOperation", func() {
				BeforeEach(func() {
					// Insert sample value
					initialOperationStatus := oc.Status_IN_PROGRESS.String()
					_, err = database.ExecDb(ctx, dbClient, insertToEntityTableQuery, initialOperationStatus)
					if err != nil {
						log.Error("Error setting sample value: " + err.Error())
						return
					}
				})

				It("should update entity database with PENDING status if there was an error", func() {
					err := hookedOperation.AfterInitOperation(ctx, op, operationRequest, sampleError)
					Expect(err).To(BeNil())

					newStatus, checkErr := checkCurrentEntityStatus(ctx, dbClient, entityTableName, operationRequest)
					Expect(checkErr).To(BeNil())
					Expect(newStatus).To(Equal(oc.Status_PENDING.String()))
				})
				It("should not update entity database with IN_PROGRESS status if there was no en error", func() {
					err := hookedOperation.AfterInitOperation(ctx, op, operationRequest, nil)
					Expect(err).To(BeNil())

					newStatus, checkErr := checkCurrentEntityStatus(ctx, dbClient, entityTableName, operationRequest)
					Expect(checkErr).To(BeNil())
					Expect(newStatus).To(Equal(oc.Status_IN_PROGRESS.String()))
				})
			})

			Describe("BeforeGuardConcurrency", func() {
				BeforeEach(func() {
					// Insert sample value
					initialOperationStatus := oc.Status_IN_PROGRESS.String()
					_, err = database.ExecDb(ctx, dbClient, insertToEntityTableQuery, initialOperationStatus)
					if err != nil {
						log.Error("Error setting sample value: " + err.Error())
						return
					}
				})

				It("should not update entity database", func() {
					err := hookedOperation.BeforeGuardConcurrency(ctx, op, nil)
					Expect(err).To(BeNil())

					newStatus, checkErr := checkCurrentEntityStatus(ctx, dbClient, entityTableName, operationRequest)
					Expect(checkErr).To(BeNil())
					Expect(newStatus).To(Equal(oc.Status_IN_PROGRESS.String()))
				})
			})

			Describe("AfterGuardConcurrency", func() {
				BeforeEach(func() {
					// Insert sample value
					initialOperationStatus := oc.Status_IN_PROGRESS.String()
					_, err = database.ExecDb(ctx, dbClient, insertToEntityTableQuery, initialOperationStatus)
					if err != nil {
						log.Error("Error setting sample value: " + err.Error())
						return
					}
				})

				It("should not update entity database when there's no error", func() {
					err := hookedOperation.AfterGuardConcurrency(ctx, op, nil)
					Expect(err).To(BeNil())

					newStatus, checkErr := checkCurrentEntityStatus(ctx, dbClient, entityTableName, operationRequest)
					Expect(checkErr).To(BeNil())
					Expect(newStatus).To(Equal(oc.Status_IN_PROGRESS.String()))
				})
				It("should update entity database value to PENDING when there is a asyncError", func() {
					err := hookedOperation.AfterGuardConcurrency(ctx, op, sampleError)
					Expect(err).To(BeNil())

					newStatus, checkErr := checkCurrentEntityStatus(ctx, dbClient, entityTableName, operationRequest)
					Expect(checkErr).To(BeNil())
					Expect(newStatus).To(Equal(oc.Status_PENDING.String()))
				})
			})

			Describe("BeforeRun", func() {
				BeforeEach(func() {
					// Insert sample value
					initialOperationStatus := oc.Status_IN_PROGRESS.String()
					_, err = database.ExecDb(ctx, dbClient, insertToEntityTableQuery, initialOperationStatus)
					if err != nil {
						log.Error("Error setting sample value: " + err.Error())
						return
					}
				})

				It("should not update the entityDatabase", func() {
					err := hookedOperation.BeforeRun(ctx, op)
					Expect(err).To(BeNil())

					newStatus, checkErr := checkCurrentEntityStatus(ctx, dbClient, entityTableName, operationRequest)
					Expect(checkErr).To(BeNil())
					Expect(newStatus).To(Equal(oc.Status_IN_PROGRESS.String()))
				})
			})

			Describe("AfterRun", func() {
				BeforeEach(func() {
					// Insert sample value
					initialOperationStatus := oc.Status_IN_PROGRESS.String()
					_, err = database.ExecDb(ctx, dbClient, insertToEntityTableQuery, initialOperationStatus)
					if err != nil {
						log.Error("Error setting sample value: " + err.Error())
						return
					}
				})

				It("should update entity database value to COMPLETED if no error returned", func() {
					err := hookedOperation.AfterRun(ctx, op, nil)
					Expect(err).To(BeNil())

					newStatus, checkErr := checkCurrentEntityStatus(ctx, dbClient, entityTableName, operationRequest)
					Expect(checkErr).To(BeNil())
					Expect(newStatus).To(Equal(oc.Status_SUCCEEDED.String()))
				})
				It("should update entity database value to PENDING if an error returned", func() {
					err := hookedOperation.AfterRun(ctx, op, sampleError)
					Expect(err).To(BeNil())

					newStatus, checkErr := checkCurrentEntityStatus(ctx, dbClient, entityTableName, operationRequest)
					Expect(checkErr).To(BeNil())
					Expect(newStatus).To(Equal(oc.Status_PENDING.String()))
				})
			})
		})
	})
})

// This function is used to check the new status of the entity.
func checkCurrentEntityStatus(ctx context.Context, dbClient *sql.DB, entityTableName string, opReq *operation.OperationRequest) (string, error) {
	checkNewStatusQuery := fmt.Sprintf("SELECT operation_status FROM %s WHERE last_operation_id = '%s';", entityTableName, opReq.OperationId)
	rows, err := database.QueryDb(ctx, dbClient, checkNewStatusQuery)
	if err != nil {
		log.Error("Something went wrong verifying test result: " + err.Error())
		Expect(err).To(BeNil())
	}

	var operationStatus string
	if rows.Next() {
		err := rows.Scan(&operationStatus)
		if err != nil {
			log.Error("Error scanning row: " + err.Error())
			return "", err
		}
	} else {
		log.Error("No rows returned for OperationId: " + opReq.OperationId)
		return "", err
	}

	return operationStatus, nil
}
