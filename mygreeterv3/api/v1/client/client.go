package client

import (
	pb "dev.azure.com/service-hub-flg/service_hub_validation/_git/service_hub_validation_service.git/mygreeterv3/api/v1"
	"github.com/Azure/aks-middleware/grpc/interceptor"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	log "log/slog"
)

func NewClient(remoteAddr string, options interceptor.ClientInterceptorLogOptions) (pb.MyGreeterClient, *grpc.ClientConn, error) {
	conn, err := grpc.NewClient(
		remoteAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(
			interceptor.DefaultClientInterceptors(options)...,
		),
	)
	if err != nil {
		log.Error("did not connect: " + err.Error())
		return nil, nil, err
	}

	client := pb.NewMyGreeterClient(conn)

	// Return both the client and the connection so that the caller can close the connection when done
	return client, conn, nil
}
