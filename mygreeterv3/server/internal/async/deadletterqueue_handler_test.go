package async

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	log "log/slog"
	"strings"

	oc "github.com/Azure/OperationContainer/api/v1"
	ocMock "github.com/Azure/OperationContainer/api/v1/mock"
	"github.com/Azure/aks-async/runtime/operation"
	"github.com/Azure/aks-async/runtime/testutils/toolkit/convert"
	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/go-shuttle/v2"
	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("DeadLetterQueueHandler", func() {
	var (
		ctrl                         *gomock.Controller
		ctx                          context.Context
		buf                          bytes.Buffer
		operationContainerClient     *ocMock.MockOperationContainerClient
		db                           *sql.DB
		mockDb                       sqlmock.Sqlmock
		options                      Options
		settler                      shuttle.MessageSettler
		message                      *azservicebus.ReceivedMessage
		handler                      shuttle.HandlerFunc
		query                        string
		req                          *operation.OperationRequest
		failedOperationStatus        oc.Status
		updateOperationStatusRequest *oc.UpdateOperationStatusRequest
	)

	BeforeEach(func() {
		buf.Reset()
		ctrl = gomock.NewController(GinkgoT())
		logger := log.New(log.NewTextHandler(&buf, nil))
		ctx = context.TODO()
		ctx = ctxlogger.WithLogger(ctx, logger)
		operationContainerClient = ocMock.NewMockOperationContainerClient(ctrl)
		db, mockDb, _ = sqlmock.New()

		settler = &fakeMessageSettler{}
		options = Options{
			EntityTableName: "test_entity_table_name",
		}
		failedOperationStatus = oc.Status_FAILED
		req = &operation.OperationRequest{
			EntityId:    "test_entity_id",
			EntityType:  "test_entity_type",
			OperationId: "test_operation_id",
		}
		marshaller := &shuttle.DefaultProtoMarshaller{}
		marshalledOperation, err := marshaller.Marshal(req)
		message = convert.ConvertToReceivedMessage(marshalledOperation)
		if err != nil {
			return
		}
		updateOperationStatusRequest = &oc.UpdateOperationStatusRequest{
			OperationId: req.OperationId,
			Status:      failedOperationStatus,
		}
		query = fmt.Sprintf(`UPDATE %s SET operation_status = @p1 WHERE entity_id = @p2 AND entity_type = @p3 AND last_operation_id = @p4\;`, options.EntityTableName)
		handler = NewDeadLetterQueueHandler(options, operationContainerClient, db, marshaller)
	})

	Context("mock testing", func() {
		It("should handle the dead-letter queue message correctly", func() {
			mockDb.ExpectExec(query).WithArgs(failedOperationStatus.String(), req.EntityId, req.EntityType, req.OperationId).WillReturnResult(sqlmock.NewResult(1, 1))
			operationContainerClient.EXPECT().UpdateOperationStatus(gomock.Any(), updateOperationStatusRequest).Return(nil, nil)
			handler(ctx, settler, message)
			Expect(strings.Count(buf.String(), "DeadLetterQueueHandler: Successfully set the operation")).To(Equal(1))
		})
		It("should throw an error if unmarshal fails", func() {
			message = &azservicebus.ReceivedMessage{
				Body: []byte(`invalid json`),
			}
			handler(ctx, settler, message)
			Expect(strings.Count(buf.String(), "Error unmarshalling message")).To(Equal(1))
		})
		It("should throw error if query fails", func() {
			err := errors.New("Sample error")
			mockDb.ExpectExec(query).WithArgs(failedOperationStatus.String(), req.EntityId, req.EntityType, req.OperationId).WillReturnResult(sqlmock.NewErrorResult(err))
			handler(ctx, settler, message)
			Expect(strings.Count(buf.String(), "DeadLetterQueueHandler: Error in entity table query")).To(Equal(1))
		})
		It("should throw error if OperationContainerClient fails", func() {
			err := errors.New("Sample error")
			mockDb.ExpectExec(query).WithArgs(failedOperationStatus.String(), req.EntityId, req.EntityType, req.OperationId).WillReturnResult(sqlmock.NewResult(1, 1))
			operationContainerClient.EXPECT().UpdateOperationStatus(gomock.Any(), updateOperationStatusRequest).Return(nil, err)
			handler(ctx, settler, message)
			Expect(strings.Count(buf.String(), "DeadLetterQueueHandler: Error setting operation")).To(Equal(1))
		})
		It("should throw error if settler fails", func() {
			contentType := "failure_test"
			message.ContentType = &contentType
			mockDb.ExpectExec(query).WithArgs(failedOperationStatus.String(), req.EntityId, req.EntityType, req.OperationId).WillReturnResult(sqlmock.NewResult(1, 1))
			operationContainerClient.EXPECT().UpdateOperationStatus(gomock.Any(), updateOperationStatusRequest).Return(nil, nil)
			handler(ctx, settler, message)
			Expect(strings.Count(buf.String(), "Unable to settle message")).To(Equal(1))
			Expect(strings.Count(buf.String(), "settler error")).To(Equal(1))
		})
	})
})

type fakeMessageSettler struct{}

func (f *fakeMessageSettler) AbandonMessage(ctx context.Context, message *azservicebus.ReceivedMessage, options *azservicebus.AbandonMessageOptions) error {
	return nil
}
func (f *fakeMessageSettler) CompleteMessage(ctx context.Context, message *azservicebus.ReceivedMessage, options *azservicebus.CompleteMessageOptions) error {
	failureMessage := "failure_test"
	if message.ContentType != nil && strings.Compare(*message.ContentType, failureMessage) == 0 {
		return errors.New("settler error")
	}
	return nil
}
func (f *fakeMessageSettler) DeadLetterMessage(ctx context.Context, message *azservicebus.ReceivedMessage, options *azservicebus.DeadLetterOptions) error {
	return nil
}
func (f *fakeMessageSettler) DeferMessage(ctx context.Context, message *azservicebus.ReceivedMessage, options *azservicebus.DeferMessageOptions) error {
	return nil
}
func (f *fakeMessageSettler) RenewMessageLock(ctx context.Context, message *azservicebus.ReceivedMessage, options *azservicebus.RenewMessageLockOptions) error {
	return nil
}
