package outputs

import (
	"encoding/json"
	"testing"
	"text/template"

	"github.com/stretchr/testify/require"

	"github.com/falcosecurity/falcosidekick/types"
)

func TestMattermostPayload(t *testing.T) {
	expectedOutput := slackPayload{
		Text:     "Rule: Test rule Priority: Debug",
		Username: "Kubearmor",
		IconURL:  "https://github.com/kubearmor/KubeArmor/assets/47106543/2db0b636-5c82-49c0-bf7d-535e4ad0a991",
		Attachments: []slackAttachment{
			{
				Fallback: "This is a test from Kubearmor",
				Color:    "#ccfff2",
				Text:     "This is a test from kubearmor",
				Footer:   "https://github.com/kubearmor/kubearmor",
				Fields: []slackAttachmentField{
					{
						Title: "rule",
						Value: "Test rule",
						Short: true,
					},
					{
						Title: "hostname",
						Value: "test-host",
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
						Value: "kubearmor",
						Short: true,
					},
					{
						Title: "time",
						Value: "2001-01-01 01:10:00 +0000 UTC",
						Short: false,
					},
				},
			},
		},
	}

	var f types.KubearmorPayload
	require.Nil(t, json.Unmarshal([]byte(falcoTestInput), &f))
	config := &types.Configuration{
		Mattermost: types.MattermostOutputConfig{
			Username: "Kubearmor",
			Icon:     "https://github.com/kubearmor/KubeArmor/assets/47106543/2db0b636-5c82-49c0-bf7d-535e4ad0a991",
		},
	}

	var err error
	config.Mattermost.MessageFormatTemplate, err = template.New("").Parse("Rule: {{ .Rule }} Priority: {{ .Priority }}")
	require.Nil(t, err)

	output := newMattermostPayload(f, config)
	require.Equal(t, output, expectedOutput)
}
