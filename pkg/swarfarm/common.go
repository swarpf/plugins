package swarfarm

import (
	"fmt"

	"github.com/go-resty/resty/v2"
)

const (
	baseUrl = "https://swarfarm.com"
	apiUrl  = baseUrl + "/api/v2"
)

func makeAuthorizedClient(apiToken string) *resty.Client {
	client := resty.New()

	if apiToken != "" {
		client.SetHeader("Authorization", fmt.Sprintf("Token %s", apiToken))
	}

	return client
}
