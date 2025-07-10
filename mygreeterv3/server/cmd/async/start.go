// Auto generated. Can be modified.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	log "log/slog"

	async "dev.azure.com/service-hub-flg/service_hub_validation/_git/service_hub_validation_service.git/mygreeterv3/server/internal/async"
	"dev.azure.com/service-hub-flg/service_hub_validation/_git/service_hub_validation_service.git/mygreeterv3/server/internal/logattrs"
	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the service",
	Run:   start,
}

var options = async.Options{}

func init() {
	rootCmd.AddCommand(startCmd)

	startCmd.Flags().IntVar(&options.Port, "port", 50052, "the addr to serve the api on")
	startCmd.Flags().BoolVar(&options.JsonLog, "json-log", false, "The format of the log is json or user friendly key-value pairs")
	startCmd.Flags().StringVar(&options.SubscriptionID, "subscription-id", "", "The subscription ID to connect to")
	startCmd.Flags().StringVar(&options.IdentityResourceID, "identity-resource-id", "", "the MSI used to authenticate to Azure from E2E env")
	startCmd.Flags().StringVar(&options.OperationContainerAddr, "opcon-addr", "localhost:50041", "the remote server's addr for this client to connect to")
	startCmd.Flags().StringVar(&options.ServiceBusHostName, "service-bus-hostname", "servicehubval-resourceName-location-sb-ns.servicebus.windows.net", "The host name used to connect to the service bus.")
	startCmd.Flags().StringVar(&options.ServiceBusQueueName, "service-bus-queue-name", "servicehubval-resourceName-queue", "The name of the queue to which we will send messages.")
	startCmd.Flags().StringVar(&options.DatabaseConnectionString, "database-connection-string", "", "Connection string used to connect to the database")
	startCmd.Flags().StringVar(&options.DatabaseServerUrl, "database-server-url", "", "The server of the database to connect to.")
	startCmd.Flags().StringVar(&options.DatabaseName, "database-name", "", "The name of the database to connect to.")
	startCmd.Flags().IntVar(&options.DatabasePort, "database-port", 1433, "The port to connect to the database")
	startCmd.Flags().StringVar(&options.EntityTableName, "entity-table-name", "hcp", "The name of the table that holds entity metadata and last operation affecting that entity.")
}

var output io.Writer = os.Stdout

func start(cmd *cobra.Command, args []string) {
	logger := log.New(log.NewTextHandler(output, nil).WithAttrs(logattrs.GetAttrs()))
	if options.JsonLog {
		logger = log.New(log.NewJSONHandler(output, nil).WithAttrs(logattrs.GetAttrs()))
	}

	log.SetDefault(logger)
	ctx, cancel := context.WithCancel(context.Background())
	ctx = ctxlogger.WithLogger(ctx, logger)
	defer cancel()

	//TODO(mheberling): Investigate why dlq needs to be first for logs to show up.
	asyncDlq, err := async.NewAsync(ctx, options, true)
	if err != nil {
		logger.Error("New asyncDlq: " + err.Error())
		os.Exit(1)
	}

	asyncReg, err := async.NewAsync(ctx, options, false)
	if err != nil {
		logger.Error("New async: " + err.Error())
		os.Exit(1)
	}

	// We make a channel with 2 errors to allow both processors to fail and see their errors.
	// Otherwise we can have the second failure to be blocked forever because the channel
	// can't read the second error.
	errChan := make(chan error, 2)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		err = asyncDlq.Processor.Start(ctx)
		if err != nil {
			err = fmt.Errorf("Starting dlq processor: %w", err)
			logger.Error(err.Error())
			errChan <- err
		}
	}()

	go func() {
		defer wg.Done()
		err = asyncReg.Processor.Start(ctx)
		if err != nil {
			err = fmt.Errorf("Starting regular processor: %w", err)
			logger.Error(err.Error())
			errChan <- err
		}
	}()

	go func() {
		select {
		case firstErr := <-errChan: // Wait for the first error
			logger.Error("Error encountered: " + firstErr.Error())
			cancel() // Cancel both processors immediately
		}

		// Try to collect a second error (if any), but don't block forever
		select {
		case secondErr := <-errChan:
			logger.Error("Second error encountered: " + secondErr.Error())
		default:
			// No second error, continue
		}
	}()

	wg.Wait()
	close(errChan) // Close the channel after all goroutines finish

	logger.Info("Both processors have stopped. Exiting.")
	os.Exit(1)
}
