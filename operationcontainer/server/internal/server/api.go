package server

import (
	"context"
	"fmt"
	log "log/slog"
	"net/http"
	"os"
	"regexp"

	"database/sql"

	database "github.com/Azure/aks-async/database"

	"dev.azure.com/service-hub-flg/service_hub_validation/_git/service_hub_validation_service.git/operationcontainer/server/internal/logattrs"
	pb "github.com/Azure/OperationContainer/api/v1"
	"google.golang.org/grpc"
)

type ServerInfo struct {
	GrpcServer *grpc.Server
	GwServer   *http.Server
	GrpcPort   int
	HttpPort   int
}

type Server struct {
	// When the UnimplementedOperationContainerServer struct is embedded,
	// the generated method/implementation in .pb file will be associated with this struct.
	// If this struct doesn't implment some methods,
	// the .pb ones will be used. If this struct implement the methods, it will override the .pb ones.
	// The reason is that anonymous field's methods are promoted to the struct.
	//
	// When this struct is NOT embedded,, all methods have to be implemented to meet the interface requirement.
	// See https://go.dev/ref/spec#Struct_types.
	pb.UnimplementedOperationContainerServer
	client             pb.OperationContainerClient
	dbClient           *sql.DB
	operationTableName string
	ServerInfo
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Init(options Options) {
	var err error

	logger := log.New(log.NewTextHandler(os.Stdout, nil).WithAttrs(logattrs.GetAttrs()))
	if options.JsonLog {
		logger = log.New(log.NewJSONHandler(os.Stdout, nil).WithAttrs(logattrs.GetAttrs()))
	}

	log.SetDefault(logger)

	dbStarted := false
	if options.DatabaseConnectionString != "" {
		s.dbClient, err = database.NewDbClientWithConnectionString(context.Background(), options.DatabaseConnectionString)
		if err != nil {
			logger.Error("Error creating connection pool: " + err.Error())
			os.Exit(1)
		}
		dbStarted = true
	} else if options.DatabaseServerUrl != "" && options.DatabaseName != "" {
		s.dbClient, err = database.NewDbClient(context.Background(), options.DatabaseServerUrl, options.DatabasePort, options.DatabaseName)
		if err != nil {
			logger.Error("Error creating connection pool: " + err.Error())
			os.Exit(1)
		}
		dbStarted = true
	}

	if dbStarted {

		if options.OperationTableName == "" {
			logger.Error("No OperationTableName set.")
			os.Exit(1)
		}

		if err := sanitizeTableName(options.OperationTableName); err != nil {
			logger.Error("Table name is not valid: " + err.Error())
			os.Exit(1)
		}

		s.operationTableName = options.OperationTableName
		// Check if table exists
		operationListCheckQuery := "SELECT * FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_NAME = @p1"
		rows, err := database.QueryDb(context.Background(), s.dbClient, operationListCheckQuery, options.OperationTableName)
		if err != nil {
			logger.Error("Error querying the database for table information: " + err.Error())
			os.Exit(1)
		}

		var tableCatalog, tableSchema, tableName, tableType string
		for rows.Next() {
			err = rows.Scan(&tableCatalog, &tableSchema, &tableName, &tableType)
			if err != nil {
				logger.Error("Error scanning database rows: " + err.Error())
				os.Exit(1)
			}
		}

		if tableCatalog == "" && tableSchema == "" && tableName == "" && tableType == "" {
			logger.Info(fmt.Sprintf("The table %s doesn't exist!", options.OperationTableName))
			operationListCreateTableQuery := fmt.Sprintf("CREATE TABLE %s (operation_id VARCHAR(255), operation_name VARCHAR(255), operation_status VARCHAR(255), entity_id VARCHAR(255), expiration_date TIMESTAMP)", options.OperationTableName)
			_, err = database.QueryDb(context.Background(), s.dbClient, operationListCreateTableQuery)
			if err != nil {
				logger.Error("Error creating database table: " + err.Error())
				os.Exit(1)
			}
			logger.Info(fmt.Sprintf("The table %s has been created.", options.OperationTableName))
		}
	}
}

func sanitizeTableName(tableName string) error {
	// Use a regular expression to allow only alphanumeric characters and underscores
	validName := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	if !validName.MatchString(tableName) {
		return fmt.Errorf("invalid table name: %s", tableName)
	}
	return nil
}
