package outputs

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/PagerDuty/go-pagerduty"

	"github.com/kubearmor/sidekick/types"
)

const (
	USEndpoint string = "https://events.pagerduty.com"
	EUEndpoint string = "https://events.eu.pagerduty.com"
)

// PagerdutyPost posts alert event to Pagerduty
func (c *Client) PagerdutyPost(kubearmorpayload types.KubearmorPayload) {
	c.Stats.Pagerduty.Add(Total, 1)

	event := createPagerdutyEvent(kubearmorpayload, c.Config.Pagerduty)

	if strings.ToLower(c.Config.Pagerduty.Region) == "eu" {
		pagerduty.WithV2EventsAPIEndpoint(EUEndpoint)
	} else {
		pagerduty.WithV2EventsAPIEndpoint(USEndpoint)
	}

	if _, err := pagerduty.ManageEventWithContext(context.Background(), event); err != nil {
		go c.CountMetric(Outputs, 1, []string{"output:pagerduty", "status:error"})
		c.Stats.Pagerduty.Add(Error, 1)
		c.PromStats.Outputs.With(map[string]string{"destination": "pagerduty", "status": Error}).Inc()
		log.Printf("[ERROR] : PagerDuty - %v\n", err)
		return
	}

	go c.CountMetric(Outputs, 1, []string{"output:pagerduty", "status:ok"})
	c.Stats.Pagerduty.Add(OK, 1)
	c.PromStats.Outputs.With(map[string]string{"destination": "pagerduty", "status": OK}).Inc()
	log.Printf("[INFO]  : Pagerduty - Create Incident OK\n")
}

func createPagerdutyEvent(kubearmorpayload types.KubearmorPayload, config types.PagerdutyConfig) pagerduty.V2Event {
	details := make(map[string]interface{}, len(kubearmorpayload.OutputFields)+4)
	details["priority"] = kubearmorpayload.EventType
	details["source"] = kubearmorpayload.OutputFields["PodName"].(string)
	if len(kubearmorpayload.Hostname) != 0 {
		kubearmorpayload.OutputFields[Hostname] = kubearmorpayload.Hostname
	}
	timestamp := time.Unix(kubearmorpayload.Timestamp, 0)
	event := pagerduty.V2Event{
		RoutingKey: config.RoutingKey,
		Action:     "trigger",
		Payload: &pagerduty.V2Payload{
			Source:    "Kubearmor",
			Summary:   kubearmorpayload.EventType + " for " + kubearmorpayload.OutputFields["PodName"].(string),
			Severity:  "critical",
			Timestamp: timestamp.Format(time.RFC3339),
			Details:   kubearmorpayload.OutputFields,
		},
	}
	return event
}
