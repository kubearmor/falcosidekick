package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/falcosecurity/falcosidekick/types"
	"github.com/google/uuid"
)

const testRule string = "Test rule"

// mainHandler is Falco Sidekick main handler (default).
func mainHandler(w http.ResponseWriter, r *http.Request) {

	stats.Requests.Add("total", 1)
	nullClient.CountMetric("total", 1, []string{})

	if r.Body == nil {
		http.Error(w, "Please send a valid request body", http.StatusBadRequest)
		stats.Requests.Add("rejected", 1)
		promStats.Inputs.With(map[string]string{"source": "requests", "status": "rejected"}).Inc()
		nullClient.CountMetric("inputs.requests.rejected", 1, []string{"error:nobody"})

		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Please send with post http method", http.StatusBadRequest)
		stats.Requests.Add("rejected", 1)
		promStats.Inputs.With(map[string]string{"source": "requests", "status": "rejected"}).Inc()
		nullClient.CountMetric("inputs.requests.rejected", 1, []string{"error:nobody"})

		return
	}

	KubearmorPayload, err := newKubearmorPayload(r.Body)
	if err != nil || !KubearmorPayload.Check() {
		http.Error(w, "Please send a valid request body", http.StatusBadRequest)
		stats.Requests.Add("rejected", 1)
		promStats.Inputs.With(map[string]string{"source": "requests", "status": "rejected"}).Inc()
		nullClient.CountMetric("inputs.requests.rejected", 1, []string{"error:invalidjson"})

		return
	}

	nullClient.CountMetric("inputs.requests.accepted", 1, []string{})
	stats.Requests.Add("accepted", 1)
	promStats.Inputs.With(map[string]string{"source": "requests", "status": "accepted"}).Inc()
	forwardEvent(KubearmorPayload)
}

// pingHandler is a simple handler to test if daemon is UP.
func pingHandler(w http.ResponseWriter, r *http.Request) {
	// #nosec G104 nothing to be done if the following fails
	w.Write([]byte("pong\n"))
}

// healthHandler is a simple handler to test if daemon is UP.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	// #nosec G104 nothing to be done if the following fails
	w.Write([]byte(`{"status": "ok"}`))
}

// testHandler sends a test event to all enabled outputs.
func testHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = io.NopCloser(bytes.NewReader([]byte(`{"output":"This is a test from falcosidekick","priority":"Debug","hostname": "falcosidekick", "rule":"Test rule", "time":"` + time.Now().UTC().Format(time.RFC3339) + `","output_fields": {"proc.name":"falcosidekick","user.name":"falcosidekick"}, "tags":["test","example"]}`)))
	mainHandler(w, r)
}

func newKubearmorPayload(payload io.Reader) (types.KubearmorPayload, error) {
	var KubearmorPayload types.KubearmorPayload

	d := json.NewDecoder(payload)
	d.UseNumber()

	err := d.Decode(&KubearmorPayload)
	if err != nil {
		return types.KubearmorPayload{}, err
	}

	if len(config.Customfields) > 0 {
		if KubearmorPayload.OutputFields == nil {
			KubearmorPayload.OutputFields = make(map[string]interface{})
		}
		for key, value := range config.Customfields {
			KubearmorPayload.OutputFields[key] = value
		}
	}

	if KubearmorPayload.Source == "" {
		KubearmorPayload.Source = "syscalls"
	}

	KubearmorPayload.UUID = uuid.New().String()

	var kn, kp string
	for i, j := range KubearmorPayload.OutputFields {
		if j != nil {
			if i == "k8s.ns.name" {
				kn = j.(string)
			}
			if i == "k8s.pod.name" {
				kp = j.(string)
			}
		}
	}

	if len(config.Templatedfields) > 0 {
		if KubearmorPayload.OutputFields == nil {
			KubearmorPayload.OutputFields = make(map[string]interface{})
		}
		for key, value := range config.Templatedfields {
			tmpl, err := template.New("").Parse(value)
			if err != nil {
				log.Printf("[ERROR] : Parsing error for templated field '%v': %v\n", key, err)
				continue
			}
			v := new(bytes.Buffer)
			if err := tmpl.Execute(v, KubearmorPayload.OutputFields); err != nil {
				log.Printf("[ERROR] : Parsing error for templated field '%v': %v\n", key, err)
			}
			KubearmorPayload.OutputFields[key] = v.String()
		}
	}

	nullClient.CountMetric("falco.accepted", 1, []string{"priority:" + KubearmorPayload.Priority.String()})
	stats.Falco.Add(strings.ToLower(KubearmorPayload.Priority.String()), 1)
	promLabels := map[string]string{"rule": KubearmorPayload.Rule, "priority": KubearmorPayload.Priority.String(), "k8s_ns_name": kn, "k8s_pod_name": kp}
	if KubearmorPayload.Hostname != "" {
		promLabels["hostname"] = KubearmorPayload.Hostname
	}

	for key, value := range config.Customfields {
		if regPromLabels.MatchString(key) {
			promLabels[key] = value
		}
	}
	for _, i := range config.Prometheus.ExtraLabelsList {
		promLabels[strings.ReplaceAll(i, ".", "_")] = ""
		for key, value := range KubearmorPayload.OutputFields {
			if key == i && regPromLabels.MatchString(strings.ReplaceAll(key, ".", "_")) {
				switch value.(type) {
				case string:
					promLabels[strings.ReplaceAll(key, ".", "_")] = fmt.Sprintf("%v", value)
				default:
					continue
				}
			}
		}
	}
	promStats.Falco.With(promLabels).Inc()

	if config.BracketReplacer != "" {
		for i, j := range KubearmorPayload.OutputFields {
			if strings.Contains(i, "[") {
				KubearmorPayload.OutputFields[strings.ReplaceAll(strings.ReplaceAll(i, "]", ""), "[", config.BracketReplacer)] = j
				delete(KubearmorPayload.OutputFields, i)
			}
		}
	}

	if config.Debug {
		body, _ := json.Marshal(KubearmorPayload)
		log.Printf("[DEBUG] : Falco's payload : %v\n", string(body))
	}

	return KubearmorPayload, nil
}

