package outputs

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/kubearmor/sidekick/types"
)

type grafanaPayload struct {
	DashboardID int      `json:"dashboardId,omitempty"`
	PanelID     int      `json:"panelId,omitempty"`
	Time        int64    `json:"time"`
	TimeEnd     int64    `json:"timeEnd"`
	Tags        []string `json:"tags"`
	Text        string   `json:"text"`
}

type grafanaOnCallPayload struct {
	AlertUID string `json:"alert_uid"`
	State    string `json:"state"`
	Title    string `json:"title"`
	Message  string `json:"message"`
}

// The Content-Type to send along with the request
const GrafanaContentType = "application/json"

func newGrafanaPayload(kubearmorpayload types.KubearmorPayload, config *types.Configuration) grafanaPayload {
	tags := []string{
		"kubearmor",
		kubearmorpayload.EventType,
	}
	if kubearmorpayload.Hostname != "" {
		tags = append(tags, kubearmorpayload.Hostname)
	}

	if config.Grafana.AllFieldsAsTags {
		for key, i := range kubearmorpayload.OutputFields {
			s := key + ": " + fmt.Sprint(i)
			tags = append(tags, s)
		}
	}

	g := grafanaPayload{
		Text:    kubearmorpayload.EventType + "for pod" + kubearmorpayload.OutputFields["PodName"].(string),
		Time:    kubearmorpayload.Timestamp / 1000000,
		TimeEnd: kubearmorpayload.Timestamp / 1000000,
		Tags:    tags,
	}

	if config.Grafana.DashboardID != 0 {
		g.DashboardID = config.Grafana.DashboardID
	}
	if config.Grafana.PanelID != 0 {
		g.PanelID = config.Grafana.PanelID
	}

	return g
}

func newGrafanaOnCallPayload(kubearmorpayload types.KubearmorPayload, config *types.Configuration) grafanaOnCallPayload {
	return grafanaOnCallPayload{
		AlertUID: kubearmorpayload.OutputFields["UID"].(string),
		Title:    fmt.Sprintf("[%v] %v", kubearmorpayload.EventType, kubearmorpayload.OutputFields["PodName"].(string)),
		State:    "alerting",
		//Message:  kubearmorpayload.Output,
	}
}

// GrafanaPost posts event to grafana
func (c *Client) GrafanaPost(kubearmorpayload types.KubearmorPayload) {
	c.Stats.Grafana.Add(Total, 1)
	c.ContentType = GrafanaContentType
	c.httpClientLock.Lock()
	defer c.httpClientLock.Unlock()
	c.AddHeader("Authorization", "Bearer "+c.Config.Grafana.APIKey)
	for i, j := range c.Config.Grafana.CustomHeaders {
		c.AddHeader(i, j)
	}

	err := c.Post(newGrafanaPayload(kubearmorpayload, c.Config))
	if err != nil {
		go c.CountMetric(Outputs, 1, []string{"output:grafana", "status:error"})
		c.Stats.Grafana.Add(Error, 1)
		c.PromStats.Outputs.With(map[string]string{"destination": "grafana", "status": Error}).Inc()
		log.Printf("[ERROR] : Grafana - %v\n", err)
		return
	}

	go c.CountMetric(Outputs, 1, []string{"output:grafana", "status:ok"})
	c.Stats.Grafana.Add(OK, 1)
	c.PromStats.Outputs.With(map[string]string{"destination": "grafana", "status": OK}).Inc()
}

// GrafanaOnCallPost posts event to grafana onCall
func (c *Client) GrafanaOnCallPost(kubearmorpayload types.KubearmorPayload) {
	c.Stats.GrafanaOnCall.Add(Total, 1)
	c.ContentType = GrafanaContentType
	c.httpClientLock.Lock()
	defer c.httpClientLock.Unlock()
	for i, j := range c.Config.GrafanaOnCall.CustomHeaders {
		c.AddHeader(i, j)
	}

	err := c.Post(newGrafanaOnCallPayload(kubearmorpayload, c.Config))
	if err != nil {
		go c.CountMetric(Outputs, 1, []string{"output:grafanaoncall", "status:error"})
		c.Stats.Grafana.Add(Error, 1)
		c.PromStats.Outputs.With(map[string]string{"destination": "grafanaoncall", "status": Error}).Inc()
		log.Printf("[ERROR] : Grafana OnCall - %v\n", err)
		return
	}

	go c.CountMetric(Outputs, 1, []string{"output:grafanaoncall", "status:ok"})
	c.Stats.Grafana.Add(OK, 1)
	c.PromStats.Outputs.With(map[string]string{"destination": "grafanaoncall", "status": OK}).Inc()
}

func (c *Client) WatchGrafanaPostAlerts() error {
	uid := uuid.Must(uuid.NewRandom()).String()

	conn := make(chan types.KubearmorPayload, 1000)
	defer close(conn)
	addAlertStruct(uid, conn)
	defer removeAlertStruct(uid)

	Running := true
	fmt.Println("discord running")
	for Running {
		select {
		case resp := <-conn:
			fmt.Println("response \n", resp)
			c.GrafanaPost(resp)
		}
	}
	fmt.Println("discord stopped")
	return nil
}

func (c *Client) WatchGrafanaPostLogs() error {
	uid := uuid.Must(uuid.NewRandom()).String()

	conn := make(chan types.KubearmorPayload, 1000)
	defer close(conn)
	addLogStruct(uid, conn)
	defer removeLogStruct(uid)

	Running := true
	for Running {
		select {
		// case <-Context().Done():
		// 	return nil
		case resp := <-conn:
			c.GrafanaPost(resp)

		default:
			time.Sleep(time.Millisecond * 10)
		}
	}

	return nil
}

func (c *Client) WatchGrafanaOnCallPostAlerts() error {
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
			c.GrafanaOnCallPost(resp)
		}
	}
	fmt.Println("discord stopped")
	return nil
}

func (c *Client) WatchGrafanaOnCallPostLogs() error {
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
			c.GrafanaOnCallPost(resp)

		default:
			time.Sleep(time.Millisecond * 10)
		}
	}

	return nil
}
