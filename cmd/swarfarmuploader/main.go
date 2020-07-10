package main

import (
	"context"
	"net"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/thecodeteam/goodbye"
	"google.golang.org/grpc"

	"github.com/swarpf/plugins/internal/proxyapiutil"
	"github.com/swarpf/plugins/pkg/swarfarm"
	pb "github.com/swarpf/plugins/swarpf-idl/proto-gen-go/proxyapi"
)

func main() {
	// load configuration from command line or environment
	pflag.String("proxyapi_addr", "127.0.0.1:11100", "Address of the proxy host")
	pflag.String("listen_addr", "0.0.0.0:11103", "Listen address for the plugin")
	pflag.Bool("development", false, "Enable development logging")
	pflag.StringToString("api_tokens", map[string]string{}, "List of API tokens for SWARFARM. Format: 'wizardId=Token,...'")
	pflag.Parse()

	viper.SetEnvPrefix("plugin_swarfarmuploader")
	viper.AutomaticEnv()
	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		// TODO(lyrex): figure out what to do here.
		return
	}

	proxyAddress := viper.GetString("proxyapi_addr")
	listenAddress := viper.GetString("listen_addr")

	// setup logging
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if viper.GetBool("development") {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	}
	log.Logger = log.With().Timestamp().Str("log_type", "plugin").Str("plugin", "SWARFARM uploader").Logger()

	// Process all swarfarm API tokens
	for wizardId, token := range viper.GetStringMapString("api_tokens") {
		log.Info().Str("wizardId", wizardId).Msgf("Adding token for wizard %s", wizardId)

		swarfarm.AddProfile(wizardId, token)
	}

	// Main Program
	log.Info().
		Str("proxyAddr", proxyAddress).
		Msgf("Connecting SWARFARM uploader plugin to proxy %s", proxyAddress)

	// Create a context to use with the Goodbye library's functions.
	ctx := context.Background()
	defer goodbye.Exit(ctx, -1)
	goodbye.Notify(ctx)

	subscribedCommands := swarfarm.SubscribedCommands()
	goodbye.RegisterWithPriority(func(ctx context.Context, sig os.Signal) {
		proxyapiutil.DisconnectFromProxyApi(proxyAddress, listenAddress, subscribedCommands)

		log.Info().Err(err).Msg("SWARFARM uploader plugin ended")
	}, -1)

	// initialize proxy consumer
	lis, err := net.Listen("tcp", listenAddress)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create listener")
	}

	log.Info().
		Str("listenAddr", listenAddress).
		Msgf("Listening for new proxy api connections on %s", listenAddress)

	s := grpc.NewServer()
	pb.RegisterProxyApiConsumerServer(s, &swarfarm.ProxyApiConsumer{})

	go proxyapiutil.RegisterWithProxyApi(proxyAddress, listenAddress, subscribedCommands)

	if err := s.Serve(lis); err != nil {
		log.Info().Str("reason", err.Error()).Msg("Server stopped listening")
	}
}
