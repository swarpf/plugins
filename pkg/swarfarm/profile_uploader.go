package swarfarm

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
)

func GetProfileUploadCommands() []string {
	return []string{"HubUserLogin"}
}

func UploadSwarfarmProfile(wizardId int64, command, apiResponse string) error {
	apiToken, _ := FindToken(strconv.FormatInt(wizardId, 10))

	log.Info().
		Str("command", command).
		Int64("wizard_id", wizardId).
		Msg("Uploading profile to SWARFARM...")

	client := makeAuthorizedClient(apiToken)
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(apiResponse).
		Post(apiUrl + "/profiles/upload/")

	if err != nil {
		log.Error().Err(err).
			Str("command", command).
			Int64("wizardId", wizardId).
			Msg("Failed to upload profile to SWARFARM.. Could not sent request")
		return errors.New("failed to send SWARFARM request")
	}

	if resp.StatusCode() != http.StatusOK {
		log.Error().
			Str("command", command).
			Int64("wizardId", wizardId).
			Int("StatusCode", resp.StatusCode()).
			Msgf("SWARFARM upload failed. Status %d", resp.StatusCode())
		return fmt.Errorf("SWARFARM upload failed with status %d", resp.StatusCode())
	}

	swarfarmResponse := map[string]interface{}{}
	err = json.Unmarshal(resp.Body(), &swarfarmResponse)
	if err != nil {
		log.Error().
			Str("command", command).
			Int64("wizardId", wizardId).
			Msgf("Error while deserializing SWARFARM profile upload response")
		return errors.New("error while deserializing SWARFARM profile upload response")
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		jobId := swarfarmResponse["job_id"].(string)
		log.Info().
			Str("command", command).
			Int64("wizardId", wizardId).
			Str("jobId", jobId).
			Msg("SWARFARM profile successfully uploaded - awaiting import queue")

		// run job check routine
		go func(jobId string, apiKey string) {
			const maxRetires = 3
			client := makeAuthorizedClient(apiKey)

			for retries := 0; retries < maxRetires; retries++ {
				resp, err := client.R().
					SetHeader("Content-Type", "application/json").
					Get(apiUrl + fmt.Sprintf("/profiles/upload/%s/", jobId))

				if err != nil {
					log.Warn().Err(err).
						Str("jobId", jobId).
						Msg("Failed to check SWARFARM profile upload status..")
				}

				if resp.StatusCode() == http.StatusUnauthorized {
					log.Error().
						Str("jobId", jobId).
						Msg("Could not check for job - access unauthorized")
				} else if resp.StatusCode() == http.StatusOK {
					content := map[string]interface{}{}
					err = json.Unmarshal(resp.Body(), &content)
					if err != nil {
						if content["status"].(string) == "SUCCESS" {
							log.Info().
								Str("jobId", jobId).
								Msg("SWARFARM profile import complete!")
							return
						}
					}

					log.Error().Err(err).
						Str("jobId", jobId).
						Msg("Error while deserializing SWARFARM profile upload check response")
				}

				time.Sleep(10 * time.Second)
			}

			log.Error().
				Str("jobId", jobId).
				Int("maxRetires", maxRetires).
				Msg("Aborting upload - too many failed retries")
		}(jobId, apiToken)

	case http.StatusBadRequest:
		log.Error().
			Str("command", command).
			Int64("wizardId", wizardId).
			Bytes("body", resp.Body()).
			Msg("Upload failed, invalid data provided")
	case http.StatusUnauthorized:
		log.Error().
			Str("command", command).
			Int64("wizardId", wizardId).
			Bytes("body", resp.Body()).
			Msg("Unable to authorize. Please check your API key")
	case http.StatusConflict:
		log.Error().
			Str("command", command).
			Int64("wizardId", wizardId).
			Bytes("body", resp.Body()).
			Msg("You need to upload your profile manually to resolve this")
	default:
		log.Error().
			Str("command", command).
			Int64("wizardId", wizardId).
			Stringer("resp", resp).
			Msg("Received unknown response type")
	}

	log.Info().
		Str("command", command).
		Int64("wizardId", wizardId).
		Msg("SWAG upload successful.")

	return nil
}
