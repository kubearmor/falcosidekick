package outputs

import (
	"log"

	"github.com/kubearmor/sidekick/types"
)

// OpenObservePost posts event to OpenObserve
func (c *Client) OpenObservePost(kubearmorpayload types.KubearmorPayload) {
	c.Stats.OpenObserve.Add(Total, 1)

	if c.Config.OpenObserve.Username != "" && c.Config.OpenObserve.Password != "" {
		c.httpClientLock.Lock()
		defer c.httpClientLock.Unlock()
		c.BasicAuth(c.Config.OpenObserve.Username, c.Config.OpenObserve.Password)
	}

	for i, j := range c.Config.OpenObserve.CustomHeaders {
		c.AddHeader(i, j)
	}

	if err := c.Post(kubearmorpayload); err != nil {
		c.setOpenObserveErrorMetrics()
		log.Printf("[ERROR] : OpenObserve - %v\n", err)
		return
	}

	// Setting the success status
	go c.CountMetric(Outputs, 1, []string{"output:openobserve", "status:ok"})
	c.Stats.OpenObserve.Add(OK, 1)
	c.PromStats.Outputs.With(map[string]string{"destination": "openobserve", "status": OK}).Inc()
}

// setOpenObserveErrorMetrics set the error stats
func (c *Client) setOpenObserveErrorMetrics() {
	go c.CountMetric(Outputs, 1, []string{"output:openobserve", "status:error"})
	c.Stats.OpenObserve.Add(Error, 1)
	c.PromStats.Outputs.With(map[string]string{"destination": "openobserve", "status": Error}).Inc()
}
