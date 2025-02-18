package outputs

import (
	"log"
	"strings"

	"github.com/kubearmor/sidekick/types"
)

// WebhookPost posts event to an URL
func (c *Client) WebhookPost(kubearmorpayload types.KubearmorPayload) {
	c.Stats.Webhook.Add(Total, 1)

	if len(c.Config.Webhook.CustomHeaders) != 0 {
		c.httpClientLock.Lock()
		defer c.httpClientLock.Unlock()
		for i, j := range c.Config.Webhook.CustomHeaders {
			c.AddHeader(i, j)
		}
	}
	var err error
	if strings.ToUpper(c.Config.Webhook.Method) == HttpPut {
		err = c.Put(kubearmorpayload)
	} else {
		err = c.Post(kubearmorpayload)
	}

	if err != nil {
		go c.CountMetric(Outputs, 1, []string{"output:webhook", "status:error"})
		c.Stats.Webhook.Add(Error, 1)
		c.PromStats.Outputs.With(map[string]string{"destination": "webhook", "status": Error}).Inc()
		log.Printf("[ERROR] : WebHook - %v\n", err.Error())
		return
	}

	// Setting the success status
	go c.CountMetric(Outputs, 1, []string{"output:webhook", "status:ok"})
	c.Stats.Webhook.Add(OK, 1)
	c.PromStats.Outputs.With(map[string]string{"destination": "webhook", "status": OK}).Inc()
}
