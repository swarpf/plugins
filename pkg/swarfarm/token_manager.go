package swarfarm

import (
	"errors"
	"strings"
)

var apiTokenMap map[string]string

func AddProfile(summonerId, token string) {
	if apiTokenMap == nil {
		apiTokenMap = make(map[string]string)
	}

	if strings.TrimSpace(summonerId) == "" {
		return
	}

	apiTokenMap[summonerId] = token
}

func FindToken(summonerId string) (string, error) {
	if apiTokenMap == nil {
		apiTokenMap = make(map[string]string)
	}

	if strings.TrimSpace(summonerId) == "" {
		return "", errors.New("summonerId is empty")
	}

	token, ok := apiTokenMap[summonerId]

	if !ok {
		return "", errors.New("no associated token found")
	}

	return token, nil
}
