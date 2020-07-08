package profileexport

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sort"

	"github.com/rs/zerolog/log"
)

var outputDirectory string

func SubscribedCommands() []string {
	return []string{"HubUserLogin", "GuestLogin"}
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

func OnReceiveApiEvent(command, _, response string) error {
	if !isSubscribedCommand(command) {
		return nil
	}

	responseContent := map[string]interface{}{}
	if err := json.Unmarshal([]byte(response), &responseContent); err != nil {
		log.Error().Err(err).Msg("Failed to deserializie profile export response")
		return errors.New("error while deserializing profile export response")
	}

	wizardInfo := responseContent["wizard_info"].(map[string]interface{})
	wizardId := wizardInfo["wizard_id"].(float64)
	wizardName := wizardInfo["wizard_name"].(string)

	log.Info().
		Str("command", command).
		Float64("wizardId", wizardId).
		Str("wizardName", wizardName).
		Msg("Received command used in profile export")

	// check data integrity
	dataIsOk := checkData(responseContent)
	if !dataIsOk {
		log.Error().Msg("Some data in the API response is missing.")
		return errors.New("received incomplete data from API")
	}

	// sort data
	sortedData := sortData(responseContent)

	// serialize sorted data back to json
	jsonBytes, err := json.Marshal(sortedData)
	if err != nil {
		log.Error().Err(err).
			Float64("wizardId", wizardId).
			Msg("Something went wrong while re-serializing the API response.")
		return errors.New("serialization failed - sorted data is corrupt")
	}

	// write sorted data to profile file
	filePath := fmt.Sprintf("%v/%v-%v.json", GetOutputDirectory(), wizardName, wizardId)
	err = ioutil.WriteFile(filePath, jsonBytes, 0664)
	if err != nil {
		log.Error().Err(err).
			Float64("wizardId", wizardId).
			Str("filePath", filePath).
			Msg("Could not write profile JSON to file")
		return fmt.Errorf("failed to write profile to file, error: %v", err.Error())
	}

	log.Info().
		Float64("wizardId", wizardId).
		Str("filePath", filePath).
		Msgf("Profile successfully exported to %s", filePath)

	return nil
}

func checkData(data map[string]interface{}) bool {
	_, hasBuildingList := data["building_list"]

	return hasBuildingList
}

func sortData(data map[string]interface{}) map[string]interface{} {
	// find storage building
	var storageId uint64 = 999
	buildingList := data["building_list"].([]interface{})

	for _, entry := range buildingList {
		building := entry.(map[string]interface{})

		buildingMasterId := uint64(building["building_master_id"].(float64))
		buildingId := uint64(building["building_id"].(float64))

		if buildingMasterId == 25 {
			storageId = buildingId
		}
	}

	// sort unit list
	unitListRef := data["unit_list"].([]interface{})
	sort.Slice(unitListRef, func(i, j int) bool {
		a := newJsonUnit(unitListRef[i].(map[string]interface{}))
		b := newJsonUnit(unitListRef[j].(map[string]interface{}))

		if a.BuildingId == storageId || b.BuildingId == storageId {
			aIsStorage := a.BuildingId == storageId
			bIsStorage := b.BuildingId == storageId
			if aIsStorage && !bIsStorage {
				return true
			} else if !aIsStorage && bIsStorage {
				return false
			}
		}

		if abs(int64(b.Class-a.Class)) != 0 {
			return a.Class < b.Class
		}

		if abs(int64(b.UnitLevel-a.UnitLevel)) != 0 {
			return a.UnitLevel < b.UnitLevel
		}

		if abs(int64(a.Attribute-b.Attribute)) != 0 {
			return a.Attribute > b.Attribute
		}

		if abs(int64(a.UnitId-b.UnitId)) != 0 {
			return a.UnitId > b.UnitId
		}

		return false
	})

	// reverse sorting
	for i := len(unitListRef)/2 - 1; i >= 0; i-- {
		opp := len(unitListRef) - 1 - i
		unitListRef[i], unitListRef[opp] = unitListRef[opp], unitListRef[i]
	}

	// sort runes on monsters by slot
	for _, entry := range unitListRef {
		unit := entry.(map[string]interface{})

		sortedUnitRunes := sortRunes(unit["runes"])
		unit["runes"] = sortedUnitRunes
	}

	// sort runes in inventory by slot
	sortedRunes := sortRunes(data["runes"])
	data["runes"] = sortedRunes

	// sort craft items
	craftItems := data["rune_craft_item_list"].([]interface{})
	sort.Sort(CraftItemsByTypeAndId(craftItems))

	return data
}

func sortRunes(runeElement interface{}) interface{} {
	runes, ok := runeElement.([]interface{})
	if !ok {
		runes = make([]interface{}, 0)
		for _, v := range runeElement.(map[string]interface{}) {
			runes = append(runes, v)
		}
	}
	sort.Sort(RunesBySlot(runes))
	return runes
}
