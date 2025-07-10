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

func (s *Server) UpdateOperationStatus(ctx context.Context, in *pb.UpdateOperationStatusRequest) (*emptypb.Empty, error) {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Updating Operation Status.")
	logger.Info("OperationId: " + in.GetOperationId())
	logger.Info("Status: " + in.GetStatus().String())

	// Update operations table.
	// If the query didn't change any rows, then the ExecDb method will throw an error to let you know. You can
	// also inspect the first return value of the database.ExecDb function to view the details on the result of the query.
	// In said case, the user will have to first run CreateOperationStatus to create the record of the operation, and then
	// try again to update the status.
	updateOperationsQuery := fmt.Sprintf("UPDATE %s SET operation_status = @p1 WHERE operation_id = @p2 AND EXISTS (SELECT 1 FROM %s WHERE operation_id = @p2)", s.operationTableName, s.operationTableName)
	_, err := database.ExecDb(ctx, s.dbClient, updateOperationsQuery, in.GetStatus().String(), in.GetOperationId())
	if err != nil {
		logger.Error("Error executing query: " + err.Error())
		//TODO(mheberling): Change this to return a known type of error in aks-async/database, instead of errors.New(...)
		if strings.Index(err.Error(), "No rows were affected!") == -1 {
			return nil, status.Error(codes.NotFound, err.Error())
		} else {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return nil, nil
}
