package swarfarm

import (
	"encoding/json"
	"errors"

	"github.com/rs/zerolog/log"
)

var DataLogEnabled = true
var LiveSyncEnabled = false

func SubscribedCommands() []string {
	commands := make([]string, 0)

	if DataLogEnabled {
		for k := range FetchAcceptedLoggerCommands() {
			if !contains(commands, k) {
				commands = append(commands, k)
			}
		}
	}

	if LiveSyncEnabled {
		for k := range FetchSyncCommands() {
			if !contains(commands, k) {
				commands = append(commands, k)
			}
		}
	}

	return commands
}

func OnReceiveApiEvent(command, request, response string) error {
	if !isCommandLoggerCommand(command) && !isProfileSyncCommand(command) {
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

	if DataLogEnabled && isCommandLoggerCommand(command) {
		wizardId, ok := tryExtractWizardId(requestContent, responseContent)
		if !ok {
			log.Error().Msg("Failed to get wizardId from API request/response.")
			return errors.New("failed to get wizardId from API request/response")
		}

		if err := UploadSwarfarmCommand(wizardId, command, requestContent, responseContent); err != nil {
			log.Error().Err(err).
				Str("swarfarmLogType", "data_log").
				Msg("Failed to upload SWARFARM data log command.")

			log.Debug().
				Str("swarfarmLogType", "data_log").
				Str("request", request).
				Str("response", response).
				Msg("Details of the failed data log command")
		}
	}

	if LiveSyncEnabled && isProfileSyncCommand(command) {
		wizardId, ok := tryExtractWizardId(requestContent, responseContent)
		if !ok {
			log.Error().Msg("Failed to get wizardId from API request/response.")
			return errors.New("failed to get wizardId from API request/response")
		}

		if err := UploadSwarfarmLiveSyncCommand(wizardId, command, requestContent, responseContent); err != nil {
			log.Error().Err(err).
				Str("swarfarmLogType", "profile_sync").
				Msg("Failed to upload SWARFARM profile sync command.")

			log.Debug().
				Str("swarfarmLogType", "profile_sync").
				Str("request", request).
				Str("response", response).
				Msg("Details of the failed sync command")
		}
	}

	return errors.New("unknown command")
}

func tryExtractWizardId(request, response map[string]interface{}) (wizardId int64, ok bool) {
	// try to extract wizardId using the request
	if wizardIdField, found := request["wizard_id"]; found {
		if wizardId, ok := wizardIdField.(float64); ok {
			return int64(wizardId), true
		}
	}

	// try to extract wizardId using the response directly
	if wizardIdField, found := response["wizard_id"]; found {
		if wizardId, ok := wizardIdField.(float64); ok {
			return int64(wizardId), true
		}
	}

	// try to extract wizardId using the response and the wizard_info field
	if wizardInfoField, found := response["wizard_info"]; found {
		if wizardInfo, ok := wizardInfoField.(map[string]interface{}); ok {
			if wizardIdField, found := wizardInfo["wizard_id"]; found {
				if wizardId, ok := wizardIdField.(float64); ok {
					return int64(wizardId), true
				}
			}
		}
	}

	return -1, false
}

func isCommandLoggerCommand(command string) bool {
	for k := range FetchAcceptedLoggerCommands() {
		if k == command {
			return true
		}
	}
	return false
}

func isProfileSyncCommand(command string) bool {
	for k := range FetchSyncCommands() {
		if k == command {
			return true
		}
	}
	return false
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
