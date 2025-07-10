package async

import (
	"context"
	"database/sql"
	"fmt"

	oc "github.com/Azure/OperationContainer/api/v1"
	database "github.com/Azure/aks-async/database"
	"github.com/Azure/aks-async/runtime/entity"
	asyncErrors "github.com/Azure/aks-async/runtime/errors"
	"github.com/Azure/aks-async/runtime/hooks"
	"github.com/Azure/aks-async/runtime/operation"
)

var _ hooks.BaseOperationHooksInterface = &OperationStatusHook{}

// TODO(mheberling): Move this behavior to aks-async.
// OperationContainer is already taken care of here because we passed it in to the processor. We only need to update the entity table here.
type OperationStatusHook struct {
	dbClient        *sql.DB
	EntityTableName string
}

func (h *OperationStatusHook) BeforeInitOperation(ctx context.Context, req *operation.OperationRequest) *asyncErrors.AsyncError {
	// set operation as in in progress
	inProgressOperationStatus := oc.Status_IN_PROGRESS.String()
	err := h.updateEntityDatabase(ctx, inProgressOperationStatus, req.EntityId, req.EntityType, req.OperationId)
	if err != nil {
		return &asyncErrors.AsyncError{
			OriginalError: err,
			Message:       fmt.Sprintf("Error updating the entity database of entity with id: %s and type: %s to IN_PROGRESS status: %s", req.EntityId, req.EntityType, err.Error()),
		}
	}

	return nil
}

func (h *OperationStatusHook) AfterInitOperation(ctx context.Context, op operation.ApiOperation, req *operation.OperationRequest, err *asyncErrors.AsyncError) *asyncErrors.AsyncError {
	// on error set as pending
	if err != nil {
		pendingOperationStatus := oc.Status_PENDING.String()
		updateErr := h.updateEntityDatabase(ctx, pendingOperationStatus, req.EntityId, req.EntityType, req.OperationId)
		if updateErr != nil {
			return &asyncErrors.AsyncError{
				OriginalError: err,
				Message:       fmt.Sprintf("Error updating the entity database of entity with id: %s and type: %s to PENDING status: %s", req.EntityId, req.EntityType, updateErr.Error()),
			}
		}
	}
	return nil
}

func (h *OperationStatusHook) BeforeGuardConcurrency(ctx context.Context, op operation.ApiOperation, entity entity.Entity) *asyncErrors.AsyncError {
	// If there was an error with any function before getting here, it would've been caught. Nothing to do here.
	return nil
}

func (h *OperationStatusHook) AfterGuardConcurrency(ctx context.Context, op operation.ApiOperation, asyncErr *asyncErrors.AsyncError) *asyncErrors.AsyncError {
	// on error set as pending
	if asyncErr != nil {
		req := op.GetOperationRequest()
		pendingOperationStatus := oc.Status_PENDING.String()
		err := h.updateEntityDatabase(ctx, pendingOperationStatus, req.EntityId, req.EntityType, req.OperationId)
		if err != nil {
			return &asyncErrors.AsyncError{
				OriginalError: asyncErr,
				Message:       fmt.Sprintf("Error updating the entity database of entity with id: %s and type: %s to PENDING status: %s", req.EntityId, req.EntityType, err.Error()),
			}
		}
	}
	return nil
}

func (h *OperationStatusHook) BeforeRun(ctx context.Context, op operation.ApiOperation) *asyncErrors.AsyncError {
	// If there was an error with any function before getting here, it would've been caught. Nothing to do here.
	return nil
}

func (h *OperationStatusHook) AfterRun(ctx context.Context, op operation.ApiOperation, err *asyncErrors.AsyncError) *asyncErrors.AsyncError {
	req := op.GetOperationRequest()
	if err != nil {
		// on error set as pending
		pendingOperationStatus := oc.Status_PENDING.String()
		updateErr := h.updateEntityDatabase(ctx, pendingOperationStatus, req.EntityId, req.EntityType, req.OperationId)
		if updateErr != nil {
			return &asyncErrors.AsyncError{
				OriginalError: err,
				Message:       fmt.Sprintf("Error updating the entity database of entity with id: %s and type: %s to PENDING status: %s", req.EntityId, req.EntityType, updateErr.Error()),
			}
		}
	} else {
		// no nil error set as complete
		inProgressOperationStatus := oc.Status_SUCCEEDED.String()
		updateErr := h.updateEntityDatabase(ctx, inProgressOperationStatus, req.EntityId, req.EntityType, req.OperationId)
		if updateErr != nil {
			return &asyncErrors.AsyncError{
				OriginalError: updateErr,
				Message:       fmt.Sprintf("Error updating the entity database of entity with id: %s and type: %s to SUCCEEDED status: %s", req.EntityId, req.EntityType, updateErr.Error()),
			}
		}
	}
	return nil
}

func (h *OperationStatusHook) updateEntityDatabase(ctx context.Context, newStatus string, entityId string, entityType string, operationId string) error {
	query := fmt.Sprintf(`UPDATE %s SET operation_status = @p1 WHERE entity_id = @p2 AND entity_type = @p3 AND last_operation_id = @p4 AND EXISTS (SELECT 1 FROM %s WHERE entity_id = @p2 AND entity_type = @p3 AND last_operation_id = @p4)`, h.EntityTableName, h.EntityTableName)
	_, err := database.ExecDb(ctx, h.dbClient, query, newStatus, entityId, entityType, operationId)
	if err != nil {
		return fmt.Errorf("Error in operations query: %w", err)
	}
	return nil
}
