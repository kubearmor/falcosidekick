package outputs

import (
	"bytes"
	"fmt"
	"log"

	"github.com/falcosecurity/falcosidekick/types"
)

func newRocketchatPayload(kubearmorpayload types.KubearmorPayload, config *types.Configuration) slackPayload {
	var (
		messageText string
		attachments []slackAttachment
		attachment  slackAttachment
		fields      []slackAttachmentField
		field       slackAttachmentField
	)

	if config.Rocketchat.OutputFormat == All || config.Rocketchat.OutputFormat == Fields || config.Rocketchat.OutputFormat == "" {
		field.Title = Priority
		field.Value = kubearmorpayload.EventType
		field.Short = true
		fields = append(fields, field)
		field.Title = Source
		field.Value = kubearmorpayload.OutputFields["PodName"].(string)
		field.Short = true
		fields = append(fields, field)

		for _, i := range getSortedStringKeys(kubearmorpayload.OutputFields) {
			j := kubearmorpayload.OutputFields[i]
			switch v := j.(type) {
			case string:
				field.Title = i
				field.Value = kubearmorpayload.OutputFields[i].(string)
				if len([]rune(kubearmorpayload.OutputFields[i].(string))) < 36 {
					field.Short = true
				} else {
					field.Short = false
				}
				fields = append(fields, field)
			default:
				vv := fmt.Sprint(v)
				field.Title = i
				field.Value = vv
				if len([]rune(vv)) < 36 {
					field.Short = true
				} else {
					field.Short = false
				}
				fields = append(fields, field)
			}
		}

		field.Title = Time
		field.Short = false
		field.Value = fmt.Sprint(kubearmorpayload.Timestamp)
		fields = append(fields, field)
		if kubearmorpayload.Hostname != "" {
			field.Title = Hostname
			field.Value = kubearmorpayload.Hostname
			field.Short = true
			fields = append(fields, field)
		}
	}

	if config.Rocketchat.MessageFormatTemplate != nil {
		buf := &bytes.Buffer{}
		if err := config.Rocketchat.MessageFormatTemplate.Execute(buf, kubearmorpayload); err != nil {
			log.Printf("[ERROR] : RocketChat - Error expanding RocketChat message %v", err)
		} else {
			messageText = buf.String()
		}
	}

	if config.Rocketchat.OutputFormat == All || config.Rocketchat.OutputFormat == Fields || config.Rocketchat.OutputFormat == "" {
		var color string
		switch kubearmorpayload.EventType {
		case "Alert":
			color = Orange
		case "Log":
			color = LigthBlue
		}
		attachment.Color = color

		attachments = append(attachments, attachment)
	}

	iconURL := DefaultIconURL
	if config.Rocketchat.Icon != "" {
		iconURL = config.Rocketchat.Icon
	}

	s := slackPayload{
		Text:        messageText,
		Username:    "Kubearmor",
		IconURL:     iconURL,
		Attachments: attachments}

	return s
}

// RocketchatPost posts event to Rocketchat
func (c *Client) RocketchatPost(kubearmorpayload types.KubearmorPayload) {
	c.Stats.Rocketchat.Add(Total, 1)

	err := c.Post(newRocketchatPayload(kubearmorpayload, c.Config))
	if err != nil {
		go c.CountMetric(Outputs, 1, []string{"output:rocketchat", "status:error"})
		c.Stats.Rocketchat.Add(Error, 1)
		c.PromStats.Outputs.With(map[string]string{"destination": "rocketchat", "status": Error}).Inc()
		log.Printf("[ERROR] : RocketChat - %v\n", err.Error())
		return
	}

	// Setting the success status
	go c.CountMetric(Outputs, 1, []string{"output:rocketchat", "status:ok"})
	c.Stats.Rocketchat.Add(OK, 1)
	c.PromStats.Outputs.With(map[string]string{"destination": "rocketchat", "status": OK}).Inc()
}
