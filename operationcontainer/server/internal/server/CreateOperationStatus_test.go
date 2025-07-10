package server

import (
	"context"
	"database/sql"
	"time"

	pb "github.com/Azure/OperationContainer/api/v1"
	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ = Describe("Create Operation Status Tests", func() {
	var (
		ctrl *gomock.Controller
		s    *Server
		ctx  context.Context

		operationName       string
		entityId            string
		expirationTimestamp *timestamppb.Timestamp
		operationId         string
		db                  *sql.DB
		mockDb              sqlmock.Sqlmock
		pendingStatus       string
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		ctx = context.Background()
		operationName = "LongRunning"
		entityId = "1"
		expirationTimestamp = timestamppb.New(time.Now().Add(1 * time.Hour))
		operationId = "0"
		pendingStatus = "PENDING"

		db, mockDb, _ = sqlmock.New()
		s = &Server{dbClient: db, operationTableName: "operations"}
	})

	AfterEach(func() {
		db.Close()
		ctrl.Finish()
	})

	Describe("CreateOperationStatus", func() {
		It("should successfully create an operation status for a new operation.", func() {

			createOperationStatusRequest := &pb.CreateOperationStatusRequest{
				OperationName:       operationName,
				EntityId:            entityId,
				ExpirationTimestamp: expirationTimestamp,
				OperationId:         operationId,
			}
			expectedOperationId := "1"

			expectedRows := sqlmock.NewRows([]string{"OperationId"})
			expectedRows.AddRow(expectedOperationId)

			mockDb.ExpectExec(`INSERT INTO operations \(operation_id, operation_name, operation_status, entity_id\) SELECT @p1, @p2, @p3, @p4 WHERE NOT EXISTS \(SELECT 1 FROM operations WHERE operation_id = @p1\)`).WithArgs(operationId, operationName, pendingStatus, entityId).WillReturnResult(sqlmock.NewResult(1, 1))

			_, err := s.CreateOperationStatus(ctx, createOperationStatusRequest)
			Expect(err).ToNot(HaveOccurred())

			err = mockDb.ExpectationsWereMet()
			Expect(err).To(BeNil())
		})

		It("should fail if the operationId already exists.", func() {

			createOperationStatusRequest := &pb.CreateOperationStatusRequest{
				OperationName:       operationName,
				EntityId:            entityId,
				ExpirationTimestamp: expirationTimestamp,
				OperationId:         operationId,
			}

			expectedOperationId := operationId

			expectedRows := sqlmock.NewRows([]string{"OperationId"})
			expectedRows.AddRow(expectedOperationId)

			mockDb.ExpectExec(`INSERT INTO operations \(operation_id, operation_name, operation_status, entity_id\) SELECT @p1, @p2, @p3, @p4 WHERE NOT EXISTS \(SELECT 1 FROM operations WHERE operation_id = @p1\)`).WithArgs(operationId, operationName, pendingStatus, entityId).WillReturnResult(sqlmock.NewResult(0, 0))

			_, err := s.CreateOperationStatus(ctx, createOperationStatusRequest)
			Expect(err).To(HaveOccurred())

			err = mockDb.ExpectationsWereMet()
			Expect(err).To(BeNil())
		})
	})
})
