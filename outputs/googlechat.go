package outputs

import (
	"bytes"
	"fmt"
	"log"

	"github.com/kubearmor/sidekick/types"
)

type header struct {
	Title    string `json:"title"`
	SubTitle string `json:"subtitle"`
}

type keyValue struct {
	TopLabel string `json:"topLabel"`
	Content  string `json:"content"`
}

type widget struct {
	KeyValue keyValue `json:"keyValue,omitempty"`
}

type section struct {
	Widgets []widget `json:"widgets"`
}

type card struct {
	Header   header    `json:"header,omitempty"`
	Sections []section `json:"sections,omitempty"`
}

type googlechatPayload struct {
	Text  string `json:"text,omitempty"`
	Cards []card `json:"cards,omitempty"`
}

func newGooglechatPayload(kubearmorpayload types.KubearmorPayload, config *types.Configuration) googlechatPayload {
	var messageText string
	widgets := []widget{}

	if config.Googlechat.MessageFormatTemplate != nil {
		buf := &bytes.Buffer{}
		if err := config.Googlechat.MessageFormatTemplate.Execute(buf, kubearmorpayload); err != nil {
			log.Printf("[ERROR] : GoogleChat - Error expanding Google Chat message %v", err)
		} else {
			messageText = buf.String()
		}
	}

	if config.Googlechat.OutputFormat == Text {
		return googlechatPayload{
			Text: messageText,
		}
	}

	for _, i := range getSortedStringKeys(kubearmorpayload.OutputFields) {
		widgets = append(widgets, widget{
			KeyValue: keyValue{
				TopLabel: i,
				Content:  fmt.Sprint(kubearmorpayload.OutputFields[i]),
			},
		})
	}

	widgets = append(widgets, widget{KeyValue: keyValue{"priority", kubearmorpayload.EventType}})
	widgets = append(widgets, widget{KeyValue: keyValue{"source pod", kubearmorpayload.OutputFields["PodName"].(string)}})

	if kubearmorpayload.Hostname != "" {
		widgets = append(widgets, widget{KeyValue: keyValue{Hostname, kubearmorpayload.Hostname}})
	}

	widgets = append(widgets, widget{KeyValue: keyValue{"time", fmt.Sprint(kubearmorpayload.Timestamp)}})

	return googlechatPayload{
		Text: messageText,
		Cards: []card{
			{
				Sections: []section{
					{Widgets: widgets},
				},
			},
		},
	}
}

// GooglechatPost posts event to Google Chat
func (c *Client) GooglechatPost(kubearmorpayload types.KubearmorPayload) {
	c.Stats.GoogleChat.Add(Total, 1)

	err := c.Post(newGooglechatPayload(kubearmorpayload, c.Config))
	if err != nil {
		go c.CountMetric(Outputs, 1, []string{"output:googlechat", "status:error"})
		c.Stats.GoogleChat.Add(Error, 1)
		c.PromStats.Outputs.With(map[string]string{"destination": "googlechat", "status": Error}).Inc()
		log.Printf("[ERROR] : GoogleChat - %v\n", err)
		return
	}

	go c.CountMetric(Outputs, 1, []string{"output:googlechat", "status:ok"})
	c.Stats.GoogleChat.Add(OK, 1)
	c.PromStats.Outputs.With(map[string]string{"destination": "googlechat", "status": OK}).Inc()
}
