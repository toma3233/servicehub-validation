package server

import (
	"context"
	"database/sql"

	pb "github.com/Azure/OperationContainer/api/v1"
	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Get Operation Status Tests", func() {
	var (
		ctrl *gomock.Controller
		s    *Server
		ctx  context.Context

		operationId string
		db          *sql.DB
		mockDb      sqlmock.Sqlmock
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

	Describe("GetOperationStatus", func() {
		It("should successfully get the status of the operation.", func() {
			expectedStatus := "SUCCEEDED"
			getOperationStatusRequest := &pb.GetOperationStatusRequest{
				OperationId: operationId,
			}

			expectedRows := sqlmock.NewRows([]string{"OperationStatus"})
			expectedRows.AddRow(expectedStatus)
			mockDb.ExpectQuery("SELECT operation_status FROM operations WHERE operation_id = @p1").WithArgs(operationId).WillReturnRows(expectedRows)

			getOperationStatusResponse, err := s.GetOperationStatus(ctx, getOperationStatusRequest)
			Expect(err).To(BeNil())
			Expect(getOperationStatusResponse.GetStatus().String()).To(Equal(expectedStatus))

			err = mockDb.ExpectationsWereMet()
			Expect(err).To(BeNil())
		})
		It("should fail if the operation doesn't exist", func() {
			getOperationStatusRequest := &pb.GetOperationStatusRequest{
				OperationId: operationId,
			}

			expectedRows := sqlmock.NewRows([]string{"OperationStatus"})
			mockDb.ExpectQuery("SELECT operation_status FROM operations WHERE operation_id = @p1").WithArgs(operationId).WillReturnRows(expectedRows)

			_, err := s.GetOperationStatus(ctx, getOperationStatusRequest)
			Expect(err).NotTo(BeNil())

			err = mockDb.ExpectationsWereMet()
			Expect(err).To(BeNil())
		})
	})
})
