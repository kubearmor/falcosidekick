package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	pb "github.com/kubearmor/KubeArmor/protobuf"
	"github.com/kubearmor/sidekick/outputs"
	"github.com/kubearmor/sidekick/types"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func GetLogsFromKubearmorRelay() {
	lc := outputs.Client{}
	var err error
	//get url
	//connect to url
	//conn := ConnKubeArmorRelay(url, "32767")
	conn, err := ConnectRetry(func() (*grpc.ClientConn, error) {
		url := GetKubearmorRelayURL()
		return ConnKubeArmorRelay(url, "32767")
	}, 6, 10*time.Second)
	if conn == nil || err != nil {
		fmt.Println("Unable to Connect to Relay Server.|", err, "| Shutting down......")
		os.Exit(1)
	}
	lc.Conn = conn
	//create log and alert watcher
	client := pb.NewLogServiceClient(conn)
	alertreq := pb.RequestMessage{}
	alertreq.Filter = "all"

	lc.AlertStream, err = client.WatchAlerts(context.Background(), &alertreq)
	if err != nil {
		log.Error().Msg("unable to stream systems logs: " + err.Error())
		return
	}

	//create a buffer to accept alerts
	lc.WgServer.Add(1)
	go lc.WatchAlerts()
	go lc.AddAlertFromBuffChan()

	logreq := pb.RequestMessage{}
	logreq.Filter = "all"

	lc.LogStream, err = client.WatchLogs(context.Background(), &logreq)
	if err != nil {
		log.Error().Msg("unable to stream systems logs: " + err.Error())
		return
	}
	lc.WgServer.Add(1)
	//create a buffer to accept logs
	go lc.WatchLogs()
	go lc.AddLogFromBuffChan()

	lc.WgServer.Wait()

	if err := lc.DestroyClient(); err != nil {
		fmt.Println("Failed to destroy the grpc client")
	}

}

func GetKubearmorRelayURL() string {
	client := ConnectK8sClient()
	if client == nil {
		log.Error().Msg("error is: Unable to create k8s client")
		return ""
	}
	pods, err := client.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{
		LabelSelector: "kubearmor-app",
	})
	if err != nil {
		log.Error().Msg("error is " + err.Error())
		return ""
	}

	for _, pod := range pods.Items {
		if val, ok := pod.ObjectMeta.Labels["kubearmor-app"]; !ok {
			continue
		} else if val != "kubearmor-relay" {
			continue
		}
		if pod.Status.PodIP != "" {
			log.Info().Msgf("Found RelayServer, %s", pod.Status.PodIP)

			return pod.Status.PodIP
		}

	}

	return ""
}

func ConnectK8sClient() *kubernetes.Clientset {
	config, _ := rest.InClusterConfig()
	clientset, _ := kubernetes.NewForConfig(config)

	return clientset
}

func ConnectRetry(fn func() (*grpc.ClientConn, error), maxRetries int, delay time.Duration) (*grpc.ClientConn, error) {
	var conn *grpc.ClientConn
	var err error

	for i := 0; i < maxRetries; i++ {
		conn, err = fn()
		if err == nil {
			return conn, nil // Success
		}
		log.Info().Msgf("Retry attempt %d failed with error: %s", i+1, err)
		time.Sleep(delay)
	}

	return nil, fmt.Errorf("after %d attempts, last error: %s", maxRetries, err)
}

func ConnKubeArmorRelay(url string, port string) (*grpc.ClientConn, error) {
	addr := net.JoinHostPort(url, port)
	log.Info().Msg(fmt.Sprint("url is ", url))

	// Check for kubearmor-relay with 30s timeout
	ctx, cf1 := context.WithTimeout(context.Background(), time.Second*30)
	defer cf1()
	// Blocking grpc Dial: in case of a bad connection, fails with timeout
	conn, err := grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Error().Msg("Error connecting kubearmor relay: " + err.Error())
		return nil, err
	}

	log.Info().Msg("Connected to kubearmor relay " + addr)
	return conn, nil
}

