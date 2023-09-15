package outputs

import (
	"bytes"
	"fmt"
	"log"

	"github.com/kubearmor/sidekick/types"
)

func newMattermostPayload(kubearmorpayload types.KubearmorPayload, config *types.Configuration) slackPayload {
	var (
		messageText string
		attachments []slackAttachment
		attachment  slackAttachment
		fields      []slackAttachmentField
		field       slackAttachmentField
	)

	if config.Mattermost.OutputFormat == All || config.Mattermost.OutputFormat == Fields || config.Mattermost.OutputFormat == "" {
		field.Title = Priority
		field.Value = kubearmorpayload.EventType
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

		attachment.Footer = DefaultFooter
		if config.Mattermost.Footer != "" {
			attachment.Footer = config.Mattermost.Footer
		}
	}

	if config.Mattermost.MessageFormatTemplate != nil {
		buf := &bytes.Buffer{}
		if err := config.Mattermost.MessageFormatTemplate.Execute(buf, kubearmorpayload); err != nil {
			log.Printf("[ERROR] : Mattermost - Error expanding Mattermost message %v", err)
		} else {
			messageText = buf.String()
		}
	}

	var color string
	switch kubearmorpayload.EventType {
	case "Alert":
		color = Orange
	case "log":
		color = LigthBlue
	}
	attachment.Color = color

	attachments = append(attachments, attachment)

	iconURL := DefaultIconURL
	if config.Mattermost.Icon != "" {
		iconURL = config.Mattermost.Icon
	}

	s := slackPayload{
		Text:        messageText,
		Username:    "Kubearmor",
		IconURL:     iconURL,
		Attachments: attachments,
	}

	return s
}

// MattermostPost posts event to Mattermost
func (c *Client) MattermostPost(kubearmorpayload types.KubearmorPayload) {
	c.Stats.Mattermost.Add(Total, 1)

	err := c.Post(newMattermostPayload(kubearmorpayload, c.Config))
	if err != nil {
		go c.CountMetric(Outputs, 1, []string{"output:mattermost", "status:error"})
		c.Stats.Mattermost.Add(Error, 1)
		c.PromStats.Outputs.With(map[string]string{"destination": "mattermost", "status": Error}).Inc()
		log.Printf("[ERROR] : Mattermost - %v\n", err)
		return
	}

	// Setting the success status
	go c.CountMetric(Outputs, 1, []string{"output:mattermost", "status:ok"})
	c.Stats.Mattermost.Add(OK, 1)
	c.PromStats.Outputs.With(map[string]string{"destination": "mattermost", "status": OK}).Inc()
}
