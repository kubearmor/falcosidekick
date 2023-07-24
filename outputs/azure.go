package outputs

import (
	"context"
	"encoding/json"
	"log"
	"time"

	eventhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/DataDog/datadog-go/statsd"
	"github.com/google/uuid"

	"github.com/falcosecurity/falcosidekick/types"
)

// NewEventHubClient returns a new output.Client for accessing the Azure Event Hub.
func NewEventHubClient(config *types.Configuration, stats *types.Statistics, promStats *types.PromStatistics, statsdClient, dogstatsdClient *statsd.Client) (*Client, error) {
	return &Client{
		OutputType:      "AzureEventHub",
		Config:          config,
		Stats:           stats,
		PromStats:       promStats,
		StatsdClient:    statsdClient,
		DogstatsdClient: dogstatsdClient,
	}, nil
}

// EventHubPost posts event to Azure Event Hub
func (c *Client) EventHubPost(kubearmorpayload types.KubearmorPayload) {
	c.Stats.AzureEventHub.Add(Total, 1)

	log.Printf("[INFO] : %v EventHub - Try sending event", c.OutputType)
	hub, err := eventhub.NewHubWithNamespaceNameAndEnvironment(c.Config.Azure.EventHub.Namespace, c.Config.Azure.EventHub.Name)
	if err != nil {
		c.setEventHubErrorMetrics()
		log.Printf("[ERROR] : %v EventHub - %v\n", c.OutputType, err.Error())
		return
	}

	log.Printf("[INFO]  : %v EventHub - Hub client created\n", c.OutputType)

	data, err := json.Marshal(kubearmorpayload)
	if err != nil {
		c.setEventHubErrorMetrics()
		log.Printf("[ERROR] : Cannot marshal payload: %v", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	err = hub.Send(ctx, eventhub.NewEvent(data))
	if err != nil {
		c.setEventHubErrorMetrics()
		log.Printf("[ERROR] : %v EventHub - %v\n", c.OutputType, err.Error())
		return
	}

	// Setting the success status
	go c.CountMetric(Outputs, 1, []string{"output:azureeventhub", "status:ok"})
	c.Stats.AzureEventHub.Add(OK, 1)
	c.PromStats.Outputs.With(map[string]string{"destination": "azureeventhub", "status": OK}).Inc()
	log.Printf("[INFO]  : %v EventHub - Publish OK", c.OutputType)
}

// setEventHubErrorMetrics set the error stats
func (c *Client) setEventHubErrorMetrics() {
	go c.CountMetric(Outputs, 1, []string{"output:azureeventhub", "status:error"})
	c.Stats.AzureEventHub.Add(Error, 1)
	c.PromStats.Outputs.With(map[string]string{"destination": "azureeventhub", "status": Error}).Inc()
}

// EnqueueSecurityLake
func (c *Client) WatchEventHubPostlerts() error {
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
			c.EventHubPost(resp)
		default:
			time.Sleep(time.Millisecond * 10)

		}
	}

	return nil
}

func (c *Client) WatchEventHubPostLogs() error {
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
			c.EventHubPost(resp)

		default:
			time.Sleep(time.Millisecond * 10)
		}
	}

	return nil
}
