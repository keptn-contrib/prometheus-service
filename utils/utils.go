package utils

import (
	"fmt"
	"net/url"
	"os"
)

const eventbroker = "EVENTBROKER"

// GetServiceEndpoint retrieves an endpoint stored in an environment variable and sets http as default scheme
func GetServiceEndpoint(service string) (url.URL, error) {
	url, err := url.Parse(os.Getenv(service))
	if err != nil {
		return *url, fmt.Errorf("Failed to retrieve value from ENVIRONMENT_VARIABLE: %s", service)
	}

	if url.Scheme == "" {
		url.Scheme = "http"
	}

	return *url, nil
}

// GetEventBrokerURL godoc
func GetEventBrokerURL() (string, error) {
	var eventBrokerURL string
	endpoint, err := GetServiceEndpoint(eventbroker)
	if err != nil {
		eventBrokerURL = "http://localhost:8081/event"
		return "", fmt.Errorf("Could not parse EVENTBROKER URL %s: %s. Using default: %s", os.Getenv(eventbroker), err.Error(), eventBrokerURL)
	} else {
		eventBrokerURL = endpoint.String()
		return "", fmt.Errorf("EVENTBROKER URL: %s", eventBrokerURL)
	}
	return eventBrokerURL, nil
}
