package swaglogger

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
)

func SubscribedCommands() []string {
	return []string{"GetGuildWarBattleLogByWizardId", "GetGuildWarBattleLogByGuildId"}
}

func isSubscribedCommand(command string) bool {
	for _, b := range SubscribedCommands() {
		if b == command {
			return true
		}
	}
	return false
}

func OnReceiveApiEvent(command, request, response string) error {
	if !isSubscribedCommand(command) {
		return nil
	}

	requestContent := map[string]interface{}{}
	if err := json.Unmarshal([]byte(request), &requestContent); err != nil {
		log.Error().Err(err).Msg("Failed to deserializie SWAG request")
		return errors.New("error while deserializing SWAG request")
	}

	responseContent := map[string]interface{}{}
	if err := json.Unmarshal([]byte(response), &responseContent); err != nil {
		log.Error().Err(err).Msg("Failed to deserializie SWAG response")
		return errors.New("error while deserializing SWAG response")
	}

	wizardId := requestContent["wizard_id"].(float64)

	log.Info().
		Str("command", command).
		Float64("wizard_id", wizardId).
		Msg("Uploading guild war data to SWAG...")

	client := resty.New()
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(response).
		Post("https://gw.swop.one/data/upload/")

	if err != nil {
		log.Error().Err(err).
			Str("command", command).
			Float64("wizardId", wizardId).
			Msg("SWAG upload failed")
		return nil
	}

	if resp.StatusCode() != http.StatusOK {
		log.Error().
			Str("command", command).
			Float64("wizardId", wizardId).
			Int("StatusCode", resp.StatusCode()).
			Msgf("SWAG upload failed. Status %d", resp.StatusCode())
		return nil
	}

	log.Info().
		Str("command", command).
		Float64("wizardId", wizardId).
		Msg("SWAG upload successful.")

	return nil
}
