package siegeexport

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/rs/zerolog/log"
)

var outputDirectory string
var exportData map[string]interface{}

func SubscribedCommands() []string {
	return []string{"GetGuildSiegeMatchupInfo", "GetGuildSiegeBattleLog",
		"GetGuildSiegeBaseDefenseUnitList", "GetGuildSiegeBaseDefenseUnitListPreset"}
}

func isSubscribedCommand(command string) bool {
	for _, b := range SubscribedCommands() {
		if b == command {
			return true
		}
	}
	return false
}

func GetOutputDirectory() string {
	if outputDirectory == "" {
		SetOutputDirectory(".")
	}

	return outputDirectory
}

func SetOutputDirectory(directory string) {
	localLogger := log.With().Str("outputDirectory", directory).Logger()

	// if the parameter is empty use the current working directory as output directory
	if directory == "" {
		directory = "."
	}

	// create the directory if it doesn't exist
	fi, err := os.Stat(directory)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(directory, 0755)
			if err != nil {
				localLogger.Fatal().Err(err).Msg("failed to create output directory")
			}
		} else {
			localLogger.Fatal().Err(err).Msg("failed to get file path information")
		}
	}

	if !fi.IsDir() {
		localLogger.Fatal().Msg("output path exists but is not a directory")
	}

	outputDirectory = directory
}

func OnReceiveApiEvent(command, request, response string) error {
	if !isSubscribedCommand(command) {
		return nil
	}

	requestContent := map[string]interface{}{}
	if err := json.Unmarshal([]byte(request), &requestContent); err != nil {
		log.Error().Err(err).Msg("Failed to deserializie siege export request")
		return errors.New("error while deserializing siege export request")
	}

	responseContent := map[string]interface{}{}
	if err := json.Unmarshal([]byte(response), &responseContent); err != nil {
		log.Error().Err(err).Msg("Failed to deserializie siege export response")
		return errors.New("error while deserializing siege export response")
	}

	wizardId := int64(requestContent["wizard_id"].(float64))

	localLogger := log.With().
		Str("command", command).
		Int64("wizardId", wizardId).
		Logger()

	localLogger.Info().Msg("Received command used in siege export")

	if exportData == nil {
		exportData = make(map[string]interface{})
	}

	switch command {
	case "GetGuildSiegeMatchupInfo":
		retCode := int64(responseContent["ret_code"].(float64))
		if retCode == 0 {
			exportData["wizard_id"] = wizardId
			exportData["matchup_info"] = responseContent
			matchId := int64(responseContent["match_info"].(map[string]interface{})["match_id"].(float64))
			if err := writeSiegeMatchToFile(&matchId, exportData); err != nil {
				return err
			}
		}
	case "GetGuildSiegeBattleLog":
		logType := int64(requestContent["log_type"].(float64))

		logList := responseContent["log_list"].([]interface{})[0].(map[string]interface{})
		guildInfoList := logList["guild_info_list"].([]interface{})[0].(map[string]interface{})
		matchId := int64(guildInfoList["match_id"].(float64))

		localLogger := localLogger.With().Int64("logType", logType).Interface("matchId", matchId).Logger()

		if logType == 1 {
			exportData["attack_log"] = responseContent
			localLogger.Info().Msg("Writing attack log to file")
		} else {
			exportData["defense_log"] = responseContent
			localLogger.Info().Msg("Writing defense log to file")
		}

		if err := writeSiegeMatchToFile(&matchId, exportData); err != nil {
			return err
		}
	case "GetGuildSiegeBaseDefenseUnitList", "GetGuildSiegeBaseDefenseUnitListPreset":
		const (
			redHqId    = 1
			blueHqId   = 14
			yellowHqId = 27
		)

		baseNumber := int64(requestContent["base_number"].(float64))
		if baseNumber == redHqId || baseNumber == blueHqId || baseNumber == yellowHqId {
			exportData["defense_list"] = responseContent
			exportData["defense_list"].(map[string]interface{})["hq_base_number"] = requestContent["base_number"]

			localLogger.Info().Msg("Writing defense list to file")
			if err := writeSiegeMatchToFile(nil, exportData); err != nil {
				return err
			}
		}
	default:
		localLogger.Warn().Msg("Received unexpected command. This should never happen.")
	}

	return nil
}

func writeSiegeMatchToFile(matchId *int64, data map[string]interface{}) error {
	// serialize sorted data back to json
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		log.Error().Err(err).
			Interface("matchId", matchId).
			Msg("Something went wrong while re-serializing the API response.")
		return errors.New("serialization failed - sorted data is corrupt")
	}

	// generate file name to write to
	var fileName string
	if matchId != nil {
		fileName = fmt.Sprintf("SiegeMatch-%d.json", *matchId)
	} else {
		fileName = "SiegeDefenseList.json"
	}

	// write match data to profile file
	filePath := fmt.Sprintf("%s/%s", GetOutputDirectory(), fileName)
	err = ioutil.WriteFile(filePath, jsonBytes, 0664)
	if err != nil {
		log.Error().Err(err).
			Interface("matchId", matchId).
			Str("filePath", filePath).
			Msg("Could not write profile JSON to file")
		return fmt.Errorf("failed to write profile to file, error: %v", err.Error())
	}

	log.Info().
		Interface("matchId", matchId).
		Str("filePath", filePath).
		Msgf("Siege data successfully written to %s", filePath)

	return nil
}
