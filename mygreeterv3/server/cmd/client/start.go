package main

import (
	"context"
	"io"
	"math/rand"
	"os"
	"time"

	"dev.azure.com/service-hub-flg/service_hub_validation/_git/service_hub_validation_service.git/mygreeterv3/api/v1/client"
	"dev.azure.com/service-hub-flg/service_hub_validation/_git/service_hub_validation_service.git/mygreeterv3/server/internal/logattrs"

	"strconv"

	pb "dev.azure.com/service-hub-flg/service_hub_validation/_git/service_hub_validation_service.git/mygreeterv3/api/v1"
	"dev.azure.com/service-hub-flg/service_hub_validation/_git/service_hub_validation_service.git/mygreeterv3/api/v1/restsdk"
	"github.com/Azure/aks-middleware/grpc/interceptor"
	"github.com/Azure/aks-middleware/http/client/direct/restlogger"
	"google.golang.org/protobuf/types/known/timestamppb"

	"strings"

	log "log/slog"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "hello",
	Short: "Call SayHello",
	Run:   hello,
}

var output io.Writer = os.Stdout

type Options struct {
	RemoteAddr       string
	HttpAddr         string
	JsonLog          bool
	Name             string
	Age              int32
	Email            string
	Address          string
	IntervalMilliSec int64
	RgName           string
	RgRegion         string
	CallAllRGOps     bool
	SubscriptionId   string
}

var options = newOptions()

func init() {
	rootCmd.AddCommand(startCmd)

	startCmd.Flags().StringVar(&options.RemoteAddr, "remote-addr", options.RemoteAddr, "the remote server's addr for this client to connect to")
	startCmd.Flags().StringVar(&options.HttpAddr, "http-addr", options.HttpAddr, "the remote http gateway addr")
	startCmd.Flags().BoolVar(&options.JsonLog, "json-log", options.JsonLog, "The format of the log is json or user friendly key-value pairs")
	startCmd.Flags().StringVar(&options.Name, "name", options.Name, "The name to send in Hello request")
	startCmd.Flags().Int32Var(&options.Age, "age", options.Age, "The age to send in Hello request")
	startCmd.Flags().StringVar(&options.Email, "email", options.Email, "The email to send in Hello request")
	startCmd.Flags().StringVar(&options.Address, "address", options.Address, "The address to send in Hello request")
	startCmd.Flags().Int64Var(&options.IntervalMilliSec, "interval-milli-sec", options.IntervalMilliSec,
		"The interval between two requests. Negative numbers mean sending one request.")
	startCmd.Flags().StringVar(&options.RgName, "rg-name", options.RgName, "The name of the resource group")
	startCmd.Flags().StringVar(&options.RgRegion, "rg-region", options.RgRegion, "The region of the resource group")
	startCmd.Flags().BoolVar(&options.CallAllRGOps, "call-all-rg-ops", options.CallAllRGOps, "Call all resource group operations")
	startCmd.Flags().StringVar(&options.SubscriptionId, "subscription-id", options.SubscriptionId, "The Azure subscription ID to use for API calls")

	// Seed at the beginning of the program once, not on every run: https://stackoverflow.com/questions/12321133/how-to-properly-seed-random-number-generator
	rand.Seed(time.Now().UnixNano())
}

func newOptions() Options {
	return Options{
		RemoteAddr: "localhost:50051",
		// Don't use proxy locally due to audit log validation issues, send REST request directly to grpc gateway port
		HttpAddr:         "http://localhost:50061",
		JsonLog:          false,
		Name:             "MyName",
		Age:              53,
		Email:            "test@test.com",
		Address:          "123 Main St, Seattle, WA 98101",
		IntervalMilliSec: -1,
		RgName:           "MyGreeter-resource-group",
		RgRegion:         "eastus",
		CallAllRGOps:     true,
		SubscriptionId:   "8ecadfc9-d1a3-4ea4-b844-0d9f87e4d7c8",
	}
}

func SetOutput(out io.Writer) {
	output = out
}

