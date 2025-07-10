package server

import (
	"context"
	"fmt"
	"strings"

	database "github.com/Azure/aks-async/database"
	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "dev.azure.com/service-hub-flg/service_hub_validation/_git/service_hub_validation_service.git/mygreeterv3/api/v1"
	oc "github.com/Azure/OperationContainer/api/v1"
)

const (
	CancelOperationStatusQuery = `
    MERGE INTO %s AS target
    USING (SELECT @p1 AS entity_id, @p2 AS entity_type, @p3 AS operation_status) AS source
    ON target.entity_id = source.entity_id AND target.entity_type = source.entity_type
    WHEN MATCHED AND target.operation_status = '%s' THEN
      UPDATE SET
        target.operation_status = source.operation_status;`
)

// CancelOperation can be used to set the status of an operation to cancelled, particularly in the case an operation
// is stuck in a non-terminal state.
func (s *Server) CancelOperation(ctx context.Context, in *pb.CancelOperationRequest) (*emptypb.Empty, error) {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Cancelling operation.")

	cancelledOperationStatus := oc.Status_CANCELED
	//TODO(mheberling): Add a way to cancel the operation manually mid flight from the processor as well if state is IN_PROGRESS.
	// Update entity database first, because OC updates don't check for the current status of the operation,
	// and we only want to update to cancelled if the operation is stuck in a non-terminal state
	// (PENDING) after a premature panic or cancellation, leaving it in such state.
	// Whereas in this query in the entity database we do check that the operation was indeed non-terminal.
	logger.Info("Updating entity database.")
	query := fmt.Sprintf(CancelOperationStatusQuery, s.entityTableName, oc.Status_PENDING.String())
	_, err := database.ExecDb(ctx, s.dbClient, query, in.GetEntityId(), in.GetEntityType(), cancelledOperationStatus.String())

	// If no rows were affected, the ExecDb function will throw an error saying `No rows were affected!`. With this error we could
	// conclude that the query run successfully but didn't insert the record due to not meeting the match criterium.
	if err != nil {
		//TODO(mheberling): Change this to return a known type of error in aks-async/database, instead of errors.New(...)
		if strings.Contains(err.Error(), "No rows were affected!") {
			errorMessage := "The combination of entityId " + in.GetEntityId() + " and entityType " + in.GetEntityType() + " was found in a finalized state. Entity was not updated to Cancelled. Error: " + err.Error()
			logger.Error(errorMessage)
			return nil, status.Error(codes.FailedPrecondition, errorMessage)
		} else {
			logger.Error("Error in entity update query: " + err.Error() + ".  Query options were: OperationId: " + in.GetOperationId() + ", EntityId: " + in.GetEntityId() + ", EntityType: " + in.GetEntityType())
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	logger.Info("Updating operation with id " + in.GetOperationId() + " to cancelled in operations database.")
	updateOperationStatusRequest := &oc.UpdateOperationStatusRequest{
		OperationId: in.GetOperationId(),
		Status:      cancelledOperationStatus,
	}

	//TODO(mheberling): Add a poller function that will periodically check for the state of the operation.
	logger.Info("Cancelling operation in OperationContainer")
	_, err = s.operationContainerClient.UpdateOperationStatus(ctx, updateOperationStatusRequest)
	if err != nil {
		logger.Error("Error updating operation status with operationId" + in.GetOperationId() + ": " + err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}

	return nil, nil
}
