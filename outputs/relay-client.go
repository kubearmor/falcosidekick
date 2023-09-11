package outputs

import (
	"fmt"
	"sync"
	"time"

	pb "github.com/kubearmor/KubeArmor/protobuf"
	"github.com/kubearmor/sidekick/types"
)

// Running bool
var AlertRunning bool
var LogRunning bool

// LogBufferChannel store incoming data from log stream in buffer
var LogBufferChannel chan *pb.Log

// AlertBufferChannel store incoming data from msg stream in buffer
var AlertBufferChannel chan *pb.Alert

// AlertStruct Structure
type AlertStruct struct {
	Broadcast chan types.KubearmorPayload
}

// AlertStructs Map
var AlertStructs map[string]AlertStruct

// AlertLock Lock
var AlertLock *sync.RWMutex

// LogStruct Structure
type LogStruct struct {
	Filter    string
	Broadcast chan types.KubearmorPayload
}

var LogLock *sync.RWMutex

// LogStructs Map
var LogStructs map[string]LogStruct

func Initvariable(logrunning bool) {

	AlertRunning = true
	LogRunning = logrunning

	//initial buffer struct
	LogBufferChannel = make(chan *pb.Log, 10000)
	AlertBufferChannel = make(chan *pb.Alert, 1000)

	// initialize alert structs
	AlertStructs = make(map[string]AlertStruct)
	AlertLock = &sync.RWMutex{}

	// initialize log structs
	LogStructs = make(map[string]LogStruct)
	LogLock = &sync.RWMutex{}

}

// DestroyClient Function
func (c *Client) DestroyClient() error {
	c.Running = false
	LogRunning = false
	AlertRunning = false

	err := c.Conn.Close()

	return err
}

func (c *Client) WatchAlerts() error {

	defer c.WgServer.Done()

	var err error

	for {
		var res *pb.Alert

		if res, err = c.AlertStream.Recv(); err != nil {
			break
		}

		select {
		case AlertBufferChannel <- res:
		default:
		}

	}

	return nil
}

// AddAlertFromBuffChan Adds ALert from AlertBufferChannel into AlertStructs
func (c *Client) AddAlertFromBuffChan() {
	for AlertRunning {
		select {
		case res := <-AlertBufferChannel:

			alert := types.KubearmorPayload{}

			alert.Timestamp = res.GetTimestamp()
			alert.UpdatedTime = res.GetUpdatedTime()
			alert.ClusterName = res.GetClusterName()
			alert.Hostname = res.GetHostName()
			alert.EventType = "Alert"
			alert.OutputFields = make(map[string]interface{})

			alert.OutputFields["OwnerRef"] = res.GetOwner().GetRef()
			alert.OutputFields["OwnerName"] = res.GetOwner().GetName()
			alert.OutputFields["OwnerNamespace"] = res.GetOwner().GetNamespace()

			alert.OutputFields["Timestamp"] = fmt.Sprint(res.GetTimestamp())
			alert.OutputFields["UpdatedTime"] = res.GetUpdatedTime()
			alert.OutputFields["ClusterName"] = res.GetClusterName()
			alert.OutputFields["Hostname"] = res.GetHostName()
			alert.OutputFields["NamespaceName"] = res.GetNamespaceName()
			alert.OutputFields["PodName"] = res.GetPodName()
			alert.OutputFields["Labels"] = res.GetLabels()
			alert.OutputFields["ContainerID"] = res.GetContainerID()
			alert.OutputFields["ContainerName"] = res.GetContainerName()
			alert.OutputFields["ContainerImage"] = res.GetContainerImage()
			alert.OutputFields["HostPPID"] = res.GetHostPPID()
			alert.OutputFields["HostPID"] = res.GetHostPID()
			alert.OutputFields["PPID"] = res.GetPPID()
			alert.OutputFields["PID"] = res.GetPID()
			alert.OutputFields["UID"] = res.GetUID()
			alert.OutputFields["ParentProcessName"] = res.GetParentProcessName()
			alert.OutputFields["ProcessName"] = res.GetProcessName()
			alert.OutputFields["Source"] = res.GetSource()
			alert.OutputFields["Operation"] = res.GetOperation()
			alert.OutputFields["Resource"] = res.GetResource()
			alert.OutputFields["Data"] = res.GetData()
			alert.OutputFields["Result"] = res.GetResult()
			alert.OutputFields["PolicyName"] = res.GetPolicyName()
			alert.OutputFields["Severity"] = res.GetSeverity()
			alert.OutputFields["Tags"] = res.GetTags()
			alert.OutputFields["ATags"] = res.GetATags()
			alert.OutputFields["Message"] = res.GetMessage()
			alert.OutputFields["Enforcer"] = res.GetEnforcer()

			AlertLock.RLock()
			for uid := range AlertStructs {
				select {
				case AlertStructs[uid].Broadcast <- (alert):
				default:
				}
			}
			AlertLock.RUnlock()

		default:
			time.Sleep(time.Millisecond * 10)
		}

	}
}

