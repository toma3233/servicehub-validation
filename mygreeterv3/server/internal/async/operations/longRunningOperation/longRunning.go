package longRunningOperation

import (
	"context"
	"errors"
	"time"

	"github.com/Azure/aks-async/runtime/entity"
	asyncErrors "github.com/Azure/aks-async/runtime/errors"
	"github.com/Azure/aks-async/runtime/operation"
	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"

	"google.golang.org/grpc/codes"
)

// Setting the variable to ensure all functions of the ApiOperation interface are implemented.
var _ operation.ApiOperation = &LongRunningOperation{}

type LongRunningOperation struct {
	Name                string
	Operation           *operation.OperationRequest
	LroEntity           *LongRunningEntity
	OperationId         string
	ExpirationTimestamp *timestamppb.Timestamp
}

var CreateLroEntityFunc entity.EntityFactoryFunc = func(id string) (entity.Entity, error) {
	return NewLongRunningEntity(id), nil
}

func (lro *LongRunningOperation) InitOperation(ctx context.Context, opRequest *operation.OperationRequest) (operation.ApiOperation, *asyncErrors.AsyncError) {
	logger := ctxlogger.GetLogger(ctx)

	logger.Info("Initializing LongRunningOperation")
	lro.Operation = opRequest
	lro.Name = opRequest.OperationName
	lro.OperationId = opRequest.OperationId

	return nil, nil
}

func (lro *LongRunningOperation) Run(ctx context.Context) *asyncErrors.AsyncError {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Running the long running operation!")

	// Logic for running the operation
	time.Sleep(20 * time.Second)
	logger.Info("Finished running the long running operation.")

	return nil
}

func (lro *LongRunningOperation) GuardConcurrency(ctx context.Context, entity entity.Entity) *asyncErrors.AsyncError {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Guarding concurrency for operation.")

	if latestOperationId := entity.GetLatestOperationID(); lro.OperationId != latestOperationId {
		err := errors.New("OperaionId and LastOperationId don't match!")
		aErr := &asyncErrors.AsyncError{
			OriginalError: err,
			ErrorCode:     int(codes.Canceled),
			Message:       "GuardConcurrency error.",
		}
		return aErr
	}

	return nil
}

func (lro *LongRunningOperation) GetOperationRequest() *operation.OperationRequest {
	return lro.Operation
}
