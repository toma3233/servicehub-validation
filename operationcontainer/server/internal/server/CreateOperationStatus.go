package server

import (
	"context"
	"fmt"
	"strings"

	pb "github.com/Azure/OperationContainer/api/v1"
	database "github.com/Azure/aks-async/database"
	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *Server) CreateOperationStatus(ctx context.Context, in *pb.CreateOperationStatusRequest) (*emptypb.Empty, error) {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Creating operation status.")
	logger.Info("Operation Name: " + in.GetOperationName())
	logger.Info("Entity id: " + in.GetEntityId())
	logger.Info("OperationId: " + in.GetOperationId())

	// This query checks that the operationId doesn't already exists and inserts it into the table if successful all in a single query to avoid race conditions.
	query := fmt.Sprintf("INSERT INTO %s (operation_id, operation_name, operation_status, entity_id) SELECT @p1, @p2, @p3, @p4 WHERE NOT EXISTS (SELECT 1 FROM %s WHERE operation_id = @p1)", s.operationTableName, s.operationTableName)
	initialOperationStatus := pb.Status_PENDING.String()
	_, err := database.ExecDb(ctx, s.dbClient, query, in.GetOperationId(), in.GetOperationName(), initialOperationStatus, in.GetEntityId())

	// If no rows were affected, the ExecDb function will throw an error saying `No rows were affected!`. With this error we could
	// conclude that the query run successfully but didn't insert the record due to another record already existing with the same
	// operation_id.
	if err != nil {
		logger.Error("Error in operations query: " + err.Error())
		//TODO(mheberling): Change this to return a known type of error in aks-async/database, instead of errors.New(...)
		if strings.Index(err.Error(), "No rows were affected!") == -1 {
			return nil, status.Error(codes.NotFound, err.Error())
		} else {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return nil, nil
}
