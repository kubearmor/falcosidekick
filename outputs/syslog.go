package outputs

import (
	"encoding/json"
	"fmt"
	"log"
	"log/syslog"
	"strings"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/falcosecurity/falcosidekick/types"
)

func NewSyslogClient(config *types.Configuration, stats *types.Statistics, promStats *types.PromStatistics, statsdClient, dogstatsdClient *statsd.Client) (*Client, error) {
	ok := isValidProtocolString(strings.ToLower(config.Syslog.Protocol))
	if !ok {
		return nil, fmt.Errorf("failed to configure Syslog client: invalid protocol %s", config.Syslog.Protocol)
	}

	return &Client{
		OutputType:      "Syslog",
		Config:          config,
		Stats:           stats,
		PromStats:       promStats,
		StatsdClient:    statsdClient,
		DogstatsdClient: dogstatsdClient,
	}, nil
}

func isValidProtocolString(protocol string) bool {
	return protocol == TCP || protocol == UDP
}

func getCEFSeverity(priority string) string {
	switch priority {
	case "Log":
		return "3"
	case "Alert":
		return "9"
	default:
		return "Uknown"
	}
}

func (c *Client) SyslogPost(kubearmorpayload types.KubearmorPayload) {
	//c.Stats.Syslog.Add(Total, 1)
	endpoint := fmt.Sprintf("%s:%s", c.Config.Syslog.Host, c.Config.Syslog.Port)
	fmt.Println("endpoint ", endpoint)
	var priority syslog.Priority
	switch kubearmorpayload.EventType {
	case "Alert":
		priority = syslog.LOG_ALERT
	case "Log":
		priority = syslog.LOG_INFO
	}

	sysLog, err := syslog.Dial(c.Config.Syslog.Protocol, endpoint, priority, Kubearmor)
	if err != nil {
		go c.CountMetric(Outputs, 1, []string{"output:syslog", "status:error"})
		c.Stats.Syslog.Add(Error, 1)
		c.PromStats.Outputs.With(map[string]string{"destination": "syslog", "status": Error}).Inc()
		log.Printf("[ERROR] : Syslog - %v\n", err)
		return
	}
	fmt.Println("syslog - ", sysLog)

	var payload []byte
	timestamp := time.Unix(kubearmorpayload.Timestamp, 0)

	if c.Config.Syslog.Format == "cef" {
		s := fmt.Sprintf(
			"CEF:0|Accuknox|Kubearmor|1.0|Kubearmor Event|%v|uid=%v start=%v",
			kubearmorpayload.EventType,
			fmt.Sprint(kubearmorpayload.OutputFields["UID"]),
			timestamp.Format(time.RFC3339),
		)
		s += " " + kubearmorpayload.EventType + "="
		for i, j := range kubearmorpayload.OutputFields {
			switch v := j.(type) {
			case string:
				if v == "" {
					continue
				}
				s += fmt.Sprintf("%v:%v ", i, v)
			default:
				vv := fmt.Sprint(v)
				s += fmt.Sprintf("%v:%v ", i, vv)
			}
		}
		fmt.Println("payload ", s)
		payload = []byte(strings.TrimSuffix(s, " "))
	} else {
		payload, _ = json.Marshal(kubearmorpayload)
	}

	_, err = sysLog.Write(payload)
	if err != nil {
		// go c.CountMetric(Outputs, 1, []string{"output:syslog", "status:error"})
		// c.Stats.Syslog.Add(Error, 1)
		// c.PromStats.Outputs.With(map[string]string{"destination": "syslog", "status": Error}).Inc()
		log.Printf("[ERROR] : Syslog - %v\n", err)
		return
	}

	go c.CountMetric(Outputs, 1, []string{"output:syslog", "status:ok"})
	c.Stats.Syslog.Add(OK, 1)
	c.PromStats.Outputs.With(map[string]string{"destination": "syslog", "status": OK}).Inc()
}

func (c *Client) WatchSyslogsAlerts() error {
	uid := "syslog"

	conn := make(chan types.KubearmorPayload, 1000)
	defer close(conn)
	addAlertStruct(uid, conn)
	defer removeAlertStruct(uid)

	for AlertRunning {
		select {
		// case <-Context().Done():
		// 	return nil
		case resp := <-conn:
			fmt.Println("got it ", resp)
			c.SyslogPost(resp)
		default:
			time.Sleep(time.Millisecond * 10)

		}
	}

	return nil
}

func (c *Client) WatchSyslogLogs() error {
	uid := "syslog"

	conn := make(chan types.KubearmorPayload, 1000)
	defer close(conn)
	addLogStruct(uid, conn)
	defer removeLogStruct(uid)

	for LogRunning {
		select {
		// case <-Context().Done():
		// 	return nil
		case resp := <-conn:
			c.SyslogPost(resp)

		default:
			time.Sleep(time.Millisecond * 10)
		}
	}

	return nil
}
