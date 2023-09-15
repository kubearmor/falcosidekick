package outputs

import (
	"encoding/json"
	"testing"

	"github.com/kubearmor/sidekick/types"
)

func TestNewDatadogPayload(t *testing.T) {
	var kubearmorPayload types.KubearmorPayload
	err := json.Unmarshal([]byte(TestInput), &kubearmorPayload)
	if err != nil {
		t.Fatalf("Failed to unmarshal TestInput: %v", err)
	}

	result := newDatadogPayload(kubearmorPayload)

	// Define expected output based on TestInput
	expected := datadogPayload{
		Title:      "", // As it's not set in the newDatadogPayload function
		Text:       "", // As it's not set in the newDatadogPayload function
		SourceType: "kubearmor",
		AlertType:  Info, // Because EventType in TestInput is "MatchedPolicy"
		Tags: []string{
			"Timestamp:1631542902",
			"UpdatedTime:2023-09-13T15:35:02Z",
			"ClusterName:default",
			"Hostname:gke-kubearmor-prerelease-default-pool-6ad71e07-cd8r",
			"NamespaceName:wordpress-mysql",
			"PodName:wordpress-7c966b5d85-xvsrl",
			"Labels:app=wordpress",
			"ContainerID:80eead8fb840e9f3f3b1bea94bb202a798b92ad8ba4e0c92f52c4027dab98e73",
			"ContainerName:wordpress",
			"ContainerImage:docker.io/library/wordpress:4.8-apache@sha256:6216f64ab88fc51d311e38c7f69ca3f9aaba621492b4f1fa93ddf63093768845",
			"HostPPID:102114",
			"HostPID:102947",
			"PPID:203",
			"PID:217",
			"UID:1001",
			"ParentProcessName:/bin/bash",
			"Source:/bin/ls",
			"Operation:File",
			"Resource:/var",
			"Data:syscall=SYS_OPENAT fd=-100 flags=O_RDONLY|O_NONBLOCK|O_DIRECTORY|O_CLOEXEC",
			"Result:Passed",
			"PolicyName:DefaultPosture",
			"Severity:Medium",
			"Tags:Tag1,Tag2",
			"ATags:ATag1,ATag2",
			"Message:Policy Matched",
			"Enforcer:eBPF Monitor",
		},
	}

	// Compare actual and expected results
	if result.SourceType != expected.SourceType {
		t.Errorf("Expected SourceType to be '%s', but got '%s'", expected.SourceType, result.SourceType)
	}
	if result.AlertType != expected.AlertType {
		t.Errorf("Expected AlertType to be '%s', but got '%s'", expected.AlertType, result.AlertType)
	}

	for _, tag := range expected.Tags {
		if !contains(result.Tags, tag) {
			t.Errorf("Expected tag '%s' to be present in result, but it wasn't", tag)
		}
	}
}

// Utility function to check if a slice contains a specific string
func contains(slice []string, item string) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}
