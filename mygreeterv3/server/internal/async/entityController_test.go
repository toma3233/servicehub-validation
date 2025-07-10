package async

import (
	"context"
	"database/sql"
	"fmt"

	"dev.azure.com/service-hub-flg/service_hub_validation/_git/service_hub_validation_service.git/mygreeterv3/server/internal/async/operations"
	"dev.azure.com/service-hub-flg/service_hub_validation/_git/service_hub_validation_service.git/mygreeterv3/server/internal/async/operations/longRunningOperation"
	"github.com/Azure/aks-async/runtime/matcher"
	"github.com/Azure/aks-async/runtime/operation"
	"github.com/DATA-DOG/go-sqlmock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("EntityController Testing", func() {
	Context("Integration tests", func() {
		var (
			ctrl     *gomock.Controller
			ctx      context.Context
			options  Options
			dbClient *sql.DB
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			ctx = context.Background()

			dbClient, _, _ = sqlmock.New()
			// Can only test the connection string, since testcontainers-go doesn't allow to retrieve
			// ServerUrl or ServerName.
			options = Options{
				EntityTableName: "hcp",
			}
		})

		AfterEach(func() {
			ctrl.Finish()
		})
		It("should fail if no EntityTableName is provided", func() {

			options.EntityTableName = ""
			_, err := NewEntityController(ctx, options, nil, nil)
			Expect(err).ToNot(BeNil())
		})
		It("should fail if no matcher is provided", func() {

			_, err := NewEntityController(ctx, options, nil, nil)
			Expect(err).ToNot(BeNil())
		})
		It("should fail if no dbClient is provided", func() {

			matcher := matcher.NewMatcher()
			_, err := NewEntityController(ctx, options, matcher, nil)
			Expect(err).ToNot(BeNil())
		})
		It("should successfully return an entity controller", func() {

			m := matcher.NewMatcher()
			_, err := NewEntityController(ctx, options, m, dbClient)
			Expect(err).To(BeNil())
		})
	})

	Context("Query tests", func() {
		var (
			ctrl             *gomock.Controller
			ctx              context.Context
			db               *sql.DB
			mockDb           sqlmock.Sqlmock
			entityTableName  string
			options          Options
			entityId         string
			lastOperationId  string
			entityController *EntityController
			opReq            *operation.OperationRequest
			m                *matcher.Matcher
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			ctx = context.Background()
			db, mockDb, _ = sqlmock.New()
			entityTableName = "hcp"
			entityId = "0"
			lastOperationId = "0"
			options = Options{
				DatabaseConnectionString: "test",
				DatabasePort:             1433,
				DatabaseServerUrl:        "test",
				DatabaseName:             "test",
				EntityTableName:          entityTableName,
			}
			opReq = &operation.OperationRequest{
				OperationName: operations.LroName,
				EntityId:      entityId,
			}

			m = matcher.NewMatcher()
			lro := &longRunningOperation.LongRunningOperation{}
			m.Register(ctx, operations.LroName, lro)
			m.RegisterEntity(ctx, operations.LroName, longRunningOperation.CreateLroEntityFunc)

			entityController = &EntityController{
				dbClient:        db,
				entityTableName: options.EntityTableName,
				matcher:         m,
			}
		})

		AfterEach(func() {
			err := mockDb.ExpectationsWereMet()
			Expect(err).To(BeNil())

			db.Close()
			ctrl.Finish()
		})
		It("should get entity from entity_id", func() {

			queryEntity := fmt.Sprintf("SELECT last_operation_id FROM %s WHERE entity_id = @p1", entityController.entityTableName)

			expectedRows := sqlmock.NewRows([]string{"last_operation_id"})
			expectedRows.AddRow(lastOperationId)
			mockDb.ExpectQuery(queryEntity).WithArgs(entityId).WillReturnRows(expectedRows)
			entity, asyncErr := entityController.GetEntity(ctx, opReq)

			Expect(asyncErr).To(BeNil())
			Expect(entity.GetLatestOperationID()).To(Equal(lastOperationId))
		})
		It("should fail if entity doesn't exist", func() {

			queryEntity := fmt.Sprintf("SELECT last_operation_id FROM %s WHERE entity_id = @p1", entityController.entityTableName)

			expectedRows := sqlmock.NewRows([]string{"last_operation_id"})
			mockDb.ExpectQuery(queryEntity).WithArgs(entityId).WillReturnRows(expectedRows)
			entity, asyncErr := entityController.GetEntity(ctx, opReq)

			Expect(asyncErr).ToNot(BeNil())
			Expect(entity).To(BeNil())
		})
		It("should fail if operation name doesn't exist in matcher", func() {
			entityController.matcher = matcher.NewMatcher()

			queryEntity := fmt.Sprintf("SELECT last_operation_id FROM %s WHERE entity_id = @p1", entityController.entityTableName)

			expectedRows := sqlmock.NewRows([]string{"last_operation_id"})
			mockDb.ExpectQuery(queryEntity).WithArgs(entityId).WillReturnRows(expectedRows)
			entity, asyncErr := entityController.GetEntity(ctx, opReq)

			Expect(asyncErr).ToNot(BeNil())
			Expect(entity).To(BeNil())
		})
	})
})
