package async

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Azure/aks-async/database"
	// opbus "github.com/Azure/aks-async/operationsbus"
	"github.com/Azure/aks-async/runtime/entity"
	"github.com/Azure/aks-async/runtime/entity_controller"
	asyncErrors "github.com/Azure/aks-async/runtime/errors"
	"github.com/Azure/aks-async/runtime/matcher"
	"github.com/Azure/aks-async/runtime/operation"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
)

// type EntityFactoryFunc func(string) e.Entity

var _ entity_controller.EntityController = &EntityController{}

type EntityController struct {
	dbClient        *sql.DB
	entityTableName string
	matcher         *matcher.Matcher
}

func NewEntityController(ctx context.Context, options Options, matcher *matcher.Matcher, dbClient *sql.DB) (*EntityController, error) {
	logger := ctxlogger.GetLogger(ctx)

	if options.EntityTableName == "" {
		logger.Error("No EntityTableName provided.")
		return nil, errors.New("No EntityTableName provided.")
	}

	if matcher == nil {
		logger.Error("No matcher provided.")
		return nil, errors.New("No matcher provided.")
	}

	if dbClient == nil {
		logger.Error("No dbClient provided.")
		return nil, errors.New("No dbClient provided.")
	}

	newEntityController := &EntityController{
		dbClient:        dbClient,
		entityTableName: options.EntityTableName,
		matcher:         matcher,
	}

	return newEntityController, nil
}

func (e *EntityController) GetEntity(ctx context.Context, opReq *operation.OperationRequest) (entity.Entity, *asyncErrors.AsyncError) {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Getting entity with id: " + opReq.EntityId)

	queryEntity := fmt.Sprintf("SELECT last_operation_id FROM %s WHERE entity_id = @p1", e.entityTableName)
	rows, err := database.QueryDb(ctx, e.dbClient, queryEntity, opReq.EntityId)
	if err != nil {
		logger.Error("Error executing query: " + err.Error())
		return nil, &asyncErrors.AsyncError{
			OriginalError: status.Error(codes.Internal, err.Error()),
			Message:       "GetEntity Error",
		}
	}
	defer rows.Close()

	var lastOperationId string
	if rows.Next() {
		err := rows.Scan(&lastOperationId)
		if err != nil {
			logger.Info("Error scanning row: " + err.Error())
			return nil, &asyncErrors.AsyncError{
				OriginalError: status.Error(codes.Internal, err.Error()),
				Message:       "GetEntity Error",
			}
		}
	} else {
		logger.Error("No rows returned for entityId: " + opReq.EntityId)
		return nil, &asyncErrors.AsyncError{
			OriginalError: status.Error(codes.NotFound, "EntityId not found in database."),
			Message:       "GetEntity Error",
		}
	}

	entity, err := e.matcher.CreateEntityInstance(ctx, opReq.OperationName, lastOperationId)
	if err != nil {
		logger.Error("Error creating the entity instance: " + err.Error())
		return nil, &asyncErrors.AsyncError{
			OriginalError: status.Error(codes.Internal, err.Error()),
			Message:       "GetEntity Error",
		}
	}

	return entity, nil
}
