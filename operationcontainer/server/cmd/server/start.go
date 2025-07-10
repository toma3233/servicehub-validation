// Auto generated. Can be modified.
package main

import (
	"os"
	"os/signal"
	"syscall"

	"dev.azure.com/service-hub-flg/service_hub_validation/_git/service_hub_validation_service.git/operationcontainer/server/internal/server"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the service",
	Run:   start,
}

var options = server.Options{}

func init() {
	rootCmd.AddCommand(startCmd)

	startCmd.Flags().IntVar(&options.Port, "port", 50251, "the addr to serve the api on")
	startCmd.Flags().BoolVar(&options.JsonLog, "json-log", false, "The format of the log is json or user friendly key-value pairs")
	startCmd.Flags().IntVar(&options.HTTPPort, "http port", 50261, "the addr to serve the gRPC-Gateway on")
	startCmd.Flags().StringVar(&options.DatabaseConnectionString, "database-connection-string", "", "Connection string used to connect to the database")
	startCmd.Flags().StringVar(&options.DatabaseServerUrl, "database-server-url", "", "The server of the database to connect to.")
	startCmd.Flags().StringVar(&options.DatabaseName, "database-name", "", "The name of the database to connect to.")
	startCmd.Flags().IntVar(&options.DatabasePort, "database-port", 1433, "The port to connect to the database")
	startCmd.Flags().StringVar(&options.OperationTableName, "operation-table-name", "operations", "The name of the table that holds all the operations.")
}

func start(cmd *cobra.Command, args []string) {
	newServer := server.NewServer()
	newServer.Init(options)
	newServer.Serve(options)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Wait indefinitely until a signal is received
	<-stop

	newServer.Cleanup()
}
