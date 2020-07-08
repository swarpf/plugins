package swarfarm

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
)

func UploadSwarfarmCommand(wizardId int64, command string, request, response map[string]interface{}) error {
	apiToken, _ := FindToken(strconv.FormatInt(wizardId, 10))

	inputMap := make(map[string]map[string]interface{})
	inputMap["request"] = request
	inputMap["response"] = response

	log.Debug().
		Str("command", command).
		Int64("wizardId", wizardId).
		Msg("Uploading command data to SWARFARM")

	swarfarmCommand := make(map[string]map[string]interface{})

	acceptedCommands := FetchAcceptedLoggerCommands()
	cmdGroup := acceptedCommands[command]
	for direction := range cmdGroup {
		swarfarmCommand[direction] = make(map[string]interface{})
	}

	// handle request fields
	categories := []string{"request", "response"}
	for _, cat := range categories {
		requestCmds, ok := cmdGroup[cat]
		if !ok {
			continue
		}

		for _, c := range requestCmds {
			e, ok := inputMap[cat][c]

			if ok {
				swarfarmCommand[cat][c] = e
			} else {
				swarfarmCommand[cat][c] = nil
			}
		}
	}

	// handle response fields
	swarfarmCommandContent := make(map[string]interface{})
	swarfarmCommandContent["data"] = swarfarmCommand

	jsonBytes, err := json.Marshal(swarfarmCommandContent)
	if err != nil {
		log.Error().Err(err).
			Str("command", command).
			Int64("wizardId", wizardId).
			Msg("Error on command serialization")
		return errors.New("error while serializing a command")
	}

	rclient := makeAuthorizedClient(apiToken)
	resp, err := rclient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(jsonBytes).
		Post(baseUrl + "/data/log/upload/")

	if err != nil {
		log.Error().Err(err).
			Str("command", command).
			Int64("wizardId", wizardId).
			Msg("SWARFARM upload failed")
		return errors.New("SWARFARM upload failed")
	}

	if resp.StatusCode() != http.StatusOK {
		log.Error().
			Str("command", command).
			Int64("wizardId", wizardId).
			Int("statusCode", resp.StatusCode()).
			Msg("SWARFARM upload failed. Invalid status code.")
		return errors.New("SWARFARM upload failed - invalid status code")
	}

	log.Info().
		Str("command", command).
		Int64("wizardId", wizardId).
		Msg("SWARFARM Upload successful")

	return nil
}

var acceptedCommandCache map[string]map[string][]string

func FetchAcceptedLoggerCommands() map[string]map[string][]string {
	log.Debug().Msg("Retrieving list of accepted log types from SWARFARM...")

	if acceptedCommandCache != nil {
		log.Debug().Msg("Using cached version of accepted log types")
		return acceptedCommandCache
	}

	acceptedCommandCache := make(map[string]map[string][]string)

	rclient := resty.New()
	resp, err := rclient.R().Get(baseUrl + "/data/log/accepted_commands/")
	if err != nil {
		log.Error().Err(err).Msg("Unable to retrieve accepted log types. SWARFARM logging is disabled.")
		return acceptedCommandCache
	}

	if resp.StatusCode() != http.StatusOK {
		log.Error().Err(err).Msg("Unable to retrieve accepted log types. Invalid status code. SWARFARM logging is disabled.")
		return acceptedCommandCache
	}

	content := map[string]interface{}{}
	err = json.Unmarshal(resp.Body(), &content)
	if err != nil {
		log.Error().Err(err).Msg("Error while deserializing accepted SWARFARM commands.")
		return acceptedCommandCache
	}

	for k, v := range content {
		// skip non-commands
		if strings.HasPrefix(k, "__") {
			continue
		}

		contentCmd := v.(map[string]interface{})

		cmd := make(map[string][]string, 1)
		for cmdDirection, validValues := range contentCmd {
			for _, validValue := range validValues.([]interface{}) {
				cmd[cmdDirection] = append(cmd[cmdDirection], validValue.(string))
			}
		}

		acceptedCommandCache[k] = cmd
	}

	keys := make([]string, 0, len(acceptedCommandCache))
	for k := range acceptedCommandCache {
		keys = append(keys, k)
	}

	log.Info().Strs("accepted_commands", keys).
		Msg("Successfully retrieded list of SWARFARM accepted commands")

	return acceptedCommandCache
}
