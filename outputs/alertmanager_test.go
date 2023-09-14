package outputs

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kubearmor/sidekick/types"
)

const defaultThresholds = `[{"priority":"critical", "value":10000}, {"priority":"critical", "value":1000}, {"priority":"critical", "value":100} ,{"priority":"warning", "value":10}, {"priority":"warning", "value":1}]`

func TestNewAlertmanagerPayloadO(t *testing.T) {
	fmt.Println("Running Alertmaneger tets")
	expectedOutput := `
	[
		{
			"Labels": {
				"ATags": "ATag1,ATag2",
				"ClusterName": "default",
				"ContainerID": "80eead8fb840e9f3f3b1bea94bb202a798b92ad8ba4e0c92f52c4027dab98e73",
				"ContainerImage": "docker.io/library/wordpress:4.8-apache@sha256:6216f64ab88fc51d311e38c7f69ca3f9aaba621492b4f1fa93ddf63093768845",
				"ContainerName": "wordpress",
				"Data": "syscall=SYS_OPENAT fd=-100 flags=O_RDONLY|O_NONBLOCK|O_DIRECTORY|O_CLOEXEC",
				"Enforcer": "eBPF Monitor",
				"HostPID": "102947",
				"HostPPID": "102114",
				"Hostname": "gke-kubearmor-prerelease-default-pool-6ad71e07-cd8r",
				"Labels": "app=wordpress",
				"Message": "Policy Matched",
				"NamespaceName": "wordpress-mysql",
				"Operation": "File",
				"OwnerName": "wordpress",
				"OwnerNamespace": "wordpress-mysql",
				"OwnerRef": "Deployment",
				"PID": "217",
				"PPID": "203",
				"ParentProcessName": "/bin/bash",
				"PodName": "wordpress-7c966b5d85-xvsrl",
				"PolicyName": "DefaultPosture",
				"ProcessName": "/bin/ls",
				"Resource": "/var",
				"Result": "Passed",
				"Severity": "Medium",
				"Source": "/bin/ls",
				"Tags": "Tag1,Tag2",
				"Timestamp": "1631542902",
				"UID": "1001",
				"UpdatedTime": "2023-09-13T15:35:02Z",
				"source": "Kubearmor"
			},
			"Annotations": null,
			"EndsAt": "0001-01-01T00:00:00Z"
		}
	]
	`

	var f types.KubearmorPayload
	d := json.NewDecoder(strings.NewReader(TestInput))
	d.UseNumber()
	err := d.Decode(&f) //have to decode it the way newkubearmorpayload does
	require.Nil(t, err)

	config := &types.Configuration{
		Log: false,
	}
	json.Unmarshal([]byte(defaultThresholds), &config.Alertmanager.DropEventThresholdsList)

	s, err := json.Marshal(newAlertmanagerPayload(f, config))
	require.Nil(t, err)

	var o1, o2 []alertmanagerPayload
	require.Nil(t, json.Unmarshal([]byte(expectedOutput), &o1))
	require.Nil(t, json.Unmarshal(s, &o2))
	fmt.Println(o2)
	require.Equal(t, o1, o2)
}
