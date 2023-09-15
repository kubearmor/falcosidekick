package outputs

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/kubearmor/sidekick/types"
)

func TestNewInfluxdbPayload(t *testing.T) {
	var testKubearmorPayload types.KubearmorPayload

	// Unmarshal the JSON string
	err := json.Unmarshal([]byte(TestInput), &testKubearmorPayload)
	if err != nil {
		t.Fatalf("Failed to unmarshal test input: %v", err)
	}

	// Sample configuration (assuming this is relevant for the function)
	config := &types.Configuration{}

	// Call the function
	gotPayload := newInfluxdbPayload(testKubearmorPayload, config)

	// Constructing the expected result based on the provided input

	expectedTags := []string{
		"rule",
		"priority",
		"source",
		"ContainerImage",
		"Hostname",
		"PodName",
		// Add any other tags you expect to be present...
	}

	if !containsAllTags(string(gotPayload), expectedTags) {
		t.Fatalf("Payload does not contain all the expected tags.")
	}
}

func containsAllTags(payload string, tags []string) bool {
	pairs := strings.Split(payload, ",")
	keys := make(map[string]bool)
	for _, pair := range pairs {
		key := strings.Split(pair, "=")[0]
		keys[key] = true
	}

	for _, tag := range tags {
		if _, exists := keys[tag]; !exists {
			return false
		}
	}
	return true
}
