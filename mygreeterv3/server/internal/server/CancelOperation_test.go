package server

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"fmt"

	pb "dev.azure.com/service-hub-flg/service_hub_validation/_git/service_hub_validation_service.git/mygreeterv3/api/v1"

	oc "github.com/Azure/OperationContainer/api/v1"
	ocMock "github.com/Azure/OperationContainer/api/v1/mock"
	"github.com/DATA-DOG/go-sqlmock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

// Escapes parentheses and other regex-special characters used for matching the query in go-sqlmock
func EscapeRegexSpecials(query string) string {
	replacer := strings.NewReplacer(
		"(", `\(`,
		")", `\)`,
		".", `\.`,
		"+", `\+`,
		"*", `\*`,
		"?", `\?`,
		"|", `\|`,
		"^", `\^`,
		"$", `\$`,
		"[", `\[`,
		"]", `\]`,
		"{", `\{`,
		"}", `\}`,
	)
	return replacer.Replace(query)
}

var _ = Describe("Mock Testing for CancelOperation", func() {
	var (
		ctrl                     *gomock.Controller
		s                        *Server
		db                       *sql.DB
		mockDb                   sqlmock.Sqlmock
		entityTableName          string
		query                    string
		operationContainerClient *ocMock.MockOperationContainerClient
		in                       *pb.CancelOperationRequest
		canceledOperationStatus  string
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		db, mockDb, _ = sqlmock.New()
		entityTableName = "hcp"
		operationContainerClient = ocMock.NewMockOperationContainerClient(ctrl)
		s = &Server{
			dbClient:                 db,
			entityTableName:          entityTableName,
			operationContainerClient: operationContainerClient,
		}

		in = &pb.CancelOperationRequest{
			EntityId:    "test",
			EntityType:  "test",
			OperationId: "test",
		}

		canceledOperationStatus = oc.Status_CANCELED.String()

		query = fmt.Sprintf(EscapeRegexSpecials(CancelOperationStatusQuery), s.entityTableName, oc.Status_PENDING.String())
	})

	AfterEach(func() {
		err := mockDb.ExpectationsWereMet()
		Expect(err).To(BeNil())
		db.Close()
		ctrl.Finish()
	})

	Context("cancelling operations", func() {
		It("should succeed", func() {

			// Need to use AnyString{} since we only care that it's a string, not really the values of it.
			mockDb.ExpectExec(query).WithArgs(in.GetEntityId(), in.GetEntityType(), canceledOperationStatus).WillReturnResult(sqlmock.NewResult(1, 1))

			operationContainerClient.EXPECT().UpdateOperationStatus(gomock.Any(), gomock.Any()).Return(nil, nil)
			_, err := s.CancelOperation(context.Background(), in)
			Expect(err).To(BeNil())
		})
		It("should fail on OperationContainer failure", func() {

			mockDb.ExpectExec(query).WithArgs(in.GetEntityId(), in.GetEntityType(), canceledOperationStatus).WillReturnResult(sqlmock.NewResult(1, 1))

			errorMessage := "OperationContainer error"
			ocErr := errors.New(errorMessage)
			operationContainerClient.EXPECT().UpdateOperationStatus(gomock.Any(), gomock.Any()).Return(nil, ocErr)

			_, err := s.CancelOperation(context.Background(), in)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(errorMessage))
		})
		It("should fail on entity database query failure", func() {

			errorMessage := "Database error"
			dbErr := errors.New(errorMessage)
			mockDb.ExpectExec(query).WithArgs(in.GetEntityId(), in.GetEntityType(), canceledOperationStatus).WillReturnError(dbErr)

			_, err := s.CancelOperation(context.Background(), in)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(errorMessage))
		})
		It("should fail on entity database no rows affected", func() {

			mockDb.ExpectExec(query).WithArgs(in.GetEntityId(), in.GetEntityType(), canceledOperationStatus).WillReturnResult(sqlmock.NewResult(0, 0))

			_, err := s.CancelOperation(context.Background(), in)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("no rows were affected!"))
		})
	})
})
