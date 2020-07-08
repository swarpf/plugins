package swarfarm

import (
	"encoding/json"
	"errors"

	"github.com/rs/zerolog/log"
)

func SubscribedCommands() []string {
	commands := GetProfileUploadCommands()
	for k := range FetchAcceptedLoggerCommands() {
		commands = append(commands, k)
	}
	return commands
}

func OnReceiveApiEvent(command, request, response string) error {
	if !isProfileUploadCommand(command) || !isCommandLoggerCommand(command) {
		return nil
	}

	requestContent := map[string]interface{}{}
	if err := json.Unmarshal([]byte(request), &requestContent); err != nil {
		log.Error().Err(err).Msg("Failed to deserializie SWARFARM request")
		return errors.New("error while deserializing SWARFARM request")
	}

	responseContent := map[string]interface{}{}
	if err := json.Unmarshal([]byte(response), &responseContent); err != nil {
		log.Error().Err(err).Msg("Failed to deserializie SWARFARM response")
		return errors.New("error while deserializing SWARFARM response")
	}

	wizardInfo := requestContent["wizard_info"].(map[string]interface{})
	wizardId := int64(wizardInfo["wizard_id"].(float64))

	if isProfileUploadCommand(command) {
		return UploadSwarfarmProfile(wizardId, command, response)
	} else if isCommandLoggerCommand(command) {
		return UploadSwarfarmCommand(wizardId, command, requestContent, responseContent)
	}

	return errors.New("unknown command")
}

func isCommandLoggerCommand(command string) bool {
	for k := range FetchAcceptedLoggerCommands() {
		if k == command {
			return true
		}
	}
	return false
}

func isProfileUploadCommand(command string) bool {
	for _, b := range GetProfileUploadCommands() {
		if b == command {
			return true
		}
	}
	return false
}