// WatchLogs Function
func (c *Client) WatchLogs() error {

	defer c.WgServer.Done()

	var err error

	for LogRunning {
		var res *pb.Log

		if res, err = c.LogStream.Recv(); err != nil {
			fmt.Println("Error streamsing logs err: ", err)
			break
		}

		select {
		case LogBufferChannel <- res:
		default:
			//not able to add it to Log buffer
		}
	}

	return nil
}

// AddLogFromBuffChan Adds Log from LogBufferChannel into LogStructs
func (c *Client) AddLogFromBuffChan() {

	for LogRunning {
		select {
		case res := <-LogBufferChannel:
			log := types.KubearmorPayload{}
			log.Timestamp = res.GetTimestamp()
			log.UpdatedTime = res.GetUpdatedTime()
			log.ClusterName = res.GetClusterName()
			log.Hostname = res.GetHostName()
			log.EventType = "Log"

			// Create new Podowner struct
			log.OutputFields = make(map[string]interface{})
			log.OutputFields["OwnerRef"] = res.GetOwner().GetRef()
			log.OutputFields["OwnerName"] = res.GetOwner().GetName()
			log.OutputFields["OwnerNamespace"] = res.GetOwner().GetNamespace()
			log.OutputFields["Timestamp"] = res.GetTimestamp()
			log.OutputFields["UpdatedTime"] = res.GetUpdatedTime()
			log.OutputFields["ClusterName"] = res.GetClusterName()
			log.OutputFields["Hostname"] = res.GetHostName()
			log.OutputFields["NamespaceName"] = res.GetNamespaceName()
			log.OutputFields["PodName"] = res.GetPodName()
			log.OutputFields["Labels"] = res.GetLabels()
			log.OutputFields["ContainerID"] = res.GetContainerID()
			log.OutputFields["ContainerName"] = res.GetContainerName()
			log.OutputFields["ContainerImage"] = res.GetContainerImage()
			log.OutputFields["HostPPID"] = res.GetHostPPID()
			log.OutputFields["HostPID"] = res.GetHostPID()
			log.OutputFields["PPID"] = res.GetPPID()
			log.OutputFields["PID"] = res.GetPID()
			log.OutputFields["UID"] = res.GetUID()
			log.OutputFields["ParentProcessName"] = res.GetParentProcessName()
			log.OutputFields["ProcessName"] = res.GetProcessName()
			log.OutputFields["Source"] = res.GetSource()
			log.OutputFields["Operation"] = res.GetOperation()
			log.OutputFields["Resource"] = res.GetResource()
			log.OutputFields["Data"] = res.GetData()
			log.OutputFields["Result"] = res.GetResult()

			for uid := range LogStructs {
				select {
				case LogStructs[uid].Broadcast <- (log):
				default:
				}
			}
		default:
			time.Sleep(time.Millisecond * 10)
		}

	}
}

func addAlertStruct(uid string, conn chan types.KubearmorPayload) {
	AlertLock.Lock()
	defer AlertLock.Unlock()

	alertStruct := AlertStruct{}
	alertStruct.Broadcast = conn

	AlertStructs[uid] = alertStruct

	fmt.Println("Added a new client (" + uid + ") for WatchAlerts")
}
func removeAlertStruct(uid string) {
	AlertLock.Lock()
	defer AlertLock.Unlock()

	delete(AlertStructs, uid)
	fmt.Println("Deleted a new client (" + uid + ") for WatchAlerts")

}

func addLogStruct(uid string, conn chan types.KubearmorPayload) {
	LogLock.Lock()
	defer LogLock.Unlock()

	logStruct := LogStruct{}
	logStruct.Broadcast = conn

	LogStructs[uid] = logStruct
	fmt.Println("Added a new client (" + uid + ") for WatchLogss")

}
func removeLogStruct(uid string) {
	LogLock.Lock()
	defer LogLock.Unlock()

	delete(LogStructs, uid)
	fmt.Println("Deleted a new client (" + uid + ") for WatchLogs")

}
