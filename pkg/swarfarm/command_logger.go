package swarfarm

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
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

	acceptedCommands := FetchAcceptedLoggerCommands()
	cmdGroup := acceptedCommands[command]
	payload := makeUploadPayload(cmdGroup, inputMap)

	// handle response fields
	swarfarmCommandContent := make(map[string]interface{})
	swarfarmCommandContent["data"] = payload

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
		Post(apiUrl + "/data_logs/")

	if err != nil {
		log.Error().Err(err).
			Str("command", command).
			Int64("wizardId", wizardId).
			Msg("SWARFARM data log upload failed")
		return errors.New("SWARFARM data log upload failed")
	}

	if resp.StatusCode() != http.StatusOK {
		response := map[string]interface{}{}
		if err := json.Unmarshal(resp.Body(), &response); err != nil {
			log.Error().Err(err).Msg("Failed to deserializie SWARFARM response")
			return errors.New("error while deserializing SWARFARM response")
		}

		detail, ok := response["detail"].(string)
		if !ok {
			detail = "no detail"
		}

		errlog := log.Error().
			Str("command", command).
			Int64("wizardId", wizardId).
			Int("statusCode", resp.StatusCode()).
			Str("detail", detail)

		message := ""
		if resp.StatusCode() == http.StatusUnauthorized {
			message = fmt.Sprintf("SWARFARM data log upload failed - authentication error. detail: %s", detail)
		} else {
			message = fmt.Sprintf("SWARFARM data log upload failed - invalid status code. detail: %s", detail)
			errlog.Str("request_json_bytes", string(jsonBytes))
		}

		errlog.Msg(message)
		return errors.New(message)
	}

	log.Info().
		Str("command", command).
		Int64("wizardId", wizardId).
		Msg("SWARFARM data log upload successful")

	return nil
}

var acceptedCommandCache map[string]map[string][]string

func FetchAcceptedLoggerCommands() map[string]map[string][]string {
	if acceptedCommandCache != nil {
		log.Debug().Msg("Using cached version of accepted log types")
		return acceptedCommandCache
	}

	acceptedCommandCache = make(map[string]map[string][]string)
	acceptedCommandCache = buildCacheFromUrl("data log commands", fmt.Sprintf("%s%s", apiUrl, "/data_logs"))

	return acceptedCommandCache
}

func UploadSwarfarmLiveSyncCommand(wizardId int64, command string, request, response map[string]interface{}) error {
	if !LiveSyncEnabled {
		return nil
	}

	apiToken, _ := FindToken(strconv.FormatInt(wizardId, 10))
	if apiToken == "" {
		return nil
	}

	inputMap := make(map[string]map[string]interface{})
	inputMap["request"] = request
	inputMap["response"] = response

	log.Debug().
		Str("command", command).
		Int64("wizardId", wizardId).
		Msg("Uploading live sync data to SWARFARM")

	syncCommands := FetchSyncCommands()
	cmdGroup := syncCommands[command]
	payload := makeUploadPayload(cmdGroup, inputMap)

	// handle response fields
	swarfarmSyncContent := make(map[string]interface{})
	swarfarmSyncContent["data"] = payload

	jsonBytes, err := json.Marshal(swarfarmSyncContent)
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
		Post(apiUrl + "/profiles/sync/")

	if err != nil {
		log.Error().Err(err).
			Str("command", command).
			Int64("wizardId", wizardId).
			Msg("SWARFARM live sync upload failed")
		return errors.New("SWARFARM live sync upload failed")
	}

	if resp.StatusCode() != http.StatusOK {
		response := map[string]interface{}{}
		if err := json.Unmarshal(resp.Body(), &response); err != nil {
			log.Error().Err(err).Str("body", string(resp.Body())).
				Msg("Failed to deserializie SWARFARM response")
			return errors.New("error while deserializing SWARFARM response")
		}

		detail, ok := response["detail"].(string)
		if !ok {
			detail = "no detail"
		}

		message := ""
		if resp.StatusCode() == http.StatusUnauthorized {
			message = fmt.Sprintf("SWARFARM live sync upload failed - authentication error. detail: %s", detail)
		} else {
			message = fmt.Sprintf("SWARFARM live sync upload failed - invalid status code. detail: %s", detail)
		}

		log.Error().
			Str("command", command).
			Int64("wizardId", wizardId).
			Int("statusCode", resp.StatusCode()).
			Str("detail", detail).
			Msg(message)
		return errors.New(message)
	}

	log.Info().
		Str("command", command).
		Int64("wizardId", wizardId).
		Msg("SWARFARM live sync upload successful")

	return nil
}

var syncCommandCache map[string]map[string][]string

func FetchSyncCommands() map[string]map[string][]string {
	if !LiveSyncEnabled {
		return make(map[string]map[string][]string)
	}

	if syncCommandCache != nil {
		log.Debug().Msg("Using cached version of live sync commands")
		return syncCommandCache
	}

	syncCommandCache = make(map[string]map[string][]string)
	syncCommandCache = buildCacheFromUrl("live sync commands", fmt.Sprintf("%s%s", apiUrl, "/profiles/accepted-commands/"))

	return syncCommandCache
}

func buildCacheFromUrl(cacheTag, url string) map[string]map[string][]string {
	log.Debug().Msgf("Fetching %s from SWARFARM...", cacheTag)

	commandCache := make(map[string]map[string][]string)

	rclient := resty.New()
	resp, err := rclient.R().Get(url)
	if err != nil {
		log.Error().Err(err).Msgf("Unable to retrieve %s. SWARFARM logging is disabled.", cacheTag)
		return commandCache
	}

	if resp.StatusCode() == http.StatusNotFound {
		log.Error().
			Str("url", url).
			Msgf("Unable to retrieve %s. Endpoint not found. SWARFARM logging is disabled.", cacheTag)
		return commandCache
	}

	if resp.StatusCode() != http.StatusOK {
		log.Error().
			Int("status_code", resp.StatusCode()).
			Str("status", resp.Status()).
			Msgf("Unable to retrieve %s. Invalid status code. SWARFARM logging is disabled.", cacheTag)
		return commandCache
	}

	content := map[string]interface{}{}
	err = json.Unmarshal(resp.Body(), &content)
	if err != nil {
		log.Error().Err(err).Msgf("Error while deserializing SWARFARM %s.", cacheTag)
		return commandCache
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

		commandCache[k] = cmd
	}

	keys := make([]string, 0, len(commandCache))
	for k := range commandCache {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	log.Info().
		Str("cache_tag", cacheTag).
		Strs("commands", keys).
		Msgf("Successfully retrieved %s from SWARFARM.", cacheTag)

	return commandCache
}

func makeUploadPayload(cmdGroup map[string][]string, inputMap map[string]map[string]interface{}) map[string]map[string]interface{} {
	payload := make(map[string]map[string]interface{})

	for direction := range cmdGroup {
		payload[direction] = make(map[string]interface{})
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
				payload[cat][c] = e
			} else {
				payload[cat][c] = nil
			}
		}
	}

	return payload
}
