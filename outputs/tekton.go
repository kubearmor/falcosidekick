package outputs

import (
	"log"

	"github.com/kubearmor/sidekick/types"
)

// TektonPost posts event to EventListner
func (c *Client) TektonPost(kubearmorpayload types.KubearmorPayload) {
	c.Stats.Tekton.Add(Total, 1)

	err := c.Post(kubearmorpayload)
	if err != nil {
		go c.CountMetric(Outputs, 1, []string{"output:tekton", "status:error"})
		c.Stats.Tekton.Add(Error, 1)
		c.PromStats.Outputs.With(map[string]string{"destination": "tekton", "status": Error}).Inc()
		log.Printf("[ERROR] : Tekton - %v\n", err.Error())
		return
	}

	// Setting the success status
	go c.CountMetric(Outputs, 1, []string{"output:tekton", "status:ok"})
	c.Stats.Tekton.Add(OK, 1)
	c.PromStats.Outputs.With(map[string]string{"destination": "tekton", "status": OK}).Inc()
}
