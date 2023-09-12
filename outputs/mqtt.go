package outputs

import (
	"crypto/tls"
	"fmt"
	"log"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"

	"github.com/kubearmor/sidekick/types"
)

// NewMQTTClient returns a new output.Client for accessing Kubernetes.
func NewMQTTClient(config *types.Configuration, stats *types.Statistics, promStats *types.PromStatistics,
	statsdClient, dogstatsdClient *statsd.Client) (*Client, error) {

	options := mqtt.NewClientOptions()
	options.AddBroker(config.MQTT.Broker)
	options.SetClientID("kubearmor-" + uuid.NewString()[:6])
	if config.MQTT.User != "" && config.MQTT.Password != "" {
		options.Username = config.MQTT.User
		options.Password = config.MQTT.Password
	}
	if !config.MQTT.CheckCert {
		// #nosec G402 This is only set as a result of explicit configuration
		options.TLSConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}
	options.OnConnectionLost = func(client mqtt.Client, err error) {
		log.Printf("[ERROR] : MQTT - Connection lost: %v\n", err.Error())
	}

	client := mqtt.NewClient(options)

	return &Client{
		OutputType:      MQTT,
		Config:          config,
		MQTTClient:      client,
		Stats:           stats,
		PromStats:       promStats,
		StatsdClient:    statsdClient,
		DogstatsdClient: dogstatsdClient,
	}, nil
}

// MQTTPublish .
func (c *Client) MQTTPublish(kubearmorpayload types.KubearmorPayload) {
	c.Stats.MQTT.Add(Total, 1)

	t := c.MQTTClient.Connect()
	t.Wait()
	if err := t.Error(); err != nil {
		go c.CountMetric(Outputs, 1, []string{"output:mqtt", "status:error"})
		c.Stats.MQTT.Add(Error, 1)
		c.PromStats.Outputs.With(map[string]string{"destination": "mqtt", "status": err.Error()}).Inc()
		log.Printf("[ERROR] : %s - %v\n", MQTT, err.Error())
		return
	}
	defer c.MQTTClient.Disconnect(100)
	if err := c.MQTTClient.Publish(c.Config.MQTT.Topic, byte(c.Config.MQTT.QOS), c.Config.MQTT.Retained, kubearmorpayload.String()).Error(); err != nil {
		go c.CountMetric(Outputs, 1, []string{"output:mqtt", "status:error"})
		c.Stats.MQTT.Add(Error, 1)
		c.PromStats.Outputs.With(map[string]string{"destination": "mqtt", "status": Error}).Inc()
		log.Printf("[ERROR] : %s - %v\n", MQTT, err.Error())
		return
	}

	log.Printf("[INFO]  : %s - Message published\n", MQTT)
	go c.CountMetric(Outputs, 1, []string{"output:mqtt", "status:ok"})
	c.Stats.MQTT.Add(OK, 1)
	c.PromStats.Outputs.With(map[string]string{"destination": "mqtt", "status": OK}).Inc()
}

func (c *Client) WatchMQTTPublishAlerts() error {
	uid := uuid.Must(uuid.NewRandom()).String()

	conn := make(chan types.KubearmorPayload, 1000)
	defer close(conn)
	addAlertStruct(uid, conn)
	defer removeAlertStruct(uid)

	fmt.Println("discord running")
	for AlertRunning {
		select {
		case resp := <-conn:
			fmt.Println("response \n", resp)
			c.MQTTPublish(resp)
		}
	}
	fmt.Println("discord stopped")
	return nil
}

func (c *Client) WatchMQTTPublishLogs() error {
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
			c.MQTTPublish(resp)

		default:
			time.Sleep(time.Millisecond * 10)
		}
	}

	return nil
}
