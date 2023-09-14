package outputs

import (
	"reflect"
	"testing"

	"github.com/kubearmor/sidekick/types"
)

func estNewSlackPayload(t *testing.T) {
	// Define a mock `types.KubearmorPayload`

	kubearmorPayload := types.KubearmorPayload{
		EventType: "Alert",
		Hostname:  "gke-kubearmor-prerelease-default-pool-6ad71e07-cd8r",
		Timestamp: 1631542902,
		OutputFields: map[string]interface{}{
			"PodName": "wordpress-7c966b5d85-xvsrl",
		},
	}

	config := &types.Configuration{
		Slack: types.SlackOutputConfig{
			OutputFormat: All,
			Username:     "Kubearmor",
			Footer:       "",
		},
	}

	// Call the function
	result := newSlackPayload(kubearmorPayload, config)

	// Define expected result
	expectedResult := slackPayload{
		Username: "Kubearmor",
		Attachments: []slackAttachment{
			{
				Footer: DefaultFooter,
				Fields: []slackAttachmentField{
					{Title: Priority, Value: "Alert", Short: true},
					{Title: Source, Value: "wordpress-7c966b5d85-xvsrl", Short: true},
					{Title: Hostname, Value: "gke-kubearmor-prerelease-default-pool-6ad71e07-cd8r", Short: true},
					{Title: Time, Value: "1631542902", Short: false},
				},
			},
		},
	}

	// Assert the result
	if !reflect.DeepEqual(result, expectedResult) {
		t.Errorf("Expected payload %+v but got %+v", expectedResult, result)
	}
}
