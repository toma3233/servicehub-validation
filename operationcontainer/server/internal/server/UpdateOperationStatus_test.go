package server

import (
	"context"
	"database/sql"

	// "time"

	pb "github.com/Azure/OperationContainer/api/v1"
	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
	// "google.golang.org/protobuf/types/known/timestamppb"
)

var _ = Describe("Update Operation Status Tests", func() {
	var (
		ctrl *gomock.Controller
		s    *Server
		ctx  context.Context

		operationId string
		db          *sql.DB
		mockDb      sqlmock.Sqlmock
		// completedStatus string
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		ctx = context.Background()
		operationId = "1"

		db, mockDb, _ = sqlmock.New()
		s = &Server{dbClient: db, operationTableName: "operations"}
	})

	AfterEach(func() {
		db.Close()
		ctrl.Finish()
	})

	Describe("UpdateOperationStatus", func() {
		It("should successfully update the status of an operation.", func() {
			newStatus := pb.Status_SUCCEEDED
			updateOperationStatusRequest := &pb.UpdateOperationStatusRequest{
				OperationId: operationId,
				Status:      newStatus,
			}

			operationExpectedRows := sqlmock.NewRows([]string{"OperationID"})
			operationExpectedRows.AddRow("1")

			mockDb.ExpectExec(`UPDATE operations SET operation_status = @p1 WHERE operation_id = @p2 AND EXISTS \(SELECT 1 FROM operations WHERE operation_id = @p2\)`).WithArgs(newStatus.String(), operationId).WillReturnResult(sqlmock.NewResult(1, 1))
			_, err := s.UpdateOperationStatus(ctx, updateOperationStatusRequest)
			Expect(err).To(BeNil())

			err = mockDb.ExpectationsWereMet()
			Expect(err).To(BeNil())
		})

		It("should fail if the operation does not exist in Operations.", func() {
			newStatus := pb.Status_SUCCEEDED
			updateOperationStatusRequest := &pb.UpdateOperationStatusRequest{
				OperationId: operationId,
				Status:      newStatus,
			}

			mockDb.ExpectExec(`UPDATE operations SET operation_status = @p1 WHERE operation_id = @p2 AND EXISTS \(SELECT 1 FROM operations WHERE operation_id = @p2\)`).WithArgs(newStatus.String(), operationId).WillReturnResult(sqlmock.NewResult(0, 0))
			_, err := s.UpdateOperationStatus(ctx, updateOperationStatusRequest)
			Expect(err).To(HaveOccurred())

			err = mockDb.ExpectationsWereMet()
			Expect(err).To(BeNil())
		})
	})
})
