package outputs

import (
	"encoding/json"
	"log"
	"strings"

	stan "github.com/nats-io/stan.go"

	"github.com/kubearmor/sidekick/types"
)

// StanPublish publishes event to NATS Streaming
func (c *Client) StanPublish(kubearmorpayload types.KubearmorPayload) {
	c.Stats.Stan.Add(Total, 1)

	nc, err := stan.Connect(c.Config.Stan.ClusterID, c.Config.Stan.ClientID, stan.NatsURL(c.EndpointURL.String()))
	if err != nil {
		c.setStanErrorMetrics()
		log.Printf("[ERROR] : STAN - %v\n", err.Error())
		return
	}
	defer nc.Close()

	j, err := json.Marshal(kubearmorpayload)
	if err != nil {
		c.setStanErrorMetrics()
		log.Printf("[ERROR] : STAN - %v\n", err.Error())
		return
	}

	err = nc.Publish("kubearmor."+strings.ToLower(kubearmorpayload.EventType)+".", j)
	if err != nil {
		c.setStanErrorMetrics()
		log.Printf("[ERROR] : STAN - %v\n", err)
		return
	}

	// Setting the success status
	go c.CountMetric(Outputs, 1, []string{"output:stan", "status:ok"})
	c.Stats.Stan.Add(OK, 1)
	c.PromStats.Outputs.With(map[string]string{"destination": "stan", "status": OK}).Inc()
	log.Printf("[INFO]  : STAN - Publish OK\n")
}

// setStanErrorMetrics set the error stats
func (c *Client) setStanErrorMetrics() {
	go c.CountMetric(Outputs, 1, []string{"output:stan", "status:error"})
	c.Stats.Stan.Add(Error, 1)
	c.PromStats.Outputs.With(map[string]string{"destination": "stan", "status": Error}).Inc()
}
