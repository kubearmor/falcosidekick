package outputs

import (
	"context"
	"log"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"

	"github.com/falcosecurity/falcosidekick/types"
)

// CloudEventsSend produces a CloudEvent and sends to the CloudEvents consumers.
func (c *Client) CloudEventsSend(kubearmorpayload types.KubearmorPayload) {
	c.Stats.CloudEvents.Add(Total, 1)

	if c.CloudEventsClient == nil {
		client, err := cloudevents.NewClientHTTP()
		if err != nil {
			go c.CountMetric(Outputs, 1, []string{"output:cloudevents", "status:error"})
			log.Printf("[ERROR] : CloudEvents - NewDefaultClient : %v\n", err)
			return
		}
		c.CloudEventsClient = client
	}

	ctx := cloudevents.ContextWithTarget(context.Background(), c.EndpointURL.String())

	event := cloudevents.NewEvent()
	event.SetTime(time.Unix(kubearmorpayload.Timestamp, 0))
	event.SetSource("falco.org") // TODO: this should have some info on the falco server that made the event.
	event.SetType("falco.rule.output.v1")
	event.SetExtension("priority", kubearmorpayload.EventType)
	if kubearmorpayload.Hostname != "" {
		event.SetExtension(Hostname, kubearmorpayload.Hostname)
	}

	// Set Extensions.
	for k, v := range c.Config.CloudEvents.Extensions {
		event.SetExtension(k, v)
	}

	if err := event.SetData(cloudevents.ApplicationJSON, kubearmorpayload); err != nil {
		log.Printf("[ERROR] : CloudEvents, failed to set data : %v\n", err)
	}

	if result := c.CloudEventsClient.Send(ctx, event); cloudevents.IsUndelivered(result) {
		go c.CountMetric(Outputs, 1, []string{"output:cloudevents", "status:error"})
		c.Stats.CloudEvents.Add(Error, 1)
		c.PromStats.Outputs.With(map[string]string{"destination": "cloudevents", "status": Error}).Inc()
		log.Printf("[ERROR] : CloudEvents - %v\n", result)
		return
	}

	// Setting the success status
	go c.CountMetric(Outputs, 1, []string{"output:cloudevents", "status:ok"})
	c.Stats.CloudEvents.Add(OK, 1)
	c.PromStats.Outputs.With(map[string]string{"destination": "cloudevents", "status": OK}).Inc()
	log.Printf("[INFO]  : CloudEvents - Send OK\n")
}

func (c *Client) WatchCloudEventsSendAlerts() error {
	uid := uuid.Must(uuid.NewRandom()).String()

	conn := make(chan types.KubearmorPayload, 1000)
	defer close(conn)
	addAlertStruct(uid, conn)
	defer removeAlertStruct(uid)

	for AlertRunning {
		select {
		// case <-Context().Done():
		// 	return nil
		case resp := <-conn:
			c.CloudEventsSend(resp)
		default:
			time.Sleep(time.Millisecond * 10)

		}
	}

	return nil
}

func (c *Client) WatchCloudEventsSendLogs() error {
	uid := uuid.Must(uuid.NewRandom()).String()

	conn := make(chan types.KubearmorPayload, 1000)
	defer close(conn)
	addLogStruct(uid, conn)
	defer removeLogStruct(uid)

	for LogRunning {
		select {
		// case <-Context().Done():
		// 	return nil
		case resp := <-conn:
			c.CloudEventsSend(resp)

		default:
			time.Sleep(time.Millisecond * 10)
		}
	}

	return nil
}
