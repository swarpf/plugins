package proxyapiutil

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"

	pb "github.com/swarpf/plugins/swarpf-idl/proto-gen-go/proxyapi"
)

func ConnectToProxyApi(proxyAddress string, delay bool) (*grpc.ClientConn, pb.ProxyApiClient,
	context.Context, context.CancelFunc) {
	if delay {
		const nSeconds = 5

		delayTimer := time.NewTimer(nSeconds * time.Second)
		log.Debug().Msgf("Delaying host registration by %d seconds", nSeconds)
		<-delayTimer.C
	}

	log.Debug().
		Str("proxyAddress", proxyAddress).
		Msgf("Trying to connect to proxy api")

	dialCtx, dialCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer dialCancel()
	conn, err := grpc.DialContext(dialCtx, proxyAddress, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatal().Err(err).
			Str("proxyAddress", proxyAddress).
			Msg("Failed to connect to proxy api")
	}

	c := pb.NewProxyApiClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

	return conn, c, ctx, cancel
}

func RegisterWithProxyApi(proxyAddress, listenAddress string, subscribedCommands []string) {
	conn, c, ctx, cancel := ConnectToProxyApi(proxyAddress, true)

	defer cancel()
	defer tryCloseConnection(conn)

	if _, err := c.Register(ctx, &pb.ProxyApiOptions{
		Address:  listenAddress,
		Commands: subscribedCommands,
	}); err != nil {
		log.Fatal().Err(err).
			Str("proxyAddress", proxyAddress).
			Str("listenAddress", listenAddress).
			Msg("Failed to register myself at the proxy host")
	}

	log.Info().Msg("Successfully registered at proxy api")
}

func DisconnectFromProxyApi(proxyAddress, listenAddress string, subscribedCommands []string) {
	conn, c, ctx, cancel := ConnectToProxyApi(proxyAddress, false)

	defer cancel()
	defer tryCloseConnection(conn)

	if _, err := c.Disconnect(ctx, &pb.ProxyApiOptions{
		Address:  listenAddress,
		Commands: subscribedCommands,
	}); err != nil {
		log.Fatal().Err(err).
			Str("proxyAddress", proxyAddress).
			Str("listenAddress", listenAddress).
			Msg("Failed to disconnect myself from the proxy api")
	}

	log.Info().Msg("Successfully disconnected from proxy api")
}

func tryCloseConnection(conn *grpc.ClientConn) {
	if err := conn.Close(); err != nil {
		log.Error().Err(err).Msg("Failed to close connection")
	}
}
