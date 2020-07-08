package debugout

import (
	"github.com/rs/zerolog/log"
)

func SubscribedCommands() []string {
	return []string{"*"}
}

func OnReceiveApiEvent(command, request, response string) error {
	log.Debug().Timestamp().
		Str("command", command).
		Str("request", request).
		Str("response", response).
		Msg("Debug Output Plugin")

	return nil
}