func hello(cmd *cobra.Command, args []string) {
	logger := log.New(log.NewTextHandler(output, nil).WithAttrs(logattrs.GetAttrs()))
	if options.JsonLog {
		logger = log.New(log.NewJSONHandler(output, nil).WithAttrs(logattrs.GetAttrs()))
	}

	log.SetDefault(logger)

	client, conn, err := client.NewClient(options.RemoteAddr, interceptor.GetClientInterceptorLogOptions(logger, logattrs.GetAttrs()))
	// logging the error for transparency, retry interceptor will handle it
	if err != nil {
		log.Error("did not connect: " + err.Error())
	}

	if options.IntervalMilliSec < 0 {
		SayHello(client, options.Name, options.Age, options.Email, options.Address, logger)
		conn.Close()
		return
	}
	for {
		SayHello(client, options.Name, options.Age, options.Email, options.Address, logger)
		time.Sleep(time.Duration(options.IntervalMilliSec) * time.Millisecond)

		defer conn.Close()
	}
}

func SayHello(client pb.MyGreeterClient, name string, age int32, email string, address string, logger *log.Logger) {
	ctx := context.Background()

	addressParts := strings.Split(address, ",")
	street := addressParts[0]
	city := addressParts[1]
	stateAndZip := strings.Split(addressParts[2], " ")
	state := stateAndZip[1]
	parseZipcode, _ := strconv.ParseInt(stateAndZip[2], 10, 32)
	zipcode := int32(parseZipcode)

	addr := &pb.Address{
		Street:  street,
		City:    city,
		State:   state,
		Zipcode: zipcode,
	}
	r, err := client.SayHello(ctx, &pb.HelloRequest{Name: name, Age: age, Email: email, Address: addr})
	if err != nil {
		log.Info("SayHello error: " + err.Error())
	} else {
		log.Info("Response message: " + r.GetMessage())
	}

	tags := map[string]string{
		"tag1": "value1",
		"tag2": "value2",
	}

	// Create a new Configuration instance
	cfg := &restsdk.Configuration{
		BasePath: options.HttpAddr,
		DefaultHeader: map[string]string{
			"Content-Type":       "application/json",
			"x-ms-client-app-id": "MyGreeter-Client",
		},
		UserAgent:  "Swagger-Codegen/1.0.0/go",
		HTTPClient: restlogger.NewLoggingClient(logger),
	}

	apiClient := restsdk.NewAPIClient(cfg)

	service := apiClient.MyGreeterApi

	// Include subID in REST request URL to conform with ARM URL formating (/subscriptions/...) for audit logging
	subId := options.SubscriptionId

	// Only update resource groups and perform storage account CRUDL ops
	// in int and stg environments due to current MSI role
	if !options.CallAllRGOps {
		_, err = client.UpdateResourceGroup(ctx, &pb.UpdateResourceGroupRequest{SubscriptionId: subId, Name: options.RgName, Tags: tags})
		if err != nil {
			log.Error("Error calling UpdateResourceGroup: " + err.Error())
		}
		_, _, err = service.MyGreeterUpdateResourceGroup(context.Background(), tags, subId, options.RgName)
		if err != nil {
			log.Error("Error calling MyGreeterUpdateResourceGroup: " + err.Error())
		}
		resp, createSAErr := client.CreateStorageAccount(ctx, &pb.CreateStorageAccountRequest{SubscriptionId: subId, RgName: options.RgName, SaName: "testsa" + strconv.Itoa(rand.Intn(1000)), Region: options.RgRegion})
		if createSAErr != nil {
			log.Error("Error calling CreateStorageAccount: " + err.Error())
		}
		storageAccountName := resp.Name
		_, err = client.ReadStorageAccount(ctx, &pb.ReadStorageAccountRequest{SubscriptionId: subId, RgName: options.RgName, SaName: storageAccountName})
		if err != nil {
			log.Error("Error calling ReadStorageAccount: " + err.Error())
		}
		_, err = client.UpdateStorageAccount(ctx, &pb.UpdateStorageAccountRequest{SubscriptionId: subId, RgName: options.RgName, SaName: storageAccountName, Tags: tags})
		if err != nil {
			log.Error("Error calling UpdateStorageAccount: " + err.Error())
		}
		_, err = client.ListStorageAccounts(ctx, &pb.ListStorageAccountRequest{SubscriptionId: subId, RgName: options.RgName})
		if err != nil {
			log.Error("Error calling ListStorageAccounts: " + err.Error())
		}
		_, err = client.DeleteStorageAccount(ctx, &pb.DeleteStorageAccountRequest{SubscriptionId: subId, RgName: options.RgName, SaName: storageAccountName})
		if err != nil {
			log.Error("Error calling DeleteStorageAccount: " + err.Error())
		}

	} else {
		_, err = client.CreateResourceGroup(ctx, &pb.CreateResourceGroupRequest{SubscriptionId: subId, Name: options.RgName, Region: options.RgRegion})
		if err != nil {
			log.Error("Error calling CreateResourceGroup: " + err.Error())
		}
		_, err = client.ReadResourceGroup(ctx, &pb.ReadResourceGroupRequest{SubscriptionId: subId, Name: options.RgName})
		if err != nil {
			log.Error("Error calling ReadResourceGroup: " + err.Error())
		}
		_, err = client.UpdateResourceGroup(ctx, &pb.UpdateResourceGroupRequest{SubscriptionId: subId, Name: options.RgName, Tags: tags})
		if err != nil {
			log.Error("Error calling UpdateResourceGroup: " + err.Error())
		}
		_, err = client.ListResourceGroups(ctx, &pb.ListResourceGroupsRequest{SubscriptionId: subId})
		if err != nil {
			log.Error("Error calling ListResourceGroup: " + err.Error())
		}
		_, err = client.DeleteResourceGroup(ctx, &pb.DeleteResourceGroupRequest{SubscriptionId: subId, Name: options.RgName})
		if err != nil {
			log.Error("Error calling DeleteResourceGroup: " + err.Error())
		}

		createRequestBody := restsdk.MyGreeterCreateResourceGroupBody{
			Region: options.RgRegion,
		}
		_, _, err = service.MyGreeterCreateResourceGroup(context.Background(), createRequestBody, subId, options.RgName)
		if err != nil {
			log.Error("Error calling MyGreeterCreateResourceGroup: " + err.Error())
		}
		_, _, err = service.MyGreeterReadResourceGroup(context.Background(), subId, options.RgName)
		if err != nil {
			log.Error("Error calling MyGreeterReadResourceGroup: " + err.Error())
		}
		_, _, err = service.MyGreeterUpdateResourceGroup(context.Background(), tags, subId, options.RgName)
		if err != nil {
			log.Error("Error calling MyGreeterUpdateResourceGroup: " + err.Error())
		}
		_, _, err = service.MyGreeterListResourceGroups(context.Background(), subId)
		if err != nil {
			log.Error("Error calling MyGreeterListResourceGroups: " + err.Error())
		}
		_, _, err = service.MyGreeterDeleteResourceGroup(context.Background(), subId, options.RgName)
		if err != nil {
			log.Error("Error calling MyGreeterDeleteResourceGroup: " + err.Error())
		}
	}

	protoExpirationTime := timestamppb.New(time.Now().Add(1 * time.Hour))
	randomNumber := rand.Intn(100) + 1
	randomNumberStr := strconv.Itoa(randomNumber)
	startOperationRequest := &pb.StartLongRunningOperationRequest{
		EntityId:            randomNumberStr,
		EntityType:          "Cluster",
		ExpirationTimestamp: protoExpirationTime,
	}
	startOperationResponse, err := client.StartLongRunningOperation(ctx, startOperationRequest)
	if err != nil {
		log.Error("Error calling StartOperation: " + err.Error())
	} else {
		log.Info("Started operation with operationId: " + startOperationResponse.GetOperationId())
	}
}
