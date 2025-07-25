package server

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	pb "dev.azure.com/service-hub-flg/service_hub_validation/_git/service_hub_validation_service.git/mygreeterv3/api/v1"
	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service Account Name needs to be globally unique
func generateID() string {
	rand.Seed(time.Now().UnixNano())
	uid := rand.Intn(90000) + 10000

	return fmt.Sprintf("%05d", uid)
}

func (s *Server) generateUniqueStorageAccountName(ctx context.Context) (string, error) {
	maxIterations := 10
	delayMilliseconds := 100
	name := ""
	for i := 0; i < maxIterations; i++ {
		name = "sa" + generateID()
		res, checkErr := s.AccountsClient.CheckNameAvailability(ctx, armstorage.AccountCheckNameAvailabilityParameters{
			Name: to.Ptr(name),
			Type: to.Ptr("Microsoft.Storage/storageAccounts"),
		}, nil)
		if checkErr != nil {
			return "", checkErr
		}
		if *res.NameAvailable {
			return name, nil
		}

		// Add a delay to avoid hitting the server too fast
		time.Sleep(time.Duration(delayMilliseconds) * time.Millisecond)
	}
	return "", status.Errorf(codes.AlreadyExists, "Storage account name (%s) already exists", name)
}

func (s *Server) CreateStorageAccount(ctx context.Context, in *pb.CreateStorageAccountRequest) (*pb.CreateStorageAccountResponse, error) {
	logger := ctxlogger.GetLogger(ctx)
	if s.AccountsClient == nil {
		logger.Error("AccountsClient is nil in CreateStorageAccount(), azuresdk feature is likely disabled")
		return nil, status.Errorf(codes.Unimplemented, "AccountsClient is nil in CreateStorageAccount(), azuresdk feature is likely disabled")
	}

	// Use the storage account name from the request, or generate one if not provided
	var name string
	var checkErr error

	if in.GetSaName() != "" {
		// Use the provided name and check its availability
		name = in.GetSaName()
		res, err := s.AccountsClient.CheckNameAvailability(ctx, armstorage.AccountCheckNameAvailabilityParameters{
			Name: to.Ptr(name),
			Type: to.Ptr("Microsoft.Storage/storageAccounts"),
		}, nil)
		if err != nil {
			logger.Error("CheckNameAvailability() error for provided name: " + err.Error())
			return nil, HandleError(err, "CheckNameAvailability()")
		}
		if !*res.NameAvailable {
			logger.Error("Provided storage account name is not available: " + name)
			return nil, status.Errorf(codes.AlreadyExists, "Storage account name (%s) is not available: %s", name, *res.Message)
		}
	} else {
		// Generate a unique name if none was provided
		name, checkErr = s.generateUniqueStorageAccountName(ctx)
		if checkErr != nil {
			logger.Error("CheckNameAvailability() error: " + checkErr.Error())
			return nil, HandleError(checkErr, "CheckNameAvailability()")
		}
	}

	params := armstorage.AccountCreateParameters{
		Location: to.Ptr(in.GetRegion()),
		SKU: &armstorage.SKU{
			Name: to.Ptr(armstorage.SKUNameStandardGRS),
		},
		Kind: to.Ptr(armstorage.KindStorageV2),
		Properties: &armstorage.AccountPropertiesCreateParameters{
			AllowBlobPublicAccess: to.Ptr(false),
		},
	}
	poller, err := s.AccountsClient.BeginCreate(ctx, in.GetRgName(), name, params, nil)
	if err != nil {
		logger.Error("BeginCreate() error: " + err.Error())
		return nil, HandleError(err, "BeginCreate()")
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		logger.Error("PollUntilDone() error: " + err.Error())
		return nil, HandleError(err, "PollUntilDone()")
	}

	logger.Info("Created storage account: " + name)
	return &pb.CreateStorageAccountResponse{Name: name}, nil
}
