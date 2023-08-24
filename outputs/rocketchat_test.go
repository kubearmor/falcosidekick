package outputs

import (
	"encoding/json"
	"testing"
	"text/template"

	"github.com/stretchr/testify/require"

	"github.com/kubearmor/sidekick/types"
)

func TestNewRocketchatPayload(t *testing.T) {
	expectedOutput := slackPayload{
		Text:     "Rule: Test rule Priority: Debug",
		Username: "Kubearmor",
		IconURL:  DefaultIconURL,
		Attachments: []slackAttachment{
			{
				Fallback: "This is a test from kubearmor",
				Color:    PaleCyan,
				Text:     "This is a test from kubearmor",
				Footer:   "",
				Fields: []slackAttachmentField{
					{
						Title: "rule",
						Value: "Test rule",
						Short: true,
					},
					{
						Title: "priority",
						Value: "Debug",
						Short: true,
					},
					{
						Title: "source",
						Value: "syscalls",
						Short: true,
					},
					{
						Title: "tags",
						Value: "test, example",
						Short: true,
					},
					{
						Title: "proc.name",
						Value: "falcosidekick",
						Short: true,
					},
					{
						Title: "time",
						Value: "2001-01-01 01:10:00 +0000 UTC",
						Short: false,
					},
					{
						Title: "hostname",
						Value: "test-host",
						Short: true,
					},
				},
			},
		},
	}

	var f types.KubearmorPayload
	require.Nil(t, json.Unmarshal([]byte(falcoTestInput), &f))
	config := &types.Configuration{
		Rocketchat: types.RocketchatOutputConfig{
			Username: "Kubearmor",
			Icon:     DefaultIconURL,
		},
	}

	var err error
	config.Rocketchat.MessageFormatTemplate, err = template.New("").Parse("Rule: {{ .Rule }} Priority: {{ .Priority }}")
	require.Nil(t, err)

	output := newRocketchatPayload(f, config)
	require.Equal(t, output, expectedOutput)
}