func forwardEvent(KubearmorPayload types.KubearmorPayload) {
	//done
	if config.Slack.WebhookURL != "" && (KubearmorPayload.Priority >= types.Priority(config.Slack.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go slackClient.WatchSlackAlerts()
	}

	if config.Cliq.WebhookURL != "" && (KubearmorPayload.Priority >= types.Priority(config.Cliq.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go cliqClient.CliqPost(KubearmorPayload)
	}

	if config.Rocketchat.WebhookURL != "" && (KubearmorPayload.Priority >= types.Priority(config.Rocketchat.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go rocketchatClient.RocketchatPost(KubearmorPayload)
	}

	if config.Mattermost.WebhookURL != "" && (KubearmorPayload.Priority >= types.Priority(config.Mattermost.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go mattermostClient.MattermostPost(KubearmorPayload)
	}

	if config.Teams.WebhookURL != "" && (KubearmorPayload.Priority >= types.Priority(config.Teams.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go teamsClient.TeamsPost(KubearmorPayload)
	}

	if config.Datadog.APIKey != "" && (KubearmorPayload.Priority >= types.Priority(config.Datadog.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go datadogClient.DatadogPost(KubearmorPayload)
	}

	if config.Discord.WebhookURL != "" && (KubearmorPayload.Priority >= types.Priority(config.Discord.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go discordClient.DiscordPost(KubearmorPayload)
	}
	//done
	if config.Alertmanager.HostPort != "" && (KubearmorPayload.Priority >= types.Priority(config.Alertmanager.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go alertmanagerClient.WatchAlertmanagerPostAlerts()
	}

	if config.Elasticsearch.HostPort != "" && (KubearmorPayload.Priority >= types.Priority(config.Elasticsearch.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go elasticsearchClient.ElasticsearchPost(KubearmorPayload)
	}

	if config.Influxdb.HostPort != "" && (KubearmorPayload.Priority >= types.Priority(config.Influxdb.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go influxdbClient.InfluxdbPost(KubearmorPayload)
	}

	if config.Loki.HostPort != "" && (KubearmorPayload.Priority >= types.Priority(config.Loki.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go lokiClient.LokiPost(KubearmorPayload)
	}

	if config.Nats.HostPort != "" && (KubearmorPayload.Priority >= types.Priority(config.Nats.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go natsClient.NatsPublish(KubearmorPayload)
	}

	if config.Stan.HostPort != "" && config.Stan.ClusterID != "" && config.Stan.ClientID != "" && (KubearmorPayload.Priority >= types.Priority(config.Stan.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go stanClient.StanPublish(KubearmorPayload)
	}

	if config.AWS.Lambda.FunctionName != "" && (KubearmorPayload.Priority >= types.Priority(config.AWS.Lambda.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go awsClient.InvokeLambda(KubearmorPayload)
	}

	if config.AWS.SQS.URL != "" && (KubearmorPayload.Priority >= types.Priority(config.AWS.SQS.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go awsClient.SendMessage(KubearmorPayload)
	}

	if config.AWS.SNS.TopicArn != "" && (KubearmorPayload.Priority >= types.Priority(config.AWS.SNS.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go awsClient.PublishTopic(KubearmorPayload)
	}

	if config.AWS.CloudWatchLogs.LogGroup != "" && (KubearmorPayload.Priority >= types.Priority(config.AWS.CloudWatchLogs.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go awsClient.SendCloudWatchLog(KubearmorPayload)
	}

	if config.AWS.S3.Bucket != "" && (KubearmorPayload.Priority >= types.Priority(config.AWS.S3.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go awsClient.UploadS3(KubearmorPayload)
	}

	if (config.AWS.SecurityLake.Bucket != "" && config.AWS.SecurityLake.Region != "" && config.AWS.SecurityLake.AccountID != "" && config.AWS.SecurityLake.Prefix != "") && (KubearmorPayload.Priority >= types.Priority(config.AWS.SecurityLake.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go awsClient.EnqueueSecurityLake(KubearmorPayload)
	}

	if config.AWS.Kinesis.StreamName != "" && (KubearmorPayload.Priority >= types.Priority(config.AWS.Kinesis.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go awsClient.PutRecord(KubearmorPayload)
	}

	if config.SMTP.HostPort != "" && (KubearmorPayload.Priority >= types.Priority(config.SMTP.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go smtpClient.SendMail(KubearmorPayload)
	}

	if config.Opsgenie.APIKey != "" && (KubearmorPayload.Priority >= types.Priority(config.Opsgenie.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go opsgenieClient.OpsgeniePost(KubearmorPayload)
	}

	if config.Webhook.Address != "" && (KubearmorPayload.Priority >= types.Priority(config.Webhook.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go webhookClient.WebhookPost(KubearmorPayload)
	}

	if config.NodeRed.Address != "" && (KubearmorPayload.Priority >= types.Priority(config.NodeRed.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go noderedClient.NodeRedPost(KubearmorPayload)
	}

	if config.CloudEvents.Address != "" && (KubearmorPayload.Priority >= types.Priority(config.CloudEvents.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go cloudeventsClient.CloudEventsSend(KubearmorPayload)
	}

	if config.Azure.EventHub.Name != "" && (KubearmorPayload.Priority >= types.Priority(config.Azure.EventHub.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go azureClient.EventHubPost(KubearmorPayload)
	}

	if config.GCP.PubSub.ProjectID != "" && config.GCP.PubSub.Topic != "" && (KubearmorPayload.Priority >= types.Priority(config.GCP.PubSub.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go gcpClient.GCPPublishTopic(KubearmorPayload)
	}

	if config.GCP.CloudFunctions.Name != "" && (KubearmorPayload.Priority >= types.Priority(config.GCP.CloudFunctions.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go gcpClient.GCPCallCloudFunction(KubearmorPayload)
	}

	if config.GCP.CloudRun.Endpoint != "" && (KubearmorPayload.Priority >= types.Priority(config.GCP.CloudRun.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go gcpCloudRunClient.CloudRunFunctionPost(KubearmorPayload)
	}

	if config.GCP.Storage.Bucket != "" && (KubearmorPayload.Priority >= types.Priority(config.GCP.Storage.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go gcpClient.UploadGCS(KubearmorPayload)
	}

	if config.Googlechat.WebhookURL != "" && (KubearmorPayload.Priority >= types.Priority(config.Googlechat.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go googleChatClient.GooglechatPost(KubearmorPayload)
	}

	if config.Kafka.HostPort != "" && (KubearmorPayload.Priority >= types.Priority(config.Kafka.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go kafkaClient.KafkaProduce(KubearmorPayload)
	}

	if config.KafkaRest.Address != "" && (KubearmorPayload.Priority >= types.Priority(config.KafkaRest.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go kafkaRestClient.KafkaRestPost(KubearmorPayload)
	}

	if config.Pagerduty.RoutingKey != "" && (KubearmorPayload.Priority >= types.Priority(config.Pagerduty.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go pagerdutyClient.PagerdutyPost(KubearmorPayload)
	}

	if config.Kubeless.Namespace != "" && config.Kubeless.Function != "" && (KubearmorPayload.Priority >= types.Priority(config.Kubeless.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go kubelessClient.KubelessCall(KubearmorPayload)
	}

	if config.Openfaas.FunctionName != "" && (KubearmorPayload.Priority >= types.Priority(config.Openfaas.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go openfaasClient.OpenfaasCall(KubearmorPayload)
	}

	if config.Tekton.EventListener != "" && (KubearmorPayload.Priority >= types.Priority(config.Tekton.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go tektonClient.TektonPost(KubearmorPayload)
	}
	//done
	if config.Rabbitmq.URL != "" && config.Rabbitmq.Queue != "" && (KubearmorPayload.Priority >= types.Priority(config.Openfaas.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go rabbitmqClient.WatchRabbitmqPublishAlerts()
	}

	if config.Wavefront.EndpointHost != "" && config.Wavefront.EndpointType != "" && (KubearmorPayload.Priority >= types.Priority(config.Wavefront.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go wavefrontClient.WavefrontPost(KubearmorPayload)
	}

	if config.Grafana.HostPort != "" && (KubearmorPayload.Priority >= types.Priority(config.Grafana.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go grafanaClient.GrafanaPost(KubearmorPayload)
	}

	if config.GrafanaOnCall.WebhookURL != "" && (KubearmorPayload.Priority >= types.Priority(config.GrafanaOnCall.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go grafanaOnCallClient.GrafanaOnCallPost(KubearmorPayload)
	}

	if config.WebUI.URL != "" {
		go webUIClient.WebUIPost(KubearmorPayload)
	}

	if config.Fission.Function != "" && (KubearmorPayload.Priority >= types.Priority(config.Fission.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go fissionClient.FissionCall(KubearmorPayload)
	}
	if config.PolicyReport.Enabled && (KubearmorPayload.Priority >= types.Priority(config.PolicyReport.MinimumPriority)) {
		go policyReportClient.UpdateOrCreatePolicyReport(KubearmorPayload)
	}

	if config.Yandex.S3.Bucket != "" && (KubearmorPayload.Priority >= types.Priority(config.Yandex.S3.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go yandexClient.UploadYandexS3(KubearmorPayload)
	}

	if config.Yandex.DataStreams.StreamName != "" && (KubearmorPayload.Priority >= types.Priority(config.Yandex.DataStreams.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go yandexClient.UploadYandexDataStreams(KubearmorPayload)
	}

	if config.Syslog.Host != "" && (KubearmorPayload.Priority >= types.Priority(config.Syslog.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go syslogClient.SyslogPost(KubearmorPayload)
	}

	if config.MQTT.Broker != "" && (KubearmorPayload.Priority >= types.Priority(config.MQTT.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go mqttClient.MQTTPublish(KubearmorPayload)
	}

	if config.Zincsearch.HostPort != "" && (KubearmorPayload.Priority >= types.Priority(config.Zincsearch.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go zincsearchClient.ZincsearchPost(KubearmorPayload)
	}

	if config.Gotify.HostPort != "" && (KubearmorPayload.Priority >= types.Priority(config.Gotify.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go gotifyClient.GotifyPost(KubearmorPayload)
	}

	if config.Spyderbat.OrgUID != "" && (KubearmorPayload.Priority >= types.Priority(config.Spyderbat.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go spyderbatClient.SpyderbatPost(KubearmorPayload)
	}

	if config.TimescaleDB.Host != "" && (KubearmorPayload.Priority >= types.Priority(config.TimescaleDB.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go timescaleDBClient.TimescaleDBPost(KubearmorPayload)
	}

	if config.Redis.Address != "" && (KubearmorPayload.Priority >= types.Priority(config.Redis.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go redisClient.RedisPost(KubearmorPayload)
	}

	if config.Telegram.ChatID != "" && config.Telegram.Token != "" && (KubearmorPayload.Priority >= types.Priority(config.Telegram.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go telegramClient.TelegramPost(KubearmorPayload)
	}

	if config.N8N.Address != "" && (KubearmorPayload.Priority >= types.Priority(config.N8N.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go n8nClient.N8NPost(KubearmorPayload)
	}

	if config.OpenObserve.HostPort != "" && (KubearmorPayload.Priority >= types.Priority(config.OpenObserve.MinimumPriority) || KubearmorPayload.Rule == testRule) {
		go openObserveClient.OpenObservePost(KubearmorPayload)
	}

	if config.Dynatrace.APIToken != "" && config.Dynatrace.APIUrl != "" && (falcopayload.Priority >= types.Priority(config.Dynatrace.MinimumPriority) || falcopayload.Rule == testRule) {
		go dynatraceClient.DynatracePost(falcopayload)
	}
}
