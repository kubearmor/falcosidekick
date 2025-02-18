package outputs

import (
	"fmt"
	"log"

	"github.com/kubearmor/sidekick/types"
)

// ZincsearchPost posts event to Zincsearch
func (c *Client) ZincsearchPost(kubearmorpayload types.KubearmorPayload) {
	c.Stats.Zincsearch.Add(Total, 1)

	if c.Config.Zincsearch.Username != "" && c.Config.Zincsearch.Password != "" {
		c.httpClientLock.Lock()
		defer c.httpClientLock.Unlock()
		c.BasicAuth(c.Config.Zincsearch.Username, c.Config.Zincsearch.Password)
	}

	fmt.Println(c.EndpointURL)
	err := c.Post(kubearmorpayload)
	if err != nil {
		c.setZincsearchErrorMetrics()
		log.Printf("[ERROR] : Zincsearch - %v\n", err)
		return
	}

	// Setting the success status
	go c.CountMetric(Outputs, 1, []string{"output:zincsearch", "status:ok"})
	c.Stats.Zincsearch.Add(OK, 1)
	c.PromStats.Outputs.With(map[string]string{"destination": "zincsearch", "status": OK}).Inc()
}

// setZincsearchErrorMetrics set the error stats
func (c *Client) setZincsearchErrorMetrics() {
	go c.CountMetric(Outputs, 1, []string{"output:zincsearch", "status:error"})
	c.Stats.Zincsearch.Add(Error, 1)
	c.PromStats.Outputs.With(map[string]string{"destination": "zincsearch", "status": Error}).Inc()
}
