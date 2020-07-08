package main

import (
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/swarpf/plugins/internal/proxyapiutil"
	"github.com/swarpf/plugins/pkg/profileexport"
	pb "github.com/swarpf/plugins/swarpf-idl/proto-gen-go/proxyapi"
)

func main() {
	// load configuration from command line or environment
	pflag.String("proxyapi_addr", "127.0.0.1:8010", "Address of the proxy host")
	pflag.String("listen_addr", "127.0.0.1:11102", "Listen address for the plugin")
	pflag.String("output_directory", "./export", "Output directory for the profile files")
	pflag.Bool("development", false, "Enable development logging")
	pflag.Parse()

	viper.SetEnvPrefix("plugin_profileexport")
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
	log.Logger = log.With().Timestamp().Str("log_type", "plugin").Str("plugin", "Profile Exporter").Logger()

	// Main Program
	log.Info().
		Str("proxyAddr", proxyAddress).
		Msgf("Connecting Profile Exporter plugin to proxy %s", proxyAddress)

	// Setting profile export directory. Important: This can fail and abort the plugin!
	profileexport.SetOutputDirectory(viper.GetString("output_directory"))

	// initialize proxy consumer
	lis, err := net.Listen("tcp", listenAddress)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create listener")
	}

	log.Info().
		Str("listenAddr", listenAddress).
		Msgf("Listening for new proxy api connections on %s", listenAddress)

	s := grpc.NewServer()
	pb.RegisterProxyApiConsumerServer(s, &profileexport.ProxyApiConsumer{})

	go proxyapiutil.RegisterWithProxyApi(proxyAddress, listenAddress, profileexport.SubscribedCommands())

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Info().Str("reason", err.Error()).Msg("Server stopped listening")
		}
	}()

	// Setting up signal capturing
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	// Waiting for SIGINT (pkill -2)
	<-stop

	proxyapiutil.DisconnectFromProxyApi(proxyAddress, listenAddress, profileexport.SubscribedCommands())

	log.Info().Err(err).Msg("Profile Exporter plugin ended")
}
