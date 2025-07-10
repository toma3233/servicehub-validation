package client

import (
	"context"
	"time"

	log "log/slog"

	"github.com/Azure/aks-middleware/grpc/interceptor"
	"google.golang.org/grpc/connectivity"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {
	attrs := []log.Attr{}

	It("should create a new client and conn should not be ready", func() {
		client, conn, err := NewClient("localhost:50051", interceptor.GetClientInterceptorLogOptions(log.Default(), attrs))
		Expect(err).To(BeNil())
		Expect(client).To(Not(BeNil()))
		Expect(conn).To(Not(BeNil()))

		defer conn.Close()
		Expect(conn.GetState()).To(Not(Equal(connectivity.Shutdown)))

		// Use context to wait for state change, ensuring it doesn't transition to READY
		// since we are not connecting to an actual server
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		initialState := conn.GetState()
		conn.WaitForStateChange(ctx, initialState)
		Expect(conn.GetState()).To(Not(Equal(connectivity.Ready)))
	})
})
