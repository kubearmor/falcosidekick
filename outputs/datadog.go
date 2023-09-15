package outputs

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/kubearmor/sidekick/types"
)

const (
	// DatadogPath is the path of Datadog's event API
	DatadogPath string = "/api/v1/events"
)

type datadogPayload struct {
	Title      string   `json:"title,omitempty"`
	Text       string   `json:"text,omitempty"`
	AlertType  string   `json:"alert_type,omitempty"`
	SourceType string   `json:"source_type_name,omitempty"`
	Tags       []string `json:"tags,omitempty"`
}

func newDatadogPayload(kubearmorpayload types.KubearmorPayload) datadogPayload {
	var d datadogPayload
	var tags []string

	for i, j := range kubearmorpayload.OutputFields {
		switch v := j.(type) {
		case string:
			tags = append(tags, i+":"+v)
		default:
			vv := fmt.Sprintln(v)
			tags = append(tags, i+":"+vv)
			continue
		}
	}

	d.Tags = tags

	d.SourceType = "kubearmor"

	var status string
	switch kubearmorpayload.EventType {
	case "Alert":
		status = Error
	default:
		status = Info
	}
	d.AlertType = status

	return d
}

// DatadogPost posts event to Datadog
func (c *Client) DatadogPost(kubearmorpayload types.KubearmorPayload) {
	c.Stats.Datadog.Add(Total, 1)

	err := c.Post(newDatadogPayload(kubearmorpayload))
	if err != nil {
		go c.CountMetric(Outputs, 1, []string{"output:datadog", "status:error"})
		c.Stats.Datadog.Add(Error, 1)
		c.PromStats.Outputs.With(map[string]string{"destination": "datadog", "status": Error}).Inc()
		log.Printf("[ERROR] : Datadog - %v\n", err)
		return
	}

	go c.CountMetric(Outputs, 1, []string{"output:datadog", "status:ok"})
	c.Stats.Datadog.Add(OK, 1)
	c.PromStats.Outputs.With(map[string]string{"destination": "datadog", "status": OK}).Inc()
}

func (c *Client) WatchDatadogPostAlerts() error {
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
			c.DatadogPost(resp)
		default:
			time.Sleep(time.Millisecond * 10)

		}
	}

	return nil
}

func (c *Client) WatchDatadogPostLogs() error {
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
			c.DatadogPost(resp)

		default:
			time.Sleep(time.Millisecond * 10)
		}
	}

	return nil
}