func createReceiveBuffer() {
	KubearmorPayload := types.KubearmorPayload{}

	if config.PolicyReport.Enabled {
		go policyReportClient.WatchPolicyAlerts()
	}

	//done
	if config.Slack.WebhookURL != "" {
		go slackClient.WatchSlackAlerts()
		go slackClient.WatchSlackLogs()
	}

	if config.Cliq.WebhookURL != "" {
		go cliqClient.WatchCliqPostAlerts()
		go cliqClient.WatchCliqPostLogs()
	}

	if config.Rocketchat.WebhookURL != "" {
		go rocketchatClient.RocketchatPost(KubearmorPayload)
	}

	if config.Mattermost.WebhookURL != "" {
		go mattermostClient.MattermostPost(KubearmorPayload)
	}

	if config.Teams.WebhookURL != "" {
		go teamsClient.TeamsPost(KubearmorPayload)
	}

	if config.Datadog.APIKey != "" {
		go datadogClient.WatchDatadogPostLogs()
		go datadogClient.WatchDatadogPostAlerts()
	}

	if config.Discord.WebhookURL != "" {
		go discordClient.WatchDiscordAlerts()
		go discordClient.WatchDiscordLogs()
	}

	//done
	if config.Alertmanager.HostPort != "" {
		go alertmanagerClient.WatchAlertmanagerPostAlerts()
		go alertmanagerClient.WatchLogmanagerPostAlerts()
	}

	if config.Elasticsearch.HostPort != "" {
		go elasticsearchClient.WatchElasticsearchPostLogs()
		go elasticsearchClient.WatchElasticsearchPostAlerts()
	}

	if config.Influxdb.HostPort != "" {
		go influxdbClient.WatchInfluxdbPostAlerts()
		go influxdbClient.WatchInfluxdbPostLogs()
	}

	if config.Loki.HostPort != "" {
		go lokiClient.LokiPost(KubearmorPayload)
	}

	if config.Nats.HostPort != "" {
		go natsClient.NatsPublish(KubearmorPayload)
	}

	if config.Stan.HostPort != "" && config.Stan.ClusterID != "" && config.Stan.ClientID != "" {
		go stanClient.StanPublish(KubearmorPayload)
	}

	if config.AWS.Lambda.FunctionName != "" {
		go awsClient.WatchInvokeLambdaAlerts()
		go awsClient.WatchInvokeLambdaLogs()
	}

	if config.AWS.SQS.URL != "" {
		go awsClient.WatchSendMessageAlerts()
		go awsClient.WatchSendMessageLogs()
	}

	if config.AWS.SNS.TopicArn != "" {
		go awsClient.WatchPublishTopicAlerts()
		go awsClient.WatchPublishTopicLogs()
	}

	if config.AWS.CloudWatchLogs.LogGroup != "" {
		go awsClient.WatchSendCloudWatchLogAlerts()
		go awsClient.WatchSendCloudWatchLogLogs()
	}

	if config.AWS.S3.Bucket != "" {
		go awsClient.WatchUploadS3Alerts()
		go awsClient.WatchUploadS3Logs()
	}

	if config.AWS.SecurityLake.Bucket != "" && config.AWS.SecurityLake.Region != "" && config.AWS.SecurityLake.AccountID != "" && config.AWS.SecurityLake.Prefix != "" {
		go awsClient.WatchEnqueueSecurityLakeAlerts()
		go awsClient.WatchEnqueueSecurityLakeLogs()
	}

	if config.AWS.Kinesis.StreamName != "" {
		go awsClient.WatchPutRecordAlerts()
		go awsClient.WatchPutRecordLogs()
	}

	if config.SMTP.HostPort != "" {
		go smtpClient.WatchSendMailAlerts()
		go smtpClient.WatchSendMailLogs()
	}

	if config.Opsgenie.APIKey != "" {
		go opsgenieClient.OpsgeniePost(KubearmorPayload)
	}

	if config.Webhook.Address != "" {
		go webhookClient.WebhookPost(KubearmorPayload)
	}

	if config.NodeRed.Address != "" {
		go noderedClient.NodeRedPost(KubearmorPayload)
	}

	if config.CloudEvents.Address != "" {
		go cloudeventsClient.WatchCloudEventsSendAlerts()
		go cloudeventsClient.WatchCloudEventsSendLogs()

	}

	if config.Azure.EventHub.Name != "" {
		go azureClient.WatchEventHubPostlerts()
		go azureClient.WatchEventHubPostLogs()
	}

	if config.GCP.PubSub.ProjectID != "" && config.GCP.PubSub.Topic != "" {
		go gcpClient.GCPPublishTopic(KubearmorPayload)
	}

	if config.GCP.CloudFunctions.Name != "" {
		go gcpClient.GCPCallCloudFunction(KubearmorPayload)
	}

	if config.GCP.CloudRun.Endpoint != "" {
		go gcpCloudRunClient.CloudRunFunctionPost(KubearmorPayload)
	}

	if config.GCP.Storage.Bucket != "" {
		go gcpClient.UploadGCS(KubearmorPayload)
	}

	if config.Googlechat.WebhookURL != "" {
		go googleChatClient.GooglechatPost(KubearmorPayload)
	}

	if config.Kafka.HostPort != "" {
		go kafkaClient.WatchKafkaProduceAlerts()
		go kafkaClient.WatchKafkaProduceLogs()
	}

	if config.KafkaRest.Address != "" {
		go kafkaRestClient.KafkaRestPost(KubearmorPayload)
	}

	if config.Pagerduty.RoutingKey != "" {
		go pagerdutyClient.PagerdutyPost(KubearmorPayload)
	}

	if config.Kubeless.Namespace != "" && config.Kubeless.Function != "" {
		go kubelessClient.KubelessCall(KubearmorPayload)
	}

	if config.Openfaas.FunctionName != "" {
		go openfaasClient.OpenfaasCall(KubearmorPayload)
	}

	if config.Tekton.EventListener != "" {
		go tektonClient.TektonPost(KubearmorPayload)
	}
	//done
	if config.Rabbitmq.URL != "" && config.Rabbitmq.Queue != "" {
		go rabbitmqClient.WatchRabbitmqPublishAlerts()
	}

	if config.Wavefront.EndpointHost != "" && config.Wavefront.EndpointType != "" {
		go wavefrontClient.WavefrontPost(KubearmorPayload)
	}

	if config.Grafana.HostPort != "" {
		go grafanaClient.GrafanaPost(KubearmorPayload)
	}

	if config.GrafanaOnCall.WebhookURL != "" {
		go grafanaOnCallClient.WatchGrafanaOnCallPostAlerts()
		go grafanaOnCallClient.WatchGrafanaOnCallPostLogs()
	}

	if config.WebUI.URL != "" {
		go webUIClient.WebUIPost(KubearmorPayload)
	}

	if config.Fission.Function != "" {
		go fissionClient.FissionCall(KubearmorPayload)
	}

	if config.Yandex.S3.Bucket != "" {
		go yandexClient.UploadYandexS3(KubearmorPayload)
	}

	if config.Yandex.DataStreams.StreamName != "" {
		go yandexClient.UploadYandexDataStreams(KubearmorPayload)
	}

	fmt.Println("before Syslog -> ", config.Syslog.Host)
	if config.Syslog.Host != "" {
		fmt.Println("Syslog -> ", config.Syslog.Host)
		go syslogClient.WatchSyslogsAlerts()
		go syslogClient.WatchSyslogLogs()
	}

	if config.MQTT.Broker != "" {
		go mqttClient.MQTTPublish(KubearmorPayload)
	}

	if config.Zincsearch.HostPort != "" {
		go zincsearchClient.ZincsearchPost(KubearmorPayload)
	}

	if config.Gotify.HostPort != "" {
		go gotifyClient.GotifyPost(KubearmorPayload)
	}

	if config.Spyderbat.OrgUID != "" {
		go spyderbatClient.SpyderbatPost(KubearmorPayload)
	}

	if config.TimescaleDB.Host != "" {
		go timescaleDBClient.TimescaleDBPost(KubearmorPayload)
	}

	if config.Redis.Address != "" {
		go redisClient.RedisPost(KubearmorPayload)
	}

	if config.Telegram.ChatID != "" {
		go telegramClient.TelegramPost(KubearmorPayload)
	}

	if config.N8N.Address != "" {
		go n8nClient.N8NPost(KubearmorPayload)
	}

	if config.OpenObserve.HostPort != "" {
		go openObserveClient.OpenObservePost(KubearmorPayload)
	}
}
