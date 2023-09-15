package outputs

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/kubearmor/sidekick/types"
)

type teamsFact struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type teamsSection struct {
	ActivityTitle    string      `json:"activityTitle"`
	ActivitySubTitle string      `json:"activitySubtitle"`
	ActivityImage    string      `json:"activityImage,omitempty"`
	Text             string      `json:"text"`
	Facts            []teamsFact `json:"facts,omitempty"`
}

// Payload
type teamsPayload struct {
	Type       string         `json:"@type"`
	Summary    string         `json:"summary,omitempty"`
	ThemeColor string         `json:"themeColor,omitempty"`
	Sections   []teamsSection `json:"sections"`
}

func newTeamsPayload(kubearmorpayload types.KubearmorPayload, config *types.Configuration) teamsPayload {
	var (
		sections []teamsSection
		section  teamsSection
		facts    []teamsFact
		fact     teamsFact
	)

	section.ActivityTitle = "Kubearmor Sidekick"
	section.ActivitySubTitle = fmt.Sprint(kubearmorpayload.Timestamp)

	if config.Teams.ActivityImage != "" {
		section.ActivityImage = config.Teams.ActivityImage
	}

	if config.Teams.OutputFormat == All || config.Teams.OutputFormat == "facts" || config.Teams.OutputFormat == "" {
		for i, j := range kubearmorpayload.OutputFields {
			switch v := j.(type) {
			case string:
				fact.Name = i
				fact.Value = v
			default:
				vv := fmt.Sprint(v)
				fact.Name = i
				fact.Value = vv

			}

			facts = append(facts, fact)
		}

		fact.Name = Priority
		fact.Value = kubearmorpayload.EventType
		facts = append(facts, fact)
		fact.Name = Source
		fact.Value = kubearmorpayload.OutputFields["PodName"].(string)
		facts = append(facts, fact)
		if kubearmorpayload.Hostname != "" {
			fact.Name = Hostname
			fact.Value = kubearmorpayload.Hostname
			facts = append(facts, fact)
		}
	}

	section.Facts = facts

	var color string
	switch kubearmorpayload.EventType {
	case "Alert":
		color = "ff5400"
	case "Log":
		color = "68c2ff"
	}

	sections = append(sections, section)

	t := teamsPayload{
		Type:       "MessageCard",
		ThemeColor: color,
		Sections:   sections,
	}

	return t
}

// TeamsPost posts event to Teams
func (c *Client) TeamsPost(kubearmorpayload types.KubearmorPayload) {
	c.Stats.Teams.Add(Total, 1)

	err := c.Post(newTeamsPayload(kubearmorpayload, c.Config))
	if err != nil {
		go c.CountMetric(Outputs, 1, []string{"output:teams", "status:error"})
		c.Stats.Teams.Add(Error, 1)
		c.PromStats.Outputs.With(map[string]string{"destination": "teams", "status": Error}).Inc()
		log.Printf("[ERROR] : Teams - %v\n", err)
		return
	}

	// Setting the success status
	go c.CountMetric(Outputs, 1, []string{"output:teams", "status:ok"})
	c.Stats.Teams.Add(OK, 1)
	c.PromStats.Outputs.With(map[string]string{"destination": "teams", "status": OK}).Inc()
}

func (c *Client) WatchTeamsPostAlerts() error {
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
			c.TeamsPost(resp)
		default:
			time.Sleep(time.Millisecond * 10)

		}
	}

	return nil
}

func (c *Client) WatchTeamsPostLogs() error {
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
			c.TeamsPost(resp)

		default:
			time.Sleep(time.Millisecond * 10)
		}
	}

	return nil
}
