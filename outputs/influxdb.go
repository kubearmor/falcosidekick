package outputs

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kubearmor/sidekick/types"
)

type influxdbPayload string

func newInfluxdbPayload(kubearmorpayload types.KubearmorPayload, config *types.Configuration) influxdbPayload {
	s := "events,rule=" + strings.Replace(kubearmorpayload.EventType, " ", "_", -1) + ",priority=" + kubearmorpayload.EventType + ",source=" + kubearmorpayload.OutputFields["PodName"].(string)

	for i, j := range kubearmorpayload.OutputFields {
		switch v := j.(type) {
		case string:
			s += "," + i + "=" + strings.Replace(v, " ", "_", -1)
		default:
			vv := fmt.Sprint(v)
			s += "," + i + "=" + strings.Replace(vv, " ", "_", -1)
			continue
		}
	}

	if kubearmorpayload.Hostname != "" {
		s += "," + Hostname + "=" + kubearmorpayload.Hostname
	}

	return influxdbPayload(s)
}

// InfluxdbPost posts event to InfluxDB
func (c *Client) InfluxdbPost(kubearmorpayload types.KubearmorPayload) {
	c.Stats.Influxdb.Add(Total, 1)

	c.httpClientLock.Lock()
	defer c.httpClientLock.Unlock()
	c.AddHeader("Accept", "application/json")

	if c.Config.Influxdb.Token != "" {
		c.AddHeader("Authorization", "Token "+c.Config.Influxdb.Token)
	}

	err := c.Post(newInfluxdbPayload(kubearmorpayload, c.Config))
	if err != nil {
		go c.CountMetric(Outputs, 1, []string{"output:influxdb", "status:error"})
		c.Stats.Influxdb.Add(Error, 1)
		c.PromStats.Outputs.With(map[string]string{"destination": "influxdb", "status": Error}).Inc()
		log.Printf("[ERROR] : InfluxDB - %v\n", err)
		return
	}

	// Setting the success status
	go c.CountMetric(Outputs, 1, []string{"output:influxdb", "status:ok"})
	c.Stats.Influxdb.Add(OK, 1)
	c.PromStats.Outputs.With(map[string]string{"destination": "influxdb", "status": OK}).Inc()
}

func (c *Client) WatchInfluxdbPostAlerts() error {
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
			c.InfluxdbPost(resp)
		}
	}
	fmt.Println("discord stopped")
	return nil
}

func (c *Client) WatchInfluxdbPostLogs() error {
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
			c.InfluxdbPost(resp)

		default:
			time.Sleep(time.Millisecond * 10)
		}
	}

	return nil
}
