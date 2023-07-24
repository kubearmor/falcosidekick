package main

import (
	"context"
	"net"
	"time"

	"github.com/falcosecurity/falcosidekick/outputs"
	pb "github.com/kubearmor/KubeArmor/protobuf"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

func GetLogsFromKubearmorRelay() {
	lc := outputs.Client{}
	var err error
	//get url
	url := GetKubearmorRelayURL()
	//connect to url
	conn := ConnectKubeArmorRelay(url, "32767")

	//create log and alert watcher
	client := pb.NewLogServiceClient(conn)
	req := pb.RequestMessage{}
	req.Filter = "all"

	lc.LogStream, err = client.WatchLogs(context.Background(), &req)
	if err != nil {
		log.Error().Msg("unable to stream systems logs: " + err.Error())
		return
	}

	lc.AlertStream, err = client.WatchAlerts(context.Background(), &req)
	if err != nil {
		log.Error().Msg("unable to stream systems logs: " + err.Error())
		return
	}

	//create a buffer to accept logs and alerts
	lc.WatchAlerts()
	lc.WatchLogs()

	//buffer to client specific buffer
	//send it from there
	go lc.AddAlertFromBuffChan()
	go lc.AddLogFromBuffChan()

}

func GetKubearmorRelayURL() string {
	client := ConnectK8sClient()
	if client == nil {
		return ""
	}

	// get kubearmor-relay pod from k8s api client
	svc, err := client.CoreV1().Services("").List(context.Background(), metav1.ListOptions{
		LabelSelector: "kubearmor-app=kubearmor-relay",
	})
	if err != nil {
		return ""
	}
	if svc == nil || len(svc.Items) == 0 {
		return ""
	}
	url := svc.Items[0].Name + "." + svc.Items[0].Namespace + ".svc.cluster.local"
	return url
}

func ConnectK8sClient() *kubernetes.Clientset {
	config, err := ctrl.GetConfig()
	if err != nil {

		return nil
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil
	}

	return clientset
}

func ConnectKubeArmorRelay(url string, port string) *grpc.ClientConn {
	addr := net.JoinHostPort(url, port)

	// Check for kubearmor-relay with 30s timeout
	ctx, cf1 := context.WithTimeout(context.Background(), time.Second*30)
	defer cf1()

	// Blocking grpc Dial: in case of a bad connection, fails with timeout
	conn, err := grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Error().Msg("Error connecting kubearmor relay: " + err.Error())
		return nil
	}

	log.Info().Msg("Connected to kubearmor relay " + addr)
	return conn
}
