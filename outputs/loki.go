package outputs

import (
	"fmt"
	"log"
	"strings"

	"github.com/kubearmor/sidekick/types"
)

type lokiPayload struct {
	Streams []lokiStream `json:"streams"`
}

type lokiStream struct {
	Stream map[string]string `json:"stream"`
	Values []lokiValue       `json:"values"`
}

type lokiValue = []string

// The Content-Type to send along with the request
const LokiContentType = "application/json"

func newLokiPayload(kubearmorpayload types.KubearmorPayload, config *types.Configuration) lokiPayload {
	s := make(map[string]string, 3+len(kubearmorpayload.OutputFields)+len(config.Loki.ExtraLabelsList))
	s["source"] = kubearmorpayload.OutputFields["PodName"].(string)
	s["priority"] = kubearmorpayload.EventType

	for i, j := range kubearmorpayload.OutputFields {
		switch v := j.(type) {
		case string:
			for k := range config.Customfields {
				if i == k {
					s[strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(i, ".", ""), "]", ""), "[", "")] = strings.ReplaceAll(v, "\"", "")
				}
			}
			for _, k := range config.Loki.ExtraLabelsList {
				if i == k {
					s[strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(i, ".", ""), "]", ""), "[", "")] = strings.ReplaceAll(v, "\"", "")
				}
			}
		default:
			vv := fmt.Sprint(v)
			for k := range config.Customfields {
				if i == k {
					s[strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(i, ".", ""), "]", ""), "[", "")] = strings.ReplaceAll(vv, "\"", "")
				}
			}
			for _, k := range config.Loki.ExtraLabelsList {
				if i == k {
					s[strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(i, ".", ""), "]", ""), "[", "")] = strings.ReplaceAll(vv, "\"", "")
				}
			}

		}
	}

	if kubearmorpayload.Hostname != "" {
		s[Hostname] = kubearmorpayload.Hostname
	}

	return lokiPayload{Streams: []lokiStream{
		{
			Stream: s,
			Values: []lokiValue{[]string{fmt.Sprintf("%v", kubearmorpayload.Timestamp)}},
		},
	}}
}

// LokiPost posts event to Loki
func (c *Client) LokiPost(kubearmorpayload types.KubearmorPayload) {
	c.Stats.Loki.Add(Total, 1)
	c.ContentType = LokiContentType
	if c.Config.Loki.Tenant != "" {
		c.httpClientLock.Lock()
		defer c.httpClientLock.Unlock()
		c.AddHeader("X-Scope-OrgID", c.Config.Loki.Tenant)
	}

	if c.Config.Loki.User != "" && c.Config.Loki.APIKey != "" {
		c.httpClientLock.Lock()
		defer c.httpClientLock.Unlock()
		c.BasicAuth(c.Config.Loki.User, c.Config.Loki.APIKey)
	}

	for i, j := range c.Config.Loki.CustomHeaders {
		c.AddHeader(i, j)
	}

	err := c.Post(newLokiPayload(kubearmorpayload, c.Config))
	if err != nil {
		go c.CountMetric(Outputs, 1, []string{"output:loki", "status:error"})
		c.Stats.Loki.Add(Error, 1)
		c.PromStats.Outputs.With(map[string]string{"destination": "loki", "status": Error}).Inc()
		log.Printf("[ERROR] : Loki - %v\n", err)
		return
	}

	go c.CountMetric(Outputs, 1, []string{"output:loki", "status:ok"})
	c.Stats.Loki.Add(OK, 1)
	c.PromStats.Outputs.With(map[string]string{"destination": "loki", "status": OK}).Inc()
}
