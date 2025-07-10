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
)

func (s *Server) GetOperationStatus(ctx context.Context, in *pb.GetOperationStatusRequest) (*pb.GetOperationStatusResponse, error) {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Getting Operation status.")
	logger.Info("OperationId: " + in.GetOperationId())

	queryOperations := fmt.Sprintf("SELECT operation_status FROM %s WHERE operation_id = @p1", s.operationTableName)
	rows, err := database.QueryDb(ctx, s.dbClient, queryOperations, in.GetOperationId())
	if err != nil {
		logger.Error("Error executing query: " + err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}
	defer rows.Close()

	var operationStatus string
	if rows.Next() {
		err := rows.Scan(&operationStatus)
		if err != nil {
			logger.Info("Error scanning row: " + err.Error())
			return nil, status.Error(codes.Internal, err.Error())
		}
	} else {
		logger.Error("No rows returned for OperationId: " + in.GetOperationId())
		return nil, status.Error(codes.NotFound, "OperationId not found in database.")
	}

	getOperationStatusResponse := &pb.GetOperationStatusResponse{
		Status: pb.Status(pb.Status_value[strings.ToUpper(operationStatus)]),
	}

	return getOperationStatusResponse, nil
}
