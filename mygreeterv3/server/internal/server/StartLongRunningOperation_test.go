package server

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"

	"time"

	pb "dev.azure.com/service-hub-flg/service_hub_validation/_git/service_hub_validation_service.git/mygreeterv3/api/v1"
	"dev.azure.com/service-hub-flg/service_hub_validation/_git/service_hub_validation_service.git/mygreeterv3/server/internal/async/operations"

	oc "github.com/Azure/OperationContainer/api/v1"
	ocMock "github.com/Azure/OperationContainer/api/v1/mock"
	"github.com/Azure/aks-async/runtime/operation"
	"github.com/Azure/aks-async/servicebus"
	"github.com/Azure/go-shuttle/v2"
	"github.com/DATA-DOG/go-sqlmock"

	asyncMocks "github.com/Azure/aks-async/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	entityTableUpdateQuery = `
      MERGE INTO %s AS target
      USING \(SELECT @p1 AS entity_id, @p2 AS entity_type, @p3 AS last_operation_id, @p4 AS operation_name, @p5 AS operation_status\) AS source
      ON target.entity_id = source.entity_id and target.entity_type = source.entity_type
      WHEN MATCHED AND \(target.operation_status = 'SUCCEEDED' OR target.operation_status = 'FAILED' OR target.operation_status = 'CANCELED' OR target.operation_status = 'UNKNOWN'\) THEN
        UPDATE SET
          target.last_operation_id = source.last_operation_id,
          target.operation_name = source.operation_name,
          target.operation_status = source.operation_status
      WHEN NOT MATCHED THEN
        INSERT \(entity_id, entity_type, last_operation_id, operation_name, operation_status\)
        VALUES \(source.entity_id, source.entity_type, source.last_operation_id, source.operation_name, source.operation_status\);
     `
)

// Match satisfies sqlmock.Argument interface.
// Required for checking that the operationId matches a string format.
type AnyString struct{}

func (a AnyString) Match(v driver.Value) bool {
	_, ok := v.(string)
	return ok
}

var _ = Describe("Mock Testing", func() {
	var (
		ctrl                     *gomock.Controller
		s                        *Server
		mockSender               *asyncMocks.MockSenderInterface
		db                       *sql.DB
		mockDb                   sqlmock.Sqlmock
		entityTableName          string
		query                    string
		operationContainerClient *ocMock.MockOperationContainerClient
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockSender = asyncMocks.NewMockSenderInterface(ctrl)
		db, mockDb, _ = sqlmock.New()
		entityTableName = "hcp"
		operationContainerClient = ocMock.NewMockOperationContainerClient(ctrl)
		s = &Server{
			serviceBusSender:         mockSender,
			dbClient:                 db,
			entityTableName:          entityTableName,
			operationContainerClient: operationContainerClient,
		}

		query = fmt.Sprintf(entityTableUpdateQuery, s.entityTableName)
	})

	AfterEach(func() {
		err := mockDb.ExpectationsWereMet()
		Expect(err).To(BeNil())
		db.Close()
		ctrl.Finish()
	})

	Context("async operations", func() {
		It("should return operationId and insert new operation into database", func() {
			protoExpirationTime := timestamppb.New(time.Now().Add(1 * time.Hour))
			in := &pb.StartLongRunningOperationRequest{
				EntityId:            "test",
				EntityType:          "test",
				ExpirationTimestamp: protoExpirationTime,
			}
			initialOperationStatus := oc.Status_PENDING.String()

			// Need to use AnyString{} since we only care that it's a string, not really the values of it.
			mockDb.ExpectExec(query).WithArgs(in.GetEntityId(), in.GetEntityType(), AnyString{}, operations.LroName, initialOperationStatus).WillReturnResult(sqlmock.NewResult(1, 1))

			operationContainerClient.EXPECT().CreateOperationStatus(gomock.Any(), gomock.Any()).Return(nil, nil)
			mockSender.EXPECT().SendMessage(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			out, err := s.StartLongRunningOperation(context.Background(), in)
			Expect(err).To(BeNil())
			Expect(out.OperationId).NotTo(BeNil())
		})
		It("should fail on sender failure", func() {
			protoExpirationTime := timestamppb.New(time.Now().Add(1 * time.Hour))
			in := &pb.StartLongRunningOperationRequest{
				EntityId:            "test",
				EntityType:          "test",
				ExpirationTimestamp: protoExpirationTime,
			}

			errorMessage := "ServiceBus Sender error"
			err := errors.New(errorMessage)
			mockSender.EXPECT().SendMessage(gomock.Any(), gomock.Any()).Return(err).Times(1)
			out, err := s.StartLongRunningOperation(context.Background(), in)
			Expect(err).ToNot(BeNil())
			Expect(out).To(BeNil())
			Expect(err.Error()).To(ContainSubstring(errorMessage))
		})
		It("should call OperationContainer to update operation database on entity database query failure", func() {
			protoExpirationTime := timestamppb.New(time.Now().Add(1 * time.Hour))
			in := &pb.StartLongRunningOperationRequest{
				EntityId:            "test",
				EntityType:          "test",
				ExpirationTimestamp: protoExpirationTime,
			}
			initialOperationStatus := oc.Status_PENDING.String()

			mockSender.EXPECT().SendMessage(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			operationContainerClient.EXPECT().CreateOperationStatus(gomock.Any(), gomock.Any()).Return(nil, nil)

			// Need to use AnyString{} since we only care that it's a string, not really the values of it.
			errorMessage := "Database failure error."
			dbErr := errors.New(errorMessage)
			mockDb.ExpectExec(query).WithArgs(in.GetEntityId(), in.GetEntityType(), AnyString{}, operations.LroName, initialOperationStatus).WillReturnError(dbErr)

			operationContainerClient.EXPECT().UpdateOperationStatus(gomock.Any(), gomock.Any()).Return(nil, nil)
			_, err := s.StartLongRunningOperation(context.Background(), in)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(errorMessage))
		})

		It("should throw an error if update operations database fails after entity database update failure", func() {
			protoExpirationTime := timestamppb.New(time.Now().Add(1 * time.Hour))
			in := &pb.StartLongRunningOperationRequest{
				EntityId:            "20",
				EntityType:          "Cluster",
				ExpirationTimestamp: protoExpirationTime,
			}

			mockSender.EXPECT().SendMessage(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			operationContainerClient.EXPECT().CreateOperationStatus(gomock.Any(), gomock.Any()).Return(nil, nil)

			initialOperationStatus := oc.Status_PENDING.String()

			dbErrorMessage := "Database failure error."
			dbErr := errors.New(dbErrorMessage)
			// Need to use AnyString{} since we only care that it's a string, not really the values of it.
			mockDb.ExpectExec(query).WithArgs(in.GetEntityId(), in.GetEntityType(), AnyString{}, operations.LroName, initialOperationStatus).WillReturnError(dbErr)

			ocErrorMessage := "Something went wrong with OperationContainer!"
			ocErr := errors.New(ocErrorMessage)
			operationContainerClient.EXPECT().UpdateOperationStatus(gomock.Any(), gomock.Any()).Return(nil, ocErr)

			out, err := s.StartLongRunningOperation(context.Background(), in)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring(dbErrorMessage))
			Expect(err.Error()).To(ContainSubstring(ocErrorMessage))
			Expect(out).To(BeNil())
		})

	})
})

var _ = Describe("Fakes Testing", func() {
	var (
		s                        *Server
		db                       *sql.DB
		mockDb                   sqlmock.Sqlmock
		entityTableName          string
		ctrl                     *gomock.Controller
		operationContainerClient *ocMock.MockOperationContainerClient
		query                    string
		ctx                      context.Context
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		ctx = context.TODO()
		sbClient := servicebus.NewFakeServiceBusClient()
		sbSender, _ := sbClient.NewServiceBusSender(nil, "", nil)
		db, mockDb, _ = sqlmock.New()
		entityTableName = "hcp"
		operationContainerClient = ocMock.NewMockOperationContainerClient(ctrl)
		s = &Server{
			ResourceGroupClient:      nil,
			serviceBusClient:         sbClient,
			serviceBusSender:         sbSender,
			dbClient:                 db,
			entityTableName:          entityTableName,
			operationContainerClient: operationContainerClient,
		}

		query = fmt.Sprintf(entityTableUpdateQuery, s.entityTableName)

	})

	AfterEach(func() {
		err := mockDb.ExpectationsWereMet()
		Expect(err).To(BeNil())
	})

	Context("Message should exist in the service bus", func() {
		It("should send the message successfully", func() {
			protoExpirationTime := timestamppb.New(time.Now().Add(1 * time.Hour))
			initialOperationStatus := oc.Status_PENDING.String()
			in := &pb.StartLongRunningOperationRequest{
				EntityId:            "test",
				EntityType:          "test",
				ExpirationTimestamp: protoExpirationTime,
			}

			operationContainerClient.EXPECT().CreateOperationStatus(gomock.Any(), gomock.Any()).Return(nil, nil)
			// Can mostly ignore this since we tested it above. We care more about the service bus mock in this test.
			// Need to add it because otherwise the server call will complain froma null pointer to try and access the db
			// if it doesn't exist in the test.
			mockDb.ExpectExec(query).WithArgs(in.GetEntityId(), in.GetEntityType(), AnyString{}, operations.LroName, initialOperationStatus).WillReturnResult(sqlmock.NewResult(1, 1))

			_, err := s.StartLongRunningOperation(context.Background(), in)
			Expect(err).ToNot(HaveOccurred())

			sbReceiver, _ := s.serviceBusClient.NewServiceBusReceiver(nil, "", nil)
			msg, err := sbReceiver.ReceiveMessage(ctx, 1, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(msg).NotTo(BeNil())

			opRequestExpected := &operation.OperationRequest{
				OperationName:       "LongRunningOperation",
				ApiVersion:          "",
				OperationId:         "",
				EntityId:            "test",
				EntityType:          "test",
				RetryCount:          0,
				ExpirationTimestamp: protoExpirationTime,
				Body:                nil,
				HttpMethod:          "",
				Extension:           nil,
			}
			// .NewOperationRequest("LongRunningOperation", "", "", "test", "test", 0, protoExpirationTime, nil, "", nil)

			marshaller := &shuttle.DefaultProtoMarshaller{}
			var opRequestReceived operation.OperationRequest
			err = marshaller.Unmarshal(msg[0].Message(), &opRequestReceived)
			Expect(err).ToNot(HaveOccurred())

			Expect(opRequestReceived.OperationName).To(Equal(opRequestExpected.OperationName))
			Expect(opRequestReceived.OperationId).NotTo(BeNil())
			Expect(opRequestReceived.RetryCount).To(Equal(opRequestExpected.RetryCount))
			Expect(opRequestReceived.EntityType).To(Equal(opRequestExpected.EntityType))
			Expect(opRequestReceived.EntityId).To(Equal(opRequestExpected.EntityId))
			Expect(proto.Equal(opRequestReceived.ExpirationTimestamp, opRequestExpected.ExpirationTimestamp)).To(BeTrue())
			Expect(opRequestReceived.ApiVersion).To(Equal(opRequestExpected.ApiVersion))
		})
	})
})
